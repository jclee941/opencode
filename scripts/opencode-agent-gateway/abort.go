package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func (g *gateway) handleAbort(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	current, ok := g.store.get(jobID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if isTerminalJobStatus(current.Status) {
		writeJSON(w, map[string]string{
			"job_id":  current.ID,
			"status":  current.Status,
			"message": "job already in terminal state",
		})
		return
	}

	cancelledAt := time.Now().UTC()
	aborted := false
	updated, ok := g.store.update(jobID, func(j *job) {
		if isTerminalJobStatus(j.Status) {
			return
		}
		j.Status = "cancelled"
		j.Error = "aborted by user"
		j.CompletedAt = &cancelledAt
		if j.StartedAt != nil {
			j.DurationMs = cancelledAt.Sub(*j.StartedAt).Milliseconds()
		}
		aborted = true
	})
	if !ok {
		http.NotFound(w, r)
		return
	}

	if !aborted {
		writeJSON(w, map[string]string{
			"job_id":  updated.ID,
			"status":  updated.Status,
			"message": "job already in terminal state",
		})
		return
	}

	if err := g.postCallback(updated); err != nil {
		fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
	}

	writeJSON(w, map[string]string{
		"job_id":  updated.ID,
		"status":  updated.Status,
		"message": "job abort requested",
	})
}
