package main

import (
	"encoding/json"
	"testing"
)

func TestAnnouncementPublishDecision(t *testing.T) {
	tests := []struct {
		name      string
		input     Announcement
		wantEvent string
		wantErr   bool
	}{
		{
			name:      "player asset reaches every backend member",
			input:     Announcement{ID: "asset-104", Audience: backendAudience, Kind: "player_asset", Payload: json.RawMessage(`{"asset_id":"asset-104","state":"ready"}`)},
			wantEvent: "player_asset.created",
		},
		{
			name:      "live event reaches every backend member",
			input:     Announcement{ID: "event-22", Audience: backendAudience, Kind: "live_event", Payload: json.RawMessage(`{"event_id":"event-22","starts_at":"2030-01-02T15:00:00Z"}`)},
			wantEvent: "live_event.announced",
		},
		{
			name:      "moderation queue reaches every backend member",
			input:     Announcement{ID: "queue-9", Audience: backendAudience, Kind: "moderation_queue", Payload: json.RawMessage(`{"queue":"ugc-review","pending":12}`)},
			wantEvent: "moderation_queue.updated",
		},
		{
			name:    "narrow audience is rejected",
			input:   Announcement{ID: "asset-105", Audience: "one_server", Kind: "player_asset", Payload: json.RawMessage(`{"asset_id":"asset-105"}`)},
			wantErr: true,
		},
		{
			name:    "unknown domain kind is rejected",
			input:   Announcement{ID: "other-1", Audience: backendAudience, Kind: "chat_message", Payload: json.RawMessage(`{"text":"hello"}`)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, key, err := tt.input.publishRequest("game-backend-members")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("publishRequest() error = %v", err)
			}
			if got.Event != tt.wantEvent || got.Channel != "game-backend-members" {
				t.Fatalf("publish request = %#v", got)
			}
			if key == "" {
				t.Fatal("expected stable idempotency key")
			}
		})
	}
}
