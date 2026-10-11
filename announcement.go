package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const backendAudience = "all_backend_members"

var supportedKinds = map[string]string{
	"player_asset":     "player_asset.created",
	"live_event":       "live_event.announced",
	"moderation_queue": "moderation_queue.updated",
}

type Announcement struct {
	ID        string          `json:"id"`
	Audience  string          `json:"audience"`
	Kind      string          `json:"kind"`
	AccountID string          `json:"account_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type PublishRequest struct {
	Channel   string          `json:"channel"`
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	AccountID string          `json:"account_id,omitempty"`
}

func (a Announcement) publishRequest(channel string) (PublishRequest, string, error) {
	if strings.TrimSpace(a.ID) == "" {
		return PublishRequest{}, "", errors.New("id is required")
	}
	if a.Audience != backendAudience {
		return PublishRequest{}, "", fmt.Errorf("audience must be %q", backendAudience)
	}
	event, ok := supportedKinds[a.Kind]
	if !ok {
		return PublishRequest{}, "", fmt.Errorf("unsupported kind %q", a.Kind)
	}
	if len(a.Payload) == 0 || !json.Valid(a.Payload) {
		return PublishRequest{}, "", errors.New("payload must be valid JSON")
	}
	sum := sha256.Sum256([]byte(channel + "\x00" + a.ID))
	return PublishRequest{
		Channel:   channel,
		Event:     event,
		Data:      a.Payload,
		AccountID: a.AccountID,
	}, "announcement-" + hex.EncodeToString(sum[:]), nil
}
