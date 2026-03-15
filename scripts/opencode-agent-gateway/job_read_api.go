package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type jobStatusResponse struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	SessionStatus string `json:"session_status"`
	Mode          string `json:"mode"`
}

type jobProgressResponse struct {
	JobID       string     `json:"job_id"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int64      `json:"duration_ms"`
	SessionID   string     `json:"session_id"`
	Mode        string     `json:"mode"`
}

func parseJobReadPath(path string) (string, string, bool) {
	relative := strings.TrimSpace(strings.TrimPrefix(path, "/jobs/"))
	if relative == "" {
		return "", "", false
	}
	parts := strings.Split(relative, "/")
	if len(parts) == 1 {
		if strings.TrimSpace(parts[0]) == "" {
			return "", "", false
		}
		return strings.TrimSpace(parts[0]), "", true
	}
	if len(parts) == 2 {
		if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", "", false
		}
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return "", "", false
}

func (g *gateway) handleJobDetails(w http.ResponseWriter, r *http.Request, id string) {
	v, ok := g.store.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, v)
}

func (g *gateway) handleJobStatus(w http.ResponseWriter, r *http.Request, id string) {
	v, ok := g.store.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	sessionStatus := ""
	if v.Mode == "async" && v.Status == "running" && strings.TrimSpace(v.SessionID) != "" {
		statusBySession, err := g.client.getSessionStatus(r.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "job status session lookup failed: job_id=%s session_id=%s err=%v\n", v.ID, v.SessionID, err)
		} else {
			sessionStatus = strings.TrimSpace(statusBySession[v.SessionID])
		}
	}

	writeJSON(w, jobStatusResponse{
		JobID:         v.ID,
		Status:        v.Status,
		SessionStatus: sessionStatus,
		Mode:          v.Mode,
	})
}

func (g *gateway) handleJobProgress(w http.ResponseWriter, r *http.Request, id string) {
	v, ok := g.store.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, jobProgressResponse{
		JobID:       v.ID,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		StartedAt:   v.StartedAt,
		CompletedAt: v.CompletedAt,
		DurationMs:  v.DurationMs,
		SessionID:   v.SessionID,
		Mode:        v.Mode,
	})
}
