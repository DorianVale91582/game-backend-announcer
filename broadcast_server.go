package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type announcementPublisher interface {
	Publish(ctx context.Context, input PublishRequest, idempotencyKey string) error
}

type broadcastServer struct {
	publisher announcementPublisher
	channel   string
}

func (s broadcastServer) handleAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method must be POST"})
		return
	}
	defer r.Body.Close()
	var input Announcement
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid announcement JSON"})
		return
	}
	publish, key, err := input.publishRequest(s.channel)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if err := s.publisher.Publish(r.Context(), publish, key); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Message})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "announcement publish failed"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"announcement_id": input.ID,
		"status":          "published",
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
