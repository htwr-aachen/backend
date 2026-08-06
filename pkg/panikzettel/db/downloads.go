package db

import (
	"context"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/rs/zerolog/log"
)

// IncrementDownloads folds a batch of counted downloads into the persisted
// counters. Rows are upserted in a single statement so that concurrent
// instances can only ever add to each other's counts.
func (db *DB) IncrementDownloads(ctx context.Context, deltas []models.DownloadDelta) error {
	if len(deltas) == 0 {
		return nil
	}

	filenames := make([]string, 0, len(deltas))
	counts := make([]int64, 0, len(deltas))
	firsts := make([]time.Time, 0, len(deltas))
	lasts := make([]time.Time, 0, len(deltas))

	for _, delta := range deltas {
		filenames = append(filenames, delta.Filename)
		counts = append(counts, delta.Count)
		firsts = append(firsts, delta.FirstDownloadAt)
		lasts = append(lasts, delta.LastDownloadAt)
	}

	query := `
INSERT INTO panikzettel.downloads (
	filename, downloads, first_download_at, last_download_at
)
SELECT
	d.filename, d.downloads, d.first_download_at, d.last_download_at
FROM unnest($1::text[], $2::bigint[], $3::timestamptz[], $4::timestamptz[])
	AS d (filename, downloads, first_download_at, last_download_at)
ON CONFLICT (filename) DO UPDATE SET
	downloads = panikzettel.downloads.downloads + excluded.downloads,
	first_download_at = LEAST(
		panikzettel.downloads.first_download_at, excluded.first_download_at
	),
	last_download_at = GREATEST(
		panikzettel.downloads.last_download_at, excluded.last_download_at
	);
`

	tag, err := db.db.Exec(ctx, query, filenames, counts, firsts, lasts)
	if err != nil {
		return fmt.Errorf("incrementing panikzettel downloads: %w", err)
	}

	log.Debug().Int64("rows", tag.RowsAffected()).Int("panikzettel", len(deltas)).Msg("flushed panikzettel download counts")

	return nil
}

// ListDownloads returns the persisted download counters, most downloaded first.
func (db *DB) ListDownloads(ctx context.Context) ([]models.DownloadStat, error) {
	query := `
SELECT
	d.filename,
	d.downloads,
	d.first_download_at,
	d.last_download_at
FROM panikzettel.downloads AS d
ORDER BY d.downloads DESC, d.filename ASC;
`

	rows, err := db.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying panikzettel downloads: %w", err)
	}
	defer rows.Close()

	stats := make([]models.DownloadStat, 0)

	for rows.Next() {
		var stat models.DownloadStat
		if err := rows.Scan(&stat.Filename, &stat.Downloads, &stat.FirstDownloadAt, &stat.LastDownloadAt); err != nil {
			return nil, fmt.Errorf("scanning panikzettel downloads: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating panikzettel download rows: %w", err)
	}

	return stats, nil
}
