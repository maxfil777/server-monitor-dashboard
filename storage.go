package main

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type Metrics struct {
	CreatedAt          int64   `json:"created_at"`
	CPUPercent         float64 `json:"cpu_percent"`
	MemoryPercent      float64 `json:"memory_percent"`
	FPMActive          int     `json:"fpm_active"`
	FPMStatusErrors    int     `json:"fpm_status_errors"`
}

type Storage struct {
	db *sql.DB
	mu sync.RWMutex
}

const maxDisplayPoints = 1500

func NewStorage(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at INTEGER NOT NULL,
		cpu_percent REAL NOT NULL,
		memory_percent REAL NOT NULL,
		fpm_active INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_metrics_created_at ON metrics(created_at)"); err != nil {
		db.Close()
		return nil, fmt.Errorf("create index: %w", err)
	}

	db.Exec("ALTER TABLE metrics ADD COLUMN fpm_status_errors INTEGER NOT NULL DEFAULT 0")

	return &Storage{db: db}, nil
}

func (s *Storage) Save(m Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"INSERT INTO metrics (created_at, cpu_percent, memory_percent, fpm_active, fpm_status_errors) VALUES (?, ?, ?, ?, ?)",
		m.CreatedAt, m.CPUPercent, m.MemoryPercent, m.FPMActive, m.FPMStatusErrors,
	)
	return err
}

func (s *Storage) Latest() (*Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var m Metrics
	err := s.db.QueryRow(
		"SELECT created_at, cpu_percent, memory_percent, fpm_active, fpm_status_errors FROM metrics ORDER BY created_at DESC LIMIT 1",
	).Scan(&m.CreatedAt, &m.CPUPercent, &m.MemoryPercent, &m.FPMActive, &m.FPMStatusErrors)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Storage) GetMetrics(from, to int64) ([]Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		"SELECT created_at, cpu_percent, memory_percent, fpm_active, fpm_status_errors FROM metrics WHERE created_at >= ? AND created_at <= ? ORDER BY created_at ASC",
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Metrics
	for rows.Next() {
		var m Metrics
		if err := rows.Scan(&m.CreatedAt, &m.CPUPercent, &m.MemoryPercent, &m.FPMActive, &m.FPMStatusErrors); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(result) > maxDisplayPoints {
		step := float64(len(result)) / float64(maxDisplayPoints)
		sampled := make([]Metrics, 0, maxDisplayPoints)
		for i := 0; i < maxDisplayPoints; i++ {
			idx := int(float64(i) * step)
			if idx >= len(result) {
				idx = len(result) - 1
			}
			sampled = append(sampled, result[idx])
		}
		result = sampled
	}

	return result, nil
}

func (s *Storage) Cleanup(before int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM metrics WHERE created_at < ?", before)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}
