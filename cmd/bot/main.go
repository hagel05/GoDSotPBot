// Command bot runs the DSotP fantasy-football Slack bot as a long-lived HTTP
// server. It replaces the original AWS Lambda entrypoint so the same code can
// run in a Docker container on TrueNAS.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hagel05/go-dsotp-bot/internal/commands"
	"github.com/hagel05/go-dsotp-bot/internal/config"
	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
)

// maxBodyBytes caps how much of a request body we read. Slash-command payloads
// are tiny; this guards against a malicious oversized body.
const maxBodyBytes = 64 * 1024

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Logger isn't configured yet; use the default before exiting.
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	setupLogging(cfg.LogLevel)

	provider, err := league.NewStaticProvider(cfg.LeagueConfigPath)
	if err != nil {
		slog.Error("failed to load league configuration", "error", err)
		os.Exit(1)
	}

	server := &server{
		verifier: slack.NewVerifier(cfg.SlackSigningSecret),
		registry: commands.NewRegistry(),
		provider: provider,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("POST /slack/commands", server.handleSlashCommand)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	runWithGracefulShutdown(httpServer)
}

// server holds the dependencies each request handler needs.
type server struct {
	verifier *slack.Verifier
	registry *commands.Registry
	provider league.Provider
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func (s *server) handleSlashCommand(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		slog.Warn("failed to read request body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify the request really came from Slack before trusting anything in it.
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")
	if err := s.verifier.Verify(timestamp, signature, body); err != nil {
		slog.Warn("slack signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		slog.Warn("failed to parse slash command body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	cmd := slack.ParseSlashCommand(values)

	l, err := s.provider.Load(r.Context())
	if err != nil {
		slog.Error("failed to load league data", "error", err)
		writeJSON(w, slack.Ephemeral("Sorry, I couldn't read the league data right now."))
		return
	}

	resp := s.registry.Dispatch(cmd, l)
	slog.Info("handled command", "command", cmd.Command, "user", cmd.UserName)
	writeJSON(w, resp)
}

// writeJSON serializes a Slack response as JSON with the right content type.
func writeJSON(w http.ResponseWriter, resp slack.Response) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func setupLogging(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

func runWithGracefulShutdown(srv *http.Server) {
	go func() {
		slog.Info("starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for SIGINT/SIGTERM (docker stop sends SIGTERM) then drain in-flight
	// requests before exiting.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
