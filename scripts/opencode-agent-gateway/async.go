package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const asyncCompletionResult = "async session completed"

type asyncRunner struct {
	gw         *gateway
	cancel     context.CancelFunc
	mu         sync.Mutex
	lastStatus string
}

func (r *asyncRunner) observeStatus(status string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	completed := status == "idle" && r.lastStatus == "busy"
	r.lastStatus = status
	return completed
}

func (r *asyncRunner) lastSeenStatus() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastStatus
}

func (g *gateway) runJobAsync(jobID string) {
	select {
	case g.sem <- struct{}{}:
	case <-g.ctx.Done():
		fmt.Fprintf(os.Stderr, "skip async job, shutdown in progress: job_id=%s\n", jobID)
		cancelled := time.Now().UTC()
		updated, ok := g.store.update(jobID, func(j *job) {
			j.Status = "cancelled"
			j.Error = "gateway shutdown before execution"
			j.CompletedAt = &cancelled
		})
		if ok {
			if err := g.postCallback(updated); err != nil {
				fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
			}
		}
		return
	}
	defer func() { <-g.sem }()

	started := time.Now().UTC()
	current, ok := g.store.update(jobID, func(j *job) {
		j.Status = "running"
		j.StartedAt = &started
	})
	if !ok {
		fmt.Fprintf(os.Stderr, "async job not found while starting: job_id=%s\n", jobID)
		return
	}

	providerID, modelID, err := splitModel(current.Model)
	if err != nil {
		g.failJob(current, err)
		return
	}

	runCtx, runCancel := context.WithTimeout(g.ctx, 20*time.Minute)
	defer runCancel()

	sessionID, err := g.client.createSession(runCtx, "agent-gateway: "+current.ID)
	if err != nil {
		g.failJob(current, err)
		return
	}
	current, _ = g.store.update(current.ID, func(j *job) {
		j.SessionID = sessionID
	})

	if err := g.client.promptAsync(runCtx, sessionID, current.Prompt, providerID, modelID, current.Format); err != nil {
		g.failJob(current, err)
		return
	}

	monitorCtx, monitorCancel := context.WithCancel(runCtx)
	runner := &asyncRunner{gw: g, cancel: monitorCancel}
	defer runner.cancel()

	done := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.monitorSession(monitorCtx, sessionID, done)
	}()

	select {
	case <-done:
		g.completeAsyncJob(current, started)
	case err := <-errCh:
		if err == nil {
			g.completeAsyncJob(current, started)
			return
		}
		if errors.Is(err, context.Canceled) && g.ctx.Err() != nil {
			g.cancelAsyncJob(current, started, "gateway shutdown during async execution")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			g.failJob(current, fmt.Errorf("async wait timed out after %s", 20*time.Minute))
			return
		}
		g.failJob(current, err)
	case <-runCtx.Done():
		if errors.Is(runCtx.Err(), context.Canceled) && g.ctx.Err() != nil {
			g.cancelAsyncJob(current, started, "gateway shutdown during async execution")
			return
		}
		g.failJob(current, fmt.Errorf("async wait timed out after %s", 20*time.Minute))
	}
}

