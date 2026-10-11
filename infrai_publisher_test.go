package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPublisherRetriesRateLimitWithSameIdempotencyKey(t *testing.T) {
	requests := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization header missing")
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": map[string]string{"message": "retry later"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]string{"status": "published"}, "metadata": map[string]string{}})
	}))
	defer server.Close()

	publisher := NewPublisher("test-key")
	publisher.URL = server.URL
	publisher.Sleep = func(context.Context, time.Duration) error { return nil }
	err := publisher.Publish(context.Background(), PublishRequest{
		Channel: "game-backend-members",
		Event:   "player_asset.created",
		Data:    json.RawMessage(`{"asset_id":"asset-104"}`),
	}, "announcement-fixed")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if requests != 2 || keys[0] != "announcement-fixed" || keys[1] != keys[0] {
		t.Fatalf("requests = %d, keys = %v", requests, keys)
	}
}
