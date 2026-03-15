package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestValidateFormatNil(t *testing.T) {
	if err := validateFormat(nil); err != nil {
		t.Fatalf("validateFormat(nil) returned error: %v", err)
	}
}

func TestValidateFormatEmpty(t *testing.T) {
	if err := validateFormat(json.RawMessage{}); err != nil {
		t.Fatalf("validateFormat(empty) returned error: %v", err)
	}
}

func TestValidateFormatValidObject(t *testing.T) {
	raw := json.RawMessage(`{"type":"json_schema","schema":{"type":"object"}}`)
	if err := validateFormat(raw); err != nil {
		t.Fatalf("validateFormat(valid object) returned error: %v", err)
	}
}

func TestValidateFormatInvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{"type":`)
	err := validateFormat(raw)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJobsWithFormatStoredOnJob(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	format := json.RawMessage(`{"type":"json_schema","schema":{"type":"object","properties":{"answer":{"type":"string"}}}}`)
	req := jobRequest{
		JobID:       "job-format-stored",
		Prompt:      "return structured output",
		CallbackURL: "http://127.0.0.1:1/callback",
		Format:      format,
	}

	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	completed := waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)
	if !jsonDeepEqual(t, completed.Format, format) {
		t.Fatalf("stored format mismatch: got=%s want=%s", string(completed.Format), string(format))
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !jsonDeepEqual(t, mock.lastFormat, format) {
		t.Fatalf("forwarded format mismatch: got=%s want=%s", string(mock.lastFormat), string(format))
	}
}

func TestPostJobsWithInvalidFormatReturnsBadRequest(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	badBody := `{"job_id":"job-format-invalid","prompt":"x","callback_url":"http://127.0.0.1:1/callback","format":{"type":}`
	resp, err := http.Post(srv.URL+"/jobs", "application/json", bytes.NewBufferString(badBody))
	if err != nil {
		t.Fatalf("POST /jobs failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetJobByIDIncludesFormatField(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	format := json.RawMessage(`{"type":"json_schema","schema":{"type":"object"}}`)
	req := jobRequest{
		JobID:       "job-format-get",
		Prompt:      "return structured output",
		CallbackURL: "http://127.0.0.1:1/callback",
		Format:      format,
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	_ = waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)

	code, body = getJSON(t, srv.URL+"/jobs/"+req.JobID)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !strings.Contains(string(body), `"format"`) {
		t.Fatalf("expected format field in response body: %s", strings.TrimSpace(string(body)))
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode job response: %v", err)
	}
	if _, ok := decoded["format"]; !ok {
		t.Fatalf("format field missing in decoded response")
	}
}

func jsonDeepEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var av any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("unmarshal left JSON: %v", err)
	}
	var bv any
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("unmarshal right JSON: %v", err)
	}

	ab, err := json.Marshal(av)
	if err != nil {
		t.Fatalf("marshal left JSON: %v", err)
	}
	bb, err := json.Marshal(bv)
	if err != nil {
		t.Fatalf("marshal right JSON: %v", err)
	}
	return bytes.Equal(ab, bb)
}
