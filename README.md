# DSotP Bot

A Slack slash-command bot for our fantasy football league. It answers commands
like `/draftinfo` and `/draftorder` with data read from a YAML file you edit once
a year — no code changes required.

Originally an AWS Lambda; it now runs as a single Docker container (image on
GHCR) intended for TrueNAS or any Docker host.

## Commands

| Command                          | Response |
|----------------------------------|----------|
| `/draftinfo`                     | Draft date, time, and location |
| `/draftorder`                    | Numbered draft order |
| `/punishment`                    | League punishments |
| `/remotedraft`                   | Link to the remote draft room |
| `/champion [year]`               | Reigning champion (team + player), or a given year's |
| `/champions`                     | Full championship history, newest first |
| `/titles [player]`               | A player's title count + years, or the all-time leaderboard |
| `/recordchampion <year> <player> [team]` | Record/correct a champion — **admins only** (see `ADMIN_USER_IDS`) |

Draft/punishment text comes from `league.yaml`; championship history comes from
the SQLite database (see below).

## Configuration

Two kinds of configuration, deliberately separated:

**League content — a mounted YAML file.** This is what you edit yearly. Copy
[`config/league.example.yaml`](config/league.example.yaml) to your real
`league.yaml`, edit the draft date/order/etc., and mount it into the container at
`/config/league.yaml`. The bot **hot-reloads** it when the file changes — no
rebuild, no restart.

**Championship history — a SQLite database.** History is persisted in a SQLite
file on a mounted volume (`DB_PATH`, default `/data/dsotp.db`), so it survives
restarts and redeploys. Record champions with `/recordchampion` (admins) or seed
them at deploy time with the CLI (see below). Nothing to hand-edit.

**Secrets & runtime settings — environment variables.** Nothing secret lives in
code or in the image.

| Variable               | Required | Default              | Purpose |
|------------------------|----------|----------------------|---------|
| `SLACK_SIGNING_SECRET` | ✅       | —                    | Verifies requests genuinely come from Slack (HMAC) |
| `LEAGUE_CONFIG_PATH`   |          | `/config/league.yaml`| Path to the mounted league file |
| `DB_PATH`              |          | `/data/dsotp.db`     | SQLite database path (mounted volume) |
| `ADMIN_USER_IDS`       |          | —                    | Comma-separated Slack user IDs allowed to run guarded commands |
| `PORT`                 |          | `8080`               | HTTP listen port |
| `LOG_LEVEL`            |          | `info`               | `debug` / `info` / `warn` / `error` |
| `SLACK_BOT_TOKEN`      |          | —                    | Optional; reserved for future scheduled posts |

To seed championship history at deploy time (or correct it), exec into the
container and run the CLI subcommand:

```sh
bot record-champion 2024 Hagel "Fighting Mongooses"
bot record-champion 2025 Collin "Vamonos Pest"
```

## Architecture

```
cmd/bot            HTTP server + CLI subcommands (record-champion), graceful shutdown
internal/config    env-var configuration
internal/slack     Slack signing-secret verification + slash-command / response types
internal/league    league data model + Provider seam (StaticProvider reads the YAML)
internal/store     SQLite persistence (championship history), pure-Go driver
internal/commands  command registry -> handlers; Env carries league + store per request
```

Requests are verified with Slack's signing-secret HMAC and answered **directly**
in the slash-command HTTP response (`response_type: in_channel`/`ephemeral`).

The `league.Provider` interface is the extensibility seam: today `StaticProvider`
serves the YAML file; later a `YahooProvider` / `SleeperProvider` can implement
the same interface to serve live standings and matchups, and new commands become
new handlers.

## Running locally

```sh
export SLACK_SIGNING_SECRET=dev-secret
export LEAGUE_CONFIG_PATH=config/league.example.yaml
go run ./cmd/bot
# health check
curl localhost:8080/healthz
```

To exercise a command locally you must send a correctly-signed request (see
`internal/slack/verify_test.go` for the signing scheme), or run the unit tests:

```sh
go test ./...
```

## Setup & deployment

First-time install has five parts: the Slack app, the container image, your
league file, the TrueNAS app, and your reverse proxy. Do them in order.

### 1. Configure the Slack app

1. Go to <https://api.slack.com/apps> → **Create New App** (or open your existing
   one) → install it to your workspace.
