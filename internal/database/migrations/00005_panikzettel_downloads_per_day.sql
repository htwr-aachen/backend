-- +goose Up
-- The semester counters are replaced by daily buckets, so that rolling windows
-- can be reported next to the semester. Existing rows only carry the day of
-- their last download, so their whole count is folded into that day.
CREATE TEMP TABLE panikzettel_downloads_days AS
SELECT
    filename,
    last_download_at::date AS day,
    SUM(downloads) AS downloads,
    MIN(first_download_at) AS first_download_at,
    MAX(last_download_at) AS last_download_at
FROM panikzettel.downloads
GROUP BY filename, last_download_at::date;

TRUNCATE panikzettel.downloads;

DROP INDEX panikzettel.idx_panikzettel_downloads_count;

ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_pkey;
ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_semester_format;
ALTER TABLE panikzettel.downloads DROP COLUMN semester;
ALTER TABLE panikzettel.downloads ADD COLUMN day DATE NOT NULL;
ALTER TABLE panikzettel.downloads ADD PRIMARY KEY (filename, day);

INSERT INTO panikzettel.downloads (
    filename, day, downloads, first_download_at, last_download_at
)
SELECT filename, day, downloads, first_download_at, last_download_at
FROM panikzettel_downloads_days;

DROP TABLE panikzettel_downloads_days;

-- Every window query scans a contiguous range of days.
CREATE INDEX idx_panikzettel_downloads_day ON panikzettel.downloads (
    day DESC, filename ASC
);

-- +goose Down
CREATE TEMP TABLE panikzettel_downloads_semesters AS
SELECT
    filename,
    CASE
        WHEN EXTRACT(MONTH FROM day) BETWEEN 4 AND 9
            THEN 'ss' || EXTRACT(YEAR FROM day)::int
        WHEN EXTRACT(MONTH FROM day) >= 10
            THEN 'ws' || EXTRACT(YEAR FROM day)::int
        ELSE 'ws' || (EXTRACT(YEAR FROM day)::int - 1)
    END AS semester,
    SUM(downloads) AS downloads,
    MIN(first_download_at) AS first_download_at,
    MAX(last_download_at) AS last_download_at
FROM panikzettel.downloads
GROUP BY filename, 2;

TRUNCATE panikzettel.downloads;

DROP INDEX panikzettel.idx_panikzettel_downloads_day;

ALTER TABLE panikzettel.downloads DROP CONSTRAINT downloads_pkey;
ALTER TABLE panikzettel.downloads DROP COLUMN day;
ALTER TABLE panikzettel.downloads ADD COLUMN semester TEXT NOT NULL;
ALTER TABLE panikzettel.downloads
    ADD CONSTRAINT downloads_semester_format CHECK (semester ~ '^(ss|ws)[0-9]{4}$');
ALTER TABLE panikzettel.downloads ADD PRIMARY KEY (filename, semester);

INSERT INTO panikzettel.downloads (
    filename, semester, downloads, first_download_at, last_download_at
)
SELECT filename, semester, downloads, first_download_at, last_download_at
FROM panikzettel_downloads_semesters;

DROP TABLE panikzettel_downloads_semesters;

CREATE INDEX idx_panikzettel_downloads_count ON panikzettel.downloads (
    semester, downloads DESC, filename ASC
);
