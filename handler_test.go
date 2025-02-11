package slackwebhookdispatcher

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestHandler(t *testing.T) {
	t.Setenv("SLACK_WEBHOOK_URL_FOR_SERVICE1", "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX")
	t.Setenv("SLACK_WEBHOOK_URL_FOR_SERVICE2", "https://hooks.slack.com/services/T00000000/B00000000/YYYYYYYYYYYYYYYYYYYYYYYY")

	cfg, err := LoadConfig(context.Background(), "testdata/config.jsonnet")
	require.NoError(t, err)
	h := New(cfg)
	require.NotNil(t, h)

	var mu sync.Mutex
	accessLogs := make([]string, 0)
	destinationHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		accessLogs = append(accessLogs, fmt.Sprintf("%s %s", r.Method, r.URL.String()))
		mu.Unlock()
	})
	h.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			destinationHandler.ServeHTTP(w, r)
			return w.Result(), nil
		}),
	})
	t.Run("match rule1", func(t *testing.T) {
		mu.Lock()
		accessLogs = make([]string, 0)
		mu.Unlock()

		payload := `{
			"username":"Test",
			"attachments":[
				{
					"color":"#ff3e4b",
					"title":"[service1] [development] test exception",
					"text":"Occurred at 2025-01-01T25:59:59Z"
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/services/T00000000/B00000000/ZZZZZZZZZZZZZZZZZZZZZZZZ", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, accessLogs, 1)
		require.Equal(t, "POST https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX", accessLogs[0])
	})
	t.Run("match rule2", func(t *testing.T) {
		mu.Lock()
		accessLogs = make([]string, 0)
		mu.Unlock()

		payload := `{
			"username":"Test",
			"attachments":[
				{
					"color":"#ff3e4b",
					"title":"[service2] [development] test exception",
					"text":"Occurred at 2025-01-01T25:59:59Z"
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/services/T00000000/B00000000/ZZZZZZZZZZZZZZZZZZZZZZZZ", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, accessLogs, 1)
		require.Equal(t, "POST https://hooks.slack.com/services/T00000000/B00000000/YYYYYYYYYYYYYYYYYYYYYYYY", accessLogs[0])
	})
	t.Run("not match rule", func(t *testing.T) {
		mu.Lock()
		accessLogs = make([]string, 0)
		mu.Unlock()

		payload := `{
			"username":"Test",
			"attachments":[
				{
					"color":"#ff3e4b",
					"title":"[service3] [development] test exception",
					"text":"Occurred at 2025-01-01T25:59:59Z"
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/services/T00000000/B00000000/ZZZZZZZZZZZZZZZZZZZZZZZZ", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, accessLogs, 1)
		require.Equal(t, "POST https://hooks.slack.com/services/T00000000/B00000000/ZZZZZZZZZZZZZZZZZZZZZZZZ", accessLogs[0])
	})
}
