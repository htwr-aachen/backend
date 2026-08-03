-- +goose Up
CREATE SCHEMA IF NOT EXISTS panikzettel;

CREATE TABLE panikzettel.downloads (
    filename TEXT PRIMARY KEY,
    downloads BIGINT NOT NULL DEFAULT 0 CHECK (downloads >= 0),
    first_download_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_download_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Covers the "most downloaded" listing without touching the heap for ordering.
CREATE INDEX idx_panikzettel_downloads_count ON panikzettel.downloads (
    downloads DESC, filename ASC
);

-- +goose Down
DROP INDEX IF EXISTS panikzettel.idx_panikzettel_downloads_count;
DROP TABLE IF EXISTS panikzettel.downloads;
DROP SCHEMA IF EXISTS panikzettel;
