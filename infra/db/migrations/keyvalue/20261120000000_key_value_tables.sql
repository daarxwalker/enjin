-- +goose Up
-- Free space in each page lets Increment rewrite a counter in place (a HOT
-- update), since it changes neither indexed column.
CREATE UNLOGGED TABLE enjin_cache (
    key text COLLATE "C" PRIMARY KEY,
    value bytea NOT NULL,
    expires_at timestamptz
) WITH (fillfactor = 80);

CREATE INDEX enjin_cache_expires_at_idx ON enjin_cache (expires_at) WHERE expires_at IS NOT NULL;

CREATE UNLOGGED TABLE enjin_csrf (
    key text COLLATE "C" NOT NULL,
    token text COLLATE "C" NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (key, token)
);

CREATE INDEX enjin_csrf_expires_at_idx ON enjin_csrf (expires_at);

CREATE UNLOGGED TABLE enjin_lock (
    key text COLLATE "C" PRIMARY KEY,
    token text NOT NULL,
    expires_at timestamptz
);

CREATE INDEX enjin_lock_expires_at_idx ON enjin_lock (expires_at) WHERE expires_at IS NOT NULL;

-- +goose Down
DROP TABLE enjin_lock;
DROP TABLE enjin_csrf;
DROP TABLE enjin_cache;
