package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/database"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbQueryDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"repository", "operation"},
	)
)

// MetricsExecutor wraps a database.Executor and records query latency.
type MetricsExecutor struct {
	exec       database.Executor
	repository string
}

// NewMetricsExecutor returns an Executor that records Prometheus metrics.
func NewMetricsExecutor(exec database.Executor, repository string) database.Executor {
	return &MetricsExecutor{exec: exec, repository: repository}
}

func (m *MetricsExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := m.exec.ExecContext(ctx, query, args...)
	dbQueryDurationSeconds.WithLabelValues(m.repository, "exec").Observe(time.Since(start).Seconds())
	return result, err
}

func (m *MetricsExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := m.exec.QueryContext(ctx, query, args...)
	dbQueryDurationSeconds.WithLabelValues(m.repository, "query").Observe(time.Since(start).Seconds())
	return rows, err
}

func (m *MetricsExecutor) QueryRowContext(ctx context.Context, query string, args ...any) database.Row {
	start := time.Now()
	row := m.exec.QueryRowContext(ctx, query, args...)
	return &metricsRow{row: row, record: func() {
		dbQueryDurationSeconds.WithLabelValues(m.repository, "query_row").Observe(time.Since(start).Seconds())
	}}
}

// metricsRow wraps sql.Row so we record duration when Scan is called (when the query actually runs).
type metricsRow struct {
	row    database.Row
	record func()
}

func (m *metricsRow) Scan(dest ...any) error {
	defer m.record()
	return m.row.Scan(dest...)
}
