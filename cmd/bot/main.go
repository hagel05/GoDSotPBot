// Command bot runs the DSotP fantasy-football Slack bot as a long-lived HTTP
// server. It replaces the original AWS Lambda entrypoint so the same code can
// run in a Docker container on TrueNAS.
//
// Subcommands:
//
//	bot                              run the HTTP server (default)
//	bot record-champion <year> <player>   seed/correct a champion in the DB
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hagel05/go-dsotp-bot/internal/commands"
	"github.com/hagel05/go-dsotp-bot/internal/config"
	"github.com/hagel05/go-dsotp-bot/internal/league"
	"github.com/hagel05/go-dsotp-bot/internal/slack"
	"github.com/hagel05/go-dsotp-bot/internal/store"
	"github.com/hagel05/go-dsotp-bot/internal/yahoo"
)

// maxBodyBytes caps how much of a request body we read. Slash-command payloads
// are tiny; this guards against a malicious oversized body.
const maxBodyBytes = 64 * 1024

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "record-champion":
			runRecordChampion(os.Args[2:])
			return
		case "yahoo-auth":
			runYahooAuth(os.Args[2:])
			return
		case "yahoo-get":
			runYahooGet(os.Args[2:])
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
			os.Exit(2)
		}
	}
	runServer()
}

func runServer() {
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

	db, err := store.Open(context.Background(), cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "path", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()

	server := &server{
		verifier:  slack.NewVerifier(cfg.SlackSigningSecret),
		registry:  commands.NewRegistry(),
		provider:  provider,
		champions: db,
		admins:    cfg.AdminUserIDs,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("POST /slack/commands", server.handleSlashCommand)

	if cfg.YahooEnabled() {
		oauthCfg := yahoo.OAuthConfig(cfg.YahooClientID, cfg.YahooClientSecret, cfg.YahooRedirectURI)
		authSrv := yahoo.NewAuthServer(oauthCfg, db, cfg.YahooAuthUser, cfg.YahooAuthPass)
		mux.HandleFunc("GET /yahoo/login", authSrv.LoginHandler)
		mux.HandleFunc("GET /yahoo/callback", authSrv.CallbackHandler)
		if authSrv.BasicAuthConfigured() {
			slog.Info("Yahoo browser auth enabled at /yahoo/login (Basic Auth protected)")
		} else {
			slog.Warn("Yahoo auth endpoints served WITHOUT app Basic Auth; set YAHOO_AUTH_USER/YAHOO_AUTH_PASS or gate /yahoo/ at your reverse proxy")
		}
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	runWithGracefulShutdown(httpServer)
}

// runRecordChampion seeds or corrects a champion from the command line, e.g.
// `bot record-champion 2024 Hagel "Vamonos Pest"`. The player is a single token;
// anything after it is the (optional) team name. Used to bootstrap history at
// deploy time.
func runRecordChampion(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, `usage: bot record-champion <year> <player> [team name]`)
		os.Exit(2)
	}
	year, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid year %q\n", args[0])
		os.Exit(2)
	}
	player := args[1]
	teamName := strings.Join(args[2:], " ")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = config.DefaultDBPath
	}

	ctx := context.Background()
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database %q: %v\n", dbPath, err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Record(ctx, store.Champion{Year: year, TeamName: teamName, Player: player}); err != nil {
		fmt.Fprintf(os.Stderr, "record champion: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Recorded %d champion: %s (team %q) in %s.\n", year, player, teamName, dbPath)
}

