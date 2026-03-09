CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS games (
    game_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    game_id UUID REFERENCES games(game_id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    event_type TEXT NOT NULL,
    payload_json JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS snapshots (
    id BIGSERIAL PRIMARY KEY,
    game_id UUID REFERENCES games(game_id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    state_json JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS factories (
    id TEXT NOT NULL,
    game_id UUID REFERENCES games(game_id) ON DELETE CASCADE,
    state_json JSONB NOT NULL,
    PRIMARY KEY (id, game_id)
);

CREATE INDEX IF NOT EXISTS idx_events_game_id ON events(game_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_game_id ON snapshots(game_id, timestamp DESC);
