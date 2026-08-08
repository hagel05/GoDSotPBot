-- Schema for the DSotP bot's persistent data.
-- Applied idempotently at startup; safe to run on every boot.

CREATE TABLE IF NOT EXISTS champions (
    year      INTEGER PRIMARY KEY,
    team_name TEXT NOT NULL DEFAULT '',
    player    TEXT NOT NULL
);

-- Generic key/value settings (e.g. the persisted Yahoo OAuth token).
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