func (r *asyncRunner) monitorSession(ctx context.Context, sessionID string, done chan struct{}) error {
	var doneOnce sync.Once
	signalDone := func() {
		doneOnce.Do(func() {
			close(done)
			if r.cancel != nil {
				r.cancel()
			}
		})
	}

	reconnectUsed := false
	for {
		body, err := r.gw.client.connectEvents(ctx)
		if err != nil {
			if reconnectUsed {
				return r.reconcileSession(ctx, sessionID, err, signalDone)
			}
			reconnectUsed = true
			if err := waitForReconnect(ctx); err != nil {
				return err
			}
			body, err = r.gw.client.connectEvents(ctx)
			if err != nil {
				return r.reconcileSession(ctx, sessionID, err, signalDone)
			}
		}

		readErr := r.readEventStream(ctx, body, sessionID, signalDone)
		if readErr == nil {
			select {
			case <-done:
				return nil
			default:
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if reconnectUsed {
			return r.reconcileSession(ctx, sessionID, readErr, signalDone)
		}
		reconnectUsed = true
		if err := waitForReconnect(ctx); err != nil {
			return err
		}
	}
}

func (r *asyncRunner) readEventStream(ctx context.Context, body io.ReadCloser, sessionID string, signalDone func()) error {
	defer body.Close()

	return decodeSSEEvents(body, func(env eventEnvelope) bool {
		if ctx.Err() != nil {
			return false
		}
		if extractSessionID(env) != sessionID {
			return true
		}
		if env.Payload.Type != "session.status" {
			return true
		}

		var props sessionStatusProps
		if err := json.Unmarshal(env.Payload.Properties, &props); err != nil {
			return true
		}
		if r.observeStatus(props.Status) {
			signalDone()
			return false
		}
		return true
	})
}

func (r *asyncRunner) reconcileSession(ctx context.Context, sessionID string, streamErr error, signalDone func()) error {
	statusBySession, err := r.gw.client.getSessionStatus(ctx)
	if err != nil {
		if streamErr != nil {
			return fmt.Errorf("async event reconciliation failed after stream error: %w (reconcile: %v)", streamErr, err)
		}
		return fmt.Errorf("async event reconciliation failed: %w", err)
	}

	status := statusBySession[sessionID]
	if status == "idle" {
		signalDone()
		return nil
	}
	if status == "busy" {
		if streamErr != nil {
			return fmt.Errorf("async session still busy after event stream dropped: %w", streamErr)
		}
		return fmt.Errorf("async session still busy after reconciliation")
	}
	if status == "" {
		return fmt.Errorf("async session status missing during reconciliation")
	}
	return fmt.Errorf("async session reconciliation returned unexpected status=%q (last_sse_status=%q)", status, r.lastSeenStatus())
}

func waitForReconnect(ctx context.Context) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (g *gateway) completeAsyncJob(existing *job, started time.Time) {
	completed := time.Now().UTC()
	durationMs := completed.Sub(started).Milliseconds()

	// Retrieve actual AI response from session messages
	var result string
	if existing.SessionID != "" {
		retrieveCtx, retrieveCancel := context.WithTimeout(g.ctx, 30*time.Second)
		text, err := g.client.getSessionMessages(retrieveCtx, existing.SessionID)
		retrieveCancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "async job message retrieval failed, using fallback: job_id=%s err=%v\n", existing.ID, err)
			result = asyncCompletionResult
		} else if strings.TrimSpace(text) != "" {
			result = text
		} else {
			result = asyncCompletionResult
		}
	} else {
		result = asyncCompletionResult
	}

	updated, ok := g.store.update(existing.ID, func(j *job) {
		if isTerminalJobStatus(j.Status) {
			return
		}
		j.Status = "completed"
		j.Result = result
		j.Error = ""
		j.CompletedAt = &completed
		j.DurationMs = durationMs
	})
	if !ok || updated == nil || updated.Status != "completed" {
		return
	}
	fmt.Fprintf(os.Stderr, "async job completed: job_id=%s session_id=%s duration_ms=%d result_len=%d\n", updated.ID, updated.SessionID, updated.DurationMs, len(updated.Result))
	if err := g.postCallback(updated); err != nil {
		fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
	}
}

func (g *gateway) cancelAsyncJob(existing *job, started time.Time, message string) {
	completed := time.Now().UTC()
	durationMs := completed.Sub(started).Milliseconds()
	updated, ok := g.store.update(existing.ID, func(j *job) {
		if isTerminalJobStatus(j.Status) {
			return
		}
		j.Status = "cancelled"
		j.Error = message
		j.CompletedAt = &completed
		j.DurationMs = durationMs
	})
	if !ok || updated == nil || updated.Status != "cancelled" {
		return
	}
	fmt.Fprintf(os.Stderr, "async job cancelled: job_id=%s err=%s\n", updated.ID, message)
	if err := g.postCallback(updated); err != nil {
		fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
	}
}

func isTerminalJobStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}
