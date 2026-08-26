package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"scrapehub/internal/discover"
	"scrapehub/internal/fetch"
	"scrapehub/internal/health"
	"scrapehub/internal/metric"
	"scrapehub/internal/model"
	"scrapehub/internal/schedule"
)

// scrapeDispatcher adapts the fetch worker to the scheduler's dispatcher seam.
type scrapeDispatcher struct {
	pool *fetch.Pool
}

// Dispatch runs one scrape for a target.
func (d *scrapeDispatcher) Dispatch(ctx context.Context, target model.Target, cycle int64) error {
	return d.pool.Run(ctx, target, cycle)
}

// Server exposes the agent's management API and monitor page.
type Server struct {
	scheduler *schedule.Scheduler
	store     *discover.ListStore
	judge     *health.Judge
	registry  *metric.Registry
	history   *schedule.JobHistory
	provider  *discover.StaticProvider
}

// NewServer builds the HTTP handler dependencies.
func NewServer(
	scheduler *schedule.Scheduler,
	store *discover.ListStore,
	judge *health.Judge,
	registry *metric.Registry,
	history *schedule.JobHistory,
	provider *discover.StaticProvider,
) *Server {
	return &Server{scheduler: scheduler, store: store, judge: judge, registry: registry, history: history, provider: provider}
}

// Routes registers every endpoint.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/targets", s.handleTargets)
	mux.HandleFunc("POST /api/v1/targets", s.handleAddTarget)
	mux.HandleFunc("DELETE /api/v1/targets/{id}", s.handleRemoveTarget)
	mux.HandleFunc("POST /api/v1/cycle", s.handleCycle)
	mux.HandleFunc("GET /api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/v1/jobs", s.handleJobs)
	mux.HandleFunc("GET /api/v1/health/{id}", s.handleHealthOf)
	mux.HandleFunc("GET /monitor", s.handleMonitor)
	mux.HandleFunc("GET /", s.handleMonitor)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTargets(w http.ResponseWriter, _ *http.Request) {
	snapshot := s.store.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"generation": snapshot.Generation,
		"targets":    snapshot.Targets,
	})
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var target model.Target
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !target.Valid() {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target needs id and endpoint"))
		return
	}
	if s.provider == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target registration disabled in service discovery mode"))
		return
	}
	target.Kind = model.TargetStatic
	s.provider.Add(target)
	writeJSON(w, http.StatusOK, map[string]string{"added": target.ID})
}

func (s *Server) handleRemoveTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.provider == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("target removal disabled in service discovery mode"))
		return
	}
	s.provider.Remove(id)
	writeJSON(w, http.StatusOK, map[string]string{"removed": id})
}

func (s *Server) handleCycle(w http.ResponseWriter, _ *http.Request) {
	report := s.scheduler.RunCycle(time.Now())
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.Snapshot())
}

func (s *Server) handleJobs(w http.ResponseWriter, _ *http.Request) {
	records := s.history.Recent()
	jobs := make([]map[string]any, 0, len(records))
	for _, record := range records {
		jobs = append(jobs, map[string]any{
			"target":  record.TargetID,
			"cycle":   record.Cycle,
			"state":   record.State.String(),
			"attempt": record.Attempt,
			"error":   record.Err,
			"at":      record.At,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(jobs),
		"jobs":  jobs,
	})
}

func (s *Server) handleHealthOf(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"state": s.judge.State(id).String(),
		"allow": s.judge.Allow(id),
	})
}

func (s *Server) handleMonitor(w http.ResponseWriter, _ *http.Request) {
	path := filepath.Join("web", "monitor.html")
	content, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("monitor page unavailable: %w", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}
