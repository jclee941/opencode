package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestJobStatusEndpointForRunJobs(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	created, _, err := g.store.createOrGet(jobRequest{
		JobID:       "status-run-1",
		Prompt:      "hello",
		Mode:        "run",
		CallbackURL: "http://127.0.0.1:1/callback",
	}, g.cfg.defaultModel)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	t.Run("queued", func(t *testing.T) {
		code, body := getJSON(t, srv.URL+"/jobs/"+created.ID+"/status")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var got jobStatusResponse
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode status payload: %v", err)
		}
		if got.JobID != created.ID || got.Status != "queued" || got.Mode != "run" || got.SessionStatus != "" {
			t.Fatalf("unexpected queued status payload: %+v", got)
		}
	})

	started := time.Now().UTC()
	_, ok := g.store.update(created.ID, func(j *job) {
		j.Status = "running"
		j.StartedAt = &started
	})
	if !ok {
		t.Fatalf("job missing during running update")
	}

	t.Run("running", func(t *testing.T) {
		code, body := getJSON(t, srv.URL+"/jobs/"+created.ID+"/status")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var got jobStatusResponse
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode status payload: %v", err)
		}
		if got.JobID != created.ID || got.Status != "running" || got.Mode != "run" || got.SessionStatus != "" {
			t.Fatalf("unexpected running status payload: %+v", got)
		}
	})

	completed := time.Now().UTC()
	_, ok = g.store.update(created.ID, func(j *job) {
		j.Status = "completed"
		j.CompletedAt = &completed
		j.DurationMs = completed.Sub(started).Milliseconds()
	})
	if !ok {
		t.Fatalf("job missing during completed update")
	}

	t.Run("completed", func(t *testing.T) {
		code, body := getJSON(t, srv.URL+"/jobs/"+created.ID+"/status")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var got jobStatusResponse
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode status payload: %v", err)
		}
		if got.JobID != created.ID || got.Status != "completed" || got.Mode != "run" || got.SessionStatus != "" {
			t.Fatalf("unexpected completed status payload: %+v", got)
		}
	})
}

func TestJobStatusEndpointAsyncRunningUsesSessionStatus(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	g, srv := testGatewayAndServer(t, mock.url())

	created, _, err := g.store.createOrGet(jobRequest{
		JobID:       "status-async-1",
		Prompt:      "hello",
		Mode:        "async",
		CallbackURL: "http://127.0.0.1:1/callback",
	}, g.cfg.defaultModel)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	started := time.Now().UTC()
	_, ok := g.store.update(created.ID, func(j *job) {
		j.Status = "running"
		j.StartedAt = &started
		j.SessionID = "session-42"
	})
	if !ok {
		t.Fatalf("job missing during running update")
	}

	mock.mu.Lock()
	mock.sessionStatus["session-42"] = "busy"
	mock.mu.Unlock()

	code, body := getJSON(t, srv.URL+"/jobs/"+created.ID+"/status")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	var got jobStatusResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if got.JobID != created.ID || got.Status != "running" || got.Mode != "async" || got.SessionStatus != "busy" {
		t.Fatalf("unexpected async running payload: %+v", got)
	}
}

func TestJobProgressEndpointStages(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	created, _, err := g.store.createOrGet(jobRequest{
		JobID:       "progress-1",
		Prompt:      "hello",
		Mode:        "run",
		CallbackURL: "http://127.0.0.1:1/callback",
	}, g.cfg.defaultModel)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	readProgress := func(t *testing.T) jobProgressResponse {
		t.Helper()
		code, body := getJSON(t, srv.URL+"/jobs/"+created.ID+"/progress")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var got jobProgressResponse
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode progress payload: %v", err)
		}
		return got
	}

	t.Run("queued", func(t *testing.T) {
		got := readProgress(t)
		if got.JobID != created.ID || got.Status != "queued" || got.Mode != "run" {
			t.Fatalf("unexpected queued progress payload: %+v", got)
		}
		if got.StartedAt != nil || got.CompletedAt != nil {
			t.Fatalf("queued progress should not include started/completed timestamps: %+v", got)
		}
		if got.DurationMs != 0 || got.SessionID != "" {
			t.Fatalf("queued progress should have zero duration and empty session_id: %+v", got)
		}
	})

	started := time.Now().UTC()
	_, ok := g.store.update(created.ID, func(j *job) {
		j.Status = "running"
		j.StartedAt = &started
		j.SessionID = "session-99"
	})
	if !ok {
		t.Fatalf("job missing during running update")
	}

	t.Run("running", func(t *testing.T) {
		got := readProgress(t)
		if got.Status != "running" || got.SessionID != "session-99" {
			t.Fatalf("unexpected running progress payload: %+v", got)
		}
		if got.StartedAt == nil || !got.StartedAt.Equal(started) {
			t.Fatalf("running progress missing/invalid started_at: %+v", got)
		}
		if got.CompletedAt != nil {
			t.Fatalf("running progress should not include completed_at: %+v", got)
		}
	})

	completed := started.Add(3 * time.Second)
	_, ok = g.store.update(created.ID, func(j *job) {
		j.Status = "completed"
		j.CompletedAt = &completed
		j.DurationMs = completed.Sub(started).Milliseconds()
	})
	if !ok {
		t.Fatalf("job missing during completed update")
	}

	t.Run("completed", func(t *testing.T) {
		got := readProgress(t)
		if got.Status != "completed" {
			t.Fatalf("unexpected completed progress payload: %+v", got)
		}
		if got.CompletedAt == nil || !got.CompletedAt.Equal(completed) {
			t.Fatalf("completed progress missing/invalid completed_at: %+v", got)
		}
		if got.DurationMs != completed.Sub(started).Milliseconds() {
			t.Fatalf("unexpected duration_ms: %d", got.DurationMs)
		}
	})
}

func TestJobReadEndpointsNotFound(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	for _, path := range []string{"/jobs/missing/status", "/jobs/missing/progress"} {
		code, _ := getJSON(t, srv.URL+path)
		if code != http.StatusNotFound {
			t.Fatalf("GET %s expected 404, got %d", path, code)
		}
	}
}

func TestJobReadEndpointsMethodNotAllowed(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	for _, path := range []string{"/jobs/any/status", "/jobs/any/progress"} {
		req, err := http.NewRequest(http.MethodPost, srv.URL+path, nil)
		if err != nil {
			t.Fatalf("build POST %s request: %v", path, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST %s failed: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s expected 405, got %d", path, resp.StatusCode)
		}
	}
}
