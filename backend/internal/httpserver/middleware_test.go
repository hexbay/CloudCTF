package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSensitiveRateLimiterBoundsTrackedClients(t *testing.T) {
	called := 0
	handler := newSensitiveRateLimiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}), 2)

	for index, remoteAddr := range []string{"192.0.2.1:1001", "192.0.2.2:1002", "192.0.2.3:1003"} {
		request := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		request.RemoteAddr = remoteAddr
		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(responseRecorder, request)

		wantStatus := http.StatusNoContent
		if index == 2 {
			wantStatus = http.StatusTooManyRequests
		}
		if responseRecorder.Code != wantStatus {
			t.Fatalf("request %d returned %d, want %d", index, responseRecorder.Code, wantStatus)
		}
	}

	if called != 2 {
		t.Fatalf("downstream handler called %d times, want 2", called)
	}
}

func TestPruneExpiredRateLimitEntries(t *testing.T) {
	now := time.Now()
	entries := map[string]rateLimitEntry{
		"expired": {count: 1, reset: now.Add(-time.Second)},
		"current": {count: 1, reset: now.Add(time.Second)},
	}

	pruneExpiredRateLimitEntries(entries, now)

	if _, exists := entries["expired"]; exists {
		t.Fatal("expired entry was not removed")
	}
	if _, exists := entries["current"]; !exists {
		t.Fatal("current entry was removed")
	}
}