2. Under **Basic Information → App Credentials**, copy the **Signing Secret**.
   You'll paste it into the `SLACK_SIGNING_SECRET` env var in step 4. This is the
   only Slack credential the bot needs.
3. Under **Slash Commands**, create one command per feature below. Set every
   command's **Request URL** to `https://<your-host>/slack/commands` (the public
   hostname from step 5). Slack requires this URL to be reachable and HTTPS.

   | Command           | Suggested description |
   |-------------------|-----------------------|
   | `/draftinfo`      | When and where is the draft |
   | `/draftorder`     | The draft order |
   | `/punishment`     | League punishments |
   | `/remotedraft`    | Link to the remote draft room |
   | `/champion`       | Reigning champion (or a given year) |
   | `/champions`      | Full championship history |
   | `/titles`         | Championships by player |
   | `/recordchampion` | Record a champion (admins only) |

   > Slack validates the Request URL by sending a signed request, so it's easiest
   > to finish steps 4–5 first and come back to save these.
   >
   > For `/recordchampion` to work, put the commissioners' Slack user IDs in the
   > `ADMIN_USER_IDS` env var (step 4).

### 2. Make the container image pullable

Pushing to `main` builds and publishes `ghcr.io/hagel05/go-dsotp-bot:latest`
automatically (see `.github/workflows/release.yml`). By default a new GHCR
package is **private**, so TrueNAS can't pull it. Either:

- On GitHub → your profile → **Packages → go-dsotp-bot → Package settings**, set
  visibility to **Public**, **or**
- Keep it private and give TrueNAS registry credentials: a GitHub
  [personal access token](https://github.com/settings/tokens) with the
  `read:packages` scope, added under **Apps → Docker → Registry** on TrueNAS.

### 3. Create your league file

Copy [`config/league.example.yaml`](config/league.example.yaml) to a host path on
TrueNAS (e.g. `/mnt/tank/apps/dsotp-bot/config/league.yaml`) and edit the values.
This is the file you'll edit once a year — it never goes in the image or in git.

### 4. Create the TrueNAS app

On TrueNAS SCALE → **Apps → Discover Apps → Custom App → Install via YAML**, using
[`deploy/docker-compose.example.yml`](deploy/docker-compose.example.yml) as your
starting point:

- Set `SLACK_SIGNING_SECRET` to the value from step 1.
- Set `ADMIN_USER_IDS` to your commissioners' Slack user IDs (comma-separated).
- Mount the host config directory from step 3 at `/config`.
- Mount a host data directory (e.g. `/mnt/tank/apps/dsotp-bot/data`) at `/data`,
  **read-write** — this holds the championship database and must persist.
- Publish the container port (default `8080`) to your host / reverse proxy.

Confirm it's healthy: `curl http://<truenas-ip>:8080/healthz` should return `ok`.

Seed championship history once (exec into the container):

```sh
bot record-champion 2024 Hagel "Fighting Mongooses"
bot record-champion 2025 Collin "Vamonos Pest"
```

### 5. Expose it through your reverse proxy

Route `https://<your-host>/slack/commands` to the container's port. Slack only
talks to the bot over HTTPS, so termination happens at your proxy. Once the URL is
live, go back and save the slash commands from step 1 — Slack will verify the URL.

### Updating annual data

Edit `league.yaml` on the host (draft date/time, order, punishments). The bot
reloads it automatically within a request or two — no restart, no rebuild. If a
save leaves the file temporarily invalid, the bot keeps serving the last good
version and logs the parse error.

Championship history is updated separately (it's in the database, not the YAML):
run `/recordchampion <year> <player>` in Slack as an admin, or
`bot record-champion <year> <player>` in the container.

### Rotating the signing secret

If you regenerate the Slack signing secret, update `SLACK_SIGNING_SECRET` in the
TrueNAS app config and restart the app.

## CI/CD

- `.github/workflows/ci.yml` — `go vet`, `go test -race`, `go build` on every push/PR.
- `.github/workflows/release.yml` — builds and pushes the image to GHCR on pushes
  to `main` and on `v*` tags.

## Legacy AWS

The original Lambda/Terraform stack lives in
[`deploy/aws-legacy/`](deploy/aws-legacy/) for reference and can be
`terraform destroy`ed once the container is verified live.
