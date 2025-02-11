package slackwebhookdispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/slack-go/slack"
)

type Handler struct {
	rules  []Rule
	router *mux.Router
	client *http.Client
}

func New(config *Config) *Handler {
	h := &Handler{
		rules: config.Rules,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
	h.setup()
	return h
}

func (h *Handler) SetHTTPClient(client *http.Client) {
	h.client = client
}

func (h *Handler) setup() {
	h.router = mux.NewRouter()
	h.router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeStr := r.URL.Path
			if route := mux.CurrentRoute(r); route != nil {
				routeStr = route.GetName()
			}
			slog.InfoContext(r.Context(), "accept request dispache", slog.Group("request", "method", r.Method, "path", routeStr))
			h.ServeHTTP(w, r)
		})
	})
	h.router.HandleFunc("/services/{team_id:T[A-Za-z0-9]+}/{bot_id:B[A-Za-z0-9]+}/{token:[A-Za-z0-9]+}", h.handleIncomingWebhook).Methods(http.MethodPost).Name("/services/{team_id}/{bot_id}/{token}")
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "handle request but not found", slog.Group("request", "method", r.Method, "path", r.URL.Path))
		http.Redirect(w, r, "https://api.slack.com/", http.StatusFound)
	})
	h.router.NotFoundHandler = fallbackHandler
	h.router.MethodNotAllowedHandler = fallbackHandler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	muxVars := mux.Vars(r)
	teamID := muxVars["team_id"]
	botID := muxVars["bot_id"]
	token := muxVars["token"]
	defer r.Body.Close()
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid_payload")
		return
	}
	var payload slack.Msg
	if err := json.NewDecoder(bytes.NewReader(bs)).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid_payload")
		return
	}
	ctx := r.Context()
	variables := &CELVariables{
		Payload: &payload,
		TeamID:  teamID,
		BotID:   botID,
		Token:   token,
	}
	destination := fmt.Sprintf("https://hooks.slack.com/services/%s/%s/%s", teamID, botID, token)
	isDefault := true
	for _, rule := range h.rules {
		matched, err := Evalute(ctx, rule.prog, variables)
		if err != nil {
			slog.WarnContext(ctx, "failed to evaluate rule", "details", err.Error())
			continue
		}
		if matched {
			slog.InfoContext(ctx, "matched rule", "rule_name", rule.Name)
			destination = rule.Destination
			isDefault = false
			break
		}
	}
	if isDefault {
		slog.InfoContext(ctx, "no rule matched, use default destination")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destination, bytes.NewReader(bs))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create request", "details", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal_server_error")
		return
	}
	for k, v := range r.Header {
		req.Header[k] = v
	}
	resp, err := h.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to send request", "details", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal_server_error")
		return
	}
	defer resp.Body.Close()
	respHeader := w.Header()
	for k, v := range resp.Header {
		respHeader[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