// runYahooAuth performs the one-time Yahoo authorization. With no argument it
// prints the URL to approve; run again as `bot yahoo-auth <code>` with the code
// Yahoo redirects back to exchange it for a long-lived refresh token.
func runYahooAuth(args []string) {
	clientID := os.Getenv("YAHOO_CLIENT_ID")
	clientSecret := os.Getenv("YAHOO_CLIENT_SECRET")
	redirectURI := os.Getenv("YAHOO_REDIRECT_URI")
	if clientID == "" || clientSecret == "" || redirectURI == "" {
		fmt.Fprintln(os.Stderr, "set YAHOO_CLIENT_ID, YAHOO_CLIENT_SECRET, and YAHOO_REDIRECT_URI first")
		os.Exit(2)
	}
	cfg := yahoo.OAuthConfig(clientID, clientSecret, redirectURI)

	if len(args) == 0 {
		fmt.Println("1. Open this URL in a browser and approve access:")
		fmt.Println()
		fmt.Println("   " + yahoo.AuthCodeURL(cfg, "dsotp"))
		fmt.Println()
		fmt.Printf("2. Yahoo redirects to %s?code=...  — copy the 'code' value from the URL.\n", redirectURI)
		fmt.Println("3. Run:  bot yahoo-auth <code>")
		return
	}

	ctx := context.Background()
	token, err := yahoo.Exchange(ctx, cfg, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "exchange authorization code: %v\n", err)
		os.Exit(1)
	}
	if token.RefreshToken == "" {
		fmt.Fprintln(os.Stderr, "Yahoo returned no refresh token")
		os.Exit(1)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = config.DefaultDBPath
	}
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database %q: %v\n", dbPath, err)
		os.Exit(1)
	}
	defer db.Close()
	if err := yahoo.SaveToken(ctx, db, token); err != nil {
		fmt.Fprintf(os.Stderr, "save token: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Success — Yahoo refresh token saved to the database (%s). No env var needed;\n", dbPath)
	fmt.Println("it persists across reboots/reinstalls and self-updates if Yahoo ever rotates it.")
}

// runYahooGet fetches a Yahoo Fantasy API resource with the stored token and
// prints the raw JSON. A debugging aid for capturing real response shapes, e.g.
// `bot yahoo-get league/nfl.l.12345/standings`.
func runYahooGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: bot yahoo-get <path>  (e.g. league/nfl.l.12345/standings)")
		os.Exit(2)
	}
	clientID := os.Getenv("YAHOO_CLIENT_ID")
	clientSecret := os.Getenv("YAHOO_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "set YAHOO_CLIENT_ID and YAHOO_CLIENT_SECRET first")
		os.Exit(2)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = config.DefaultDBPath
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database %q: %v\n", dbPath, err)
		os.Exit(1)
	}
	defer db.Close()

	// RedirectURI isn't needed for API calls (only for the auth code flow).
	cfg := yahoo.OAuthConfig(clientID, clientSecret, os.Getenv("YAHOO_REDIRECT_URI"))
	httpClient, err := yahoo.NewHTTPClient(ctx, cfg, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	status, body, err := yahoo.Fetch(ctx, httpClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "HTTP %d\n", status)

	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(body)) // not JSON; print raw
	}
}

// server holds the dependencies each request handler needs.
type server struct {
	verifier  *slack.Verifier
	registry  *commands.Registry
	provider  league.Provider
	champions store.ChampionStore
	admins    map[string]bool
}

func (srv *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func (srv *server) handleSlashCommand(w http.ResponseWriter, r *http.Request) {
	// Slack signs an HMAC over the raw request body, so we must read the body
	// before we can verify it — there is no way to authenticate first. Two things
	// keep this safe: the read is bounded by maxBodyBytes (an oversized body is
	// truncated, which then fails verification), and the bytes are treated as
	// opaque until Verify passes. Nothing below parses or trusts the body until
	// after the signature check.
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		slog.Warn("failed to read request body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")
	if err := srv.verifier.Verify(timestamp, signature, rawBody); err != nil {
		slog.Warn("slack signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Verified: now it is safe to parse and act on the body.
	values, err := url.ParseQuery(string(rawBody))
	if err != nil {
		slog.Warn("failed to parse slash command body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	command := slack.ParseSlashCommand(values)

	leagueData, err := srv.provider.Load(r.Context())
	if err != nil {
		slog.Error("failed to load league data", "error", err)
		writeJSON(w, slack.Ephemeral("Sorry, I couldn't read the league data right now."))
		return
	}

	env := &commands.Env{
		League:    leagueData,
		Champions: srv.champions,
		Admins:    srv.admins,
	}
	response := srv.registry.Dispatch(r.Context(), command, env)
	slog.Info("handled command", "command", command.Command, "user", command.UserName)
	writeJSON(w, response)
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
