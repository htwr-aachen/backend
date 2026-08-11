-- +goose Up
ALTER TABLE panikzettel.downloads ADD COLUMN semester TEXT;

UPDATE panikzettel.downloads SET semester = CASE
    WHEN EXTRACT(MONTH FROM first_download_at) BETWEEN 4 AND 9
        THEN 'ss' || EXTRACT(YEAR FROM first_download_at)::int
    WHEN EXTRACT(MONTH FROM first_download_at) >= 10
        THEN 'ws' || EXTRACT(YEAR FROM first_download_at)::int
    ELSE 'ws' || (EXTRACT(YEAR FROM first_download_at)::int - 1)
END;

ALTER TABLE panikzettel.downloads
    ALTER COLUMN semester SET NOT NULL,
    ADD CONSTRAINT downloads_semester_format CHECK (semester ~ '^(ss|ws)[0-9]{4}$');

DROP INDEX panikzettel.idx_panikzettel_downloads_count;

ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_pkey;
ALTER TABLE panikzettel.downloads ADD PRIMARY KEY (filename, semester);

CREATE INDEX idx_panikzettel_downloads_count ON panikzettel.downloads (
    semester, downloads DESC, filename ASC
);

-- +goose Down
DROP INDEX panikzettel.idx_panikzettel_downloads_count;

-- Downloads of all semesters are folded back into a single row per panikzettel.
CREATE TEMP TABLE panikzettel_downloads_totals AS
SELECT
    filename,
    SUM(downloads) AS downloads,
    MIN(first_download_at) AS first_download_at,
    MAX(last_download_at) AS last_download_at
FROM panikzettel.downloads
GROUP BY filename;

TRUNCATE panikzettel.downloads;

ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_pkey;
ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_semester_format;
ALTER TABLE panikzettel.downloads DROP COLUMN semester;
ALTER TABLE panikzettel.downloads ADD PRIMARY KEY (filename);

INSERT INTO panikzettel.downloads (
    filename, downloads, first_download_at, last_download_at
)
SELECT filename, downloads, first_download_at, last_download_at
FROM panikzettel_downloads_totals;

DROP TABLE panikzettel_downloads_totals;

CREATE INDEX idx_panikzettel_downloads_count ON panikzettel.downloads (
    downloads DESC, filename ASC
);
