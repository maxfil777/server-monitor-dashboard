package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type Collector struct {
	cfg    *Config
	store  *Storage
	client *http.Client
}

func NewCollector(cfg *Config, store *Storage) *Collector {
	return &Collector{
		cfg:    cfg,
		store:  store,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Collector) Start() {
	ticker := time.NewTicker(time.Duration(c.cfg.PollIntervalSec) * time.Second)
	go func() {
		c.collect()
		for range ticker.C {
			c.collect()
		}
	}()
}

func (c *Collector) collect() {
	resp, err := c.client.Get(c.cfg.MetricsURL)
	if err != nil {
		log.Printf("fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read error: %v", err)
		return
	}

	var ext struct {
		CPU             float64 `json:"cpu"`
		Mem             float64 `json:"mem"`
		FPMActive       int     `json:"fpm_active"`
		FPMStatusErrors int     `json:"fpm_status_errors"`
	}
	if err := json.Unmarshal(body, &ext); err != nil {
		log.Printf("parse error: %v", err)
		return
	}

	m := Metrics{
		CreatedAt:        time.Now().Unix(),
		CPUPercent:       ext.CPU,
		MemoryPercent:    ext.Mem,
		FPMActive:        ext.FPMActive,
		FPMStatusErrors:  ext.FPMStatusErrors,
	}

	if err := c.store.Save(m); err != nil {
		log.Printf("save error: %v", err)
		return
	}

	log.Printf("saved: cpu=%.1f%% mem=%.1f%% fpm=%d err=%d", m.CPUPercent, m.MemoryPercent, m.FPMActive, m.FPMStatusErrors)

	cutoff := time.Now().AddDate(0, 0, -c.cfg.RetentionDays).Unix()
	if err := c.store.Cleanup(cutoff); err != nil {
		log.Printf("cleanup error: %v", err)
	}
}
