package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	store *Storage
}

func NewHandler(store *Storage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Latest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m, err := h.store.Latest()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if m == nil {
		w.Write([]byte(`{"message":"no data yet"}`))
		return
	}
	json.NewEncoder(w).Encode(m)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	to := now.Unix()
	from := now.Add(-24 * time.Hour).Unix()

	if f := r.URL.Query().Get("from"); f != "" {
		if v, err := strconv.ParseInt(f, 10, 64); err == nil {
			from = v
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if v, err := strconv.ParseInt(t, 10, 64); err == nil {
			to = v
		}
	}

	metrics, err := h.store.GetMetrics(from, to)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if metrics == nil {
		metrics = []Metrics{}
	}
	json.NewEncoder(w).Encode(metrics)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}
