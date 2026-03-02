package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/betim/goqueue/queue"

	// Pure-Go SQLite driver — no CGO needed.
	_ "modernc.org/sqlite"
)

// SQLiteStore persists jobs in a SQLite database file.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the database at path and runs migrations.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id         TEXT PRIMARY KEY,
		type       TEXT NOT NULL,
		payload    TEXT NOT NULL DEFAULT '{}',
		status     TEXT NOT NULL DEFAULT 'pending',
		result     TEXT NOT NULL DEFAULT '',
		error      TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		ended_at   DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	`
	_, err := db.Exec(query)
	return err
}

func (s *SQLiteStore) Save(job *queue.Job) error {
	query := `
	INSERT INTO jobs (id, type, payload, status, result, error, created_at, started_at, ended_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		status     = excluded.status,
		result     = excluded.result,
		error      = excluded.error,
		started_at = excluded.started_at,
		ended_at   = excluded.ended_at
	`
	_, err := s.db.Exec(query,
		job.ID,
		job.Type,
		string(job.Payload),
		string(job.Status),
		job.Result,
		job.Error,
		job.CreatedAt,
		nullableTime(job.StartedAt),
		nullableTime(job.EndedAt),
	)
	return err
}

func (s *SQLiteStore) Get(id string) (*queue.Job, error) {
	row := s.db.QueryRow(
		"SELECT id, type, payload, status, result, error, created_at, started_at, ended_at FROM jobs WHERE id = ?",
		id,
	)
	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return job, err
}

func (s *SQLiteStore) List(status queue.Status) ([]*queue.Job, error) {
	var rows *sql.Rows
	var err error

	if status == "" {
		rows, err = s.db.Query(
			"SELECT id, type, payload, status, result, error, created_at, started_at, ended_at FROM jobs ORDER BY created_at DESC",
		)
	} else {
		rows, err = s.db.Query(
			"SELECT id, type, payload, status, result, error, created_at, started_at, ended_at FROM jobs WHERE status = ? ORDER BY created_at DESC",
			string(status),
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*queue.Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if jobs == nil {
		jobs = []*queue.Job{}
	}
	return jobs, rows.Err()
}

func (s *SQLiteStore) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("job %s not found", id)
	}
	return nil
}

func (s *SQLiteStore) Stats() (map[string]int, error) {
	rows, err := s.db.Query("SELECT status, COUNT(*) FROM jobs GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"total":     0,
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
		stats["total"] += count
	}
	return stats, rows.Err()
}

// scanJob scans a single row into a Job.
func scanJob(row *sql.Row) (*queue.Job, error) {
	var j queue.Job
	var payload string
	var status string
	var startedAt, endedAt sql.NullTime

	err := row.Scan(&j.ID, &j.Type, &payload, &status, &j.Result, &j.Error, &j.CreatedAt, &startedAt, &endedAt)
	if err != nil {
		return nil, err
	}

	j.Payload = []byte(payload)
	j.Status = queue.Status(status)
	if startedAt.Valid {
		j.StartedAt = startedAt.Time
	}
	if endedAt.Valid {
		j.EndedAt = endedAt.Time
	}
	return &j, nil
}

// scanJobRows scans the current row from sql.Rows into a Job.
func scanJobRows(rows *sql.Rows) (*queue.Job, error) {
	var j queue.Job
	var payload string
	var status string
	var startedAt, endedAt sql.NullTime

	err := rows.Scan(&j.ID, &j.Type, &payload, &status, &j.Result, &j.Error, &j.CreatedAt, &startedAt, &endedAt)
	if err != nil {
		return nil, err
	}

	j.Payload = []byte(payload)
	j.Status = queue.Status(status)
	if startedAt.Valid {
		j.StartedAt = startedAt.Time
	}
	if endedAt.Valid {
		j.EndedAt = endedAt.Time
	}
	return &j, nil
}

// nullableTime converts a zero time to nil for SQL NULL.
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
