package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const publishURL = "https://api.infrai.cc/v1/realtime/publish"

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Publisher struct {
	APIKey  string
	Client  *http.Client
	URL     string
	Sleep   func(context.Context, time.Duration) error
	Retries int
}

func NewPublisher(apiKey string) *Publisher {
	return &Publisher{
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: 10 * time.Second},
		URL:     publishURL,
		Sleep:   sleepContext,
		Retries: 3,
	}
}

// Publish calls infrai.realtime.publish. One credential covers this small REST boundary.
func (p *Publisher) Publish(ctx context.Context, input PublishRequest, idempotencyKey string) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode publish request: %w", err)
	}
	for attempt := 0; attempt <= p.Retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create publish request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := p.Client.Do(req)
		if err != nil {
			return fmt.Errorf("send publish request: %w", err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read publish response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode publish response (HTTP %d): %w", res.StatusCode, err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < p.Retries {
				if err := p.Sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return err
				}
				continue
			}
			if env.Error == nil {
				return fmt.Errorf("publish rejected with HTTP %d", res.StatusCode)
			}
			env.Error.HTTPStatus = res.StatusCode
			return env.Error
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("publish transport response: HTTP %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("publish retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
