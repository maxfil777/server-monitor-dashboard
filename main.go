package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Config struct {
	ListenAddr         string `json:"listen_addr"`
	MetricsURL         string `json:"metrics_url"`
	PollIntervalSec    int    `json:"poll_interval_seconds"`
	RetentionDays      int    `json:"retention_days"`
	DatabasePath       string `json:"database_path"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8090"
	}
	if cfg.MetricsURL == "" {
		cfg.MetricsURL = "http://127.0.0.1:8080/metrics.json"
	}
	if cfg.PollIntervalSec <= 0 {
		cfg.PollIntervalSec = 60
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 30
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./metrics.db"
	}
	return &cfg, nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("starting server-monitor-dashboard")

	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	store, err := NewStorage(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("storage error: %v", err)
	}
	defer store.Close()

	collector := NewCollector(cfg, store)
	collector.Start()

	handler := NewHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/latest", handler.Latest)
	mux.HandleFunc("/api/metrics", handler.Metrics)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", handler.Dashboard)

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
