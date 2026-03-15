package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

type abortResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func seedAbortJob(t *testing.T, g *gateway, id, status, callbackURL string) {
	t.Helper()
	_, created, err := g.store.createOrGet(jobRequest{
		JobID:       id,
		Prompt:      "abort test prompt",
		CallbackURL: callbackURL,
	}, g.cfg.defaultModel)
	if err != nil {
		t.Fatalf("seed job %s: %v", id, err)
	}
	if !created {
		t.Fatalf("job %s already exists", id)
	}

	switch status {
	case "queued":
		return
	case "running":
		started := time.Now().UTC().Add(-250 * time.Millisecond)
		_, ok := g.store.update(id, func(j *job) {
			j.Status = "running"
			j.StartedAt = &started
		})
		if !ok {
			t.Fatalf("seed running update failed for %s", id)
		}
	case "completed":
		completed := time.Now().UTC()
		_, ok := g.store.update(id, func(j *job) {
			j.Status = "completed"
			j.Result = "done"
			j.CompletedAt = &completed
		})
		if !ok {
			t.Fatalf("seed completed update failed for %s", id)
		}
	case "cancelled":
		completed := time.Now().UTC()
		_, ok := g.store.update(id, func(j *job) {
			j.Status = "cancelled"
			j.Error = "already cancelled"
			j.CompletedAt = &completed
		})
		if !ok {
			t.Fatalf("seed cancelled update failed for %s", id)
		}
	default:
		t.Fatalf("unsupported seed status %q", status)
	}
}

func callAbort(t *testing.T, baseURL, jobID, method string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+"/jobs/"+jobID+"/abort", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func decodeAbortResponse(t *testing.T, body []byte) abortResponse {
	t.Helper()
	var payload abortResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode abort response: %v body=%s", err, string(body))
	}
	return payload
}

func TestAbortQueuedJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-queued", "queued", "")

	code, body := callAbort(t, srv.URL, "abort-queued", http.MethodPost)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, string(body))
	}
	payload := decodeAbortResponse(t, body)
	if payload.JobID != "abort-queued" || payload.Status != "cancelled" || payload.Message != "job abort requested" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	got, ok := g.store.get("abort-queued")
	if !ok {
		t.Fatalf("expected job to exist")
	}
	if got.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %q", got.Status)
	}
	if got.Error != "aborted by user" {
		t.Fatalf("expected abort error, got %q", got.Error)
	}
	if got.CompletedAt == nil {
		t.Fatalf("expected completed_at to be set")
	}
}

func TestAbortRunningJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-running", "running", "")

	code, body := callAbort(t, srv.URL, "abort-running", http.MethodPost)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, string(body))
	}
	payload := decodeAbortResponse(t, body)
	if payload.Status != "cancelled" || payload.Message != "job abort requested" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	got, ok := g.store.get("abort-running")
	if !ok {
		t.Fatalf("expected job to exist")
	}
	if got.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %q", got.Status)
	}
	if got.Error != "aborted by user" {
		t.Fatalf("expected abort error, got %q", got.Error)
	}
	if got.DurationMs <= 0 {
		t.Fatalf("expected positive duration for running abort, got %d", got.DurationMs)
	}
}

func TestAbortAlreadyCompletedJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-completed", "completed", "")

	code, body := callAbort(t, srv.URL, "abort-completed", http.MethodPost)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, string(body))
	}
	payload := decodeAbortResponse(t, body)
	if payload.Status != "completed" || payload.Message != "job already in terminal state" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestAbortAlreadyCancelledJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-cancelled", "cancelled", "")

	code, body := callAbort(t, srv.URL, "abort-cancelled", http.MethodPost)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, string(body))
	}
	payload := decodeAbortResponse(t, body)
	if payload.Status != "cancelled" || payload.Message != "job already in terminal state" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestAbortUnknownJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	code, _ := callAbort(t, srv.URL, "missing-job", http.MethodPost)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
}

func TestAbortWrongMethod(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-method", "queued", "")

	code, _ := callAbort(t, srv.URL, "abort-method", http.MethodGet)
	if code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", code)
	}
}

func TestAbortSendsCallback(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	callback := newCallbackRecorder(t)
	g, srv := testGatewayAndServer(t, mock.url())

	seedAbortJob(t, g, "abort-callback", "queued", callback.url())

	code, body := callAbort(t, srv.URL, "abort-callback", http.MethodPost)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, string(body))
	}

	select {
	case cb := <-callback.ch:
		if cb.JobID != "abort-callback" {
			t.Fatalf("callback job mismatch: %q", cb.JobID)
		}
		if cb.Status != "cancelled" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
		if cb.Model != "openai/gpt-5.4" {
			t.Fatalf("callback model mismatch: %q", cb.Model)
		}
		if cb.Mode != "run" {
			t.Fatalf("callback mode mismatch: %q", cb.Mode)
		}
		if len(cb.Format) != 0 {
			t.Fatalf("callback format should be empty, got: %s", string(cb.Format))
		}
		if cb.Error != "aborted by user" {
			t.Fatalf("callback error mismatch: %q", cb.Error)
		}
		if cb.CompletedAt == "" {
			t.Fatalf("callback missing completed_at")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for abort callback")
	}
}
