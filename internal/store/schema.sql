-- Schema for the DSotP bot's persistent data.
-- Applied idempotently at startup; safe to run on every boot.

CREATE TABLE IF NOT EXISTS champions (
    year      INTEGER PRIMARY KEY,
    team_name TEXT NOT NULL DEFAULT '',
    player    TEXT NOT NULL
);
