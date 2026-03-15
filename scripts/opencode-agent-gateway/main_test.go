package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockOpencode struct {
	server       *httptest.Server
	messageBody  string
	messageDelay time.Duration

	mu               sync.Mutex
	healthStatusCode int
	sessionsCreated  int
	messageCalls     int
	lastProviderID   string
	lastModelID      string
	lastPrompt       string
	lastFormat       json.RawMessage
}

func newMockOpencode(t *testing.T, healthStatusCode int, messageBody string, messageDelay time.Duration) *mockOpencode {
	t.Helper()
	m := &mockOpencode{
		healthStatusCode: healthStatusCode,
		messageBody:      messageBody,
		messageDelay:     messageDelay,
	}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			w.WriteHeader(m.healthStatusCode)
			_, _ = w.Write([]byte("{}"))
			return
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			m.mu.Lock()
			m.sessionsCreated++
			sessionID := "session-" + strconvItoa(m.sessionsCreated)
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": sessionID})
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/session/") && strings.HasSuffix(r.URL.Path, "/message"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()

			var payload struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				ProviderID string          `json:"providerID"`
				ModelID    string          `json:"modelID"`
				Format     json.RawMessage `json:"format"`
			}
			_ = json.Unmarshal(body, &payload)

			m.mu.Lock()
			m.messageCalls++
			m.lastProviderID = payload.ProviderID
			m.lastModelID = payload.ModelID
			if len(payload.Parts) > 0 {
				m.lastPrompt = payload.Parts[0].Text
			}
			if len(payload.Format) > 0 {
				m.lastFormat = append(json.RawMessage(nil), payload.Format...)
			} else {
				m.lastFormat = nil
			}
			m.mu.Unlock()

			if m.messageDelay > 0 {
				time.Sleep(m.messageDelay)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(m.messageBody))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockOpencode) url() string {
	return m.server.URL
}

type callbackRecorder struct {
	server *httptest.Server
	ch     chan callbackPayload
}

func newCallbackRecorder(t *testing.T) *callbackRecorder {
	t.Helper()
	r := &callbackRecorder{ch: make(chan callbackPayload, 8)}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		defer req.Body.Close()
		var payload callbackPayload
		_ = json.NewDecoder(req.Body).Decode(&payload)
		r.ch <- payload
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(r.server.Close)
	return r
}

func (r *callbackRecorder) url() string {
	return r.server.URL
}

func testGatewayAndServer(t *testing.T, opencodeURL string) (*gateway, *httptest.Server) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	g := newGateway(ctx, config{
		opencodeURL:    opencodeURL,
		opencodePass:   "test-pass",
		defaultModel:   "openai/gpt-5.4",
		maxConcurrent:  2,
		callbackTimout: 2 * time.Second,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/jobs", g.handleJobs)
	mux.HandleFunc("/jobs/", g.handleJobByID)

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		cancel()
		g.wg.Wait()
	})

	return g, srv
}

func postJSON(t *testing.T, url string, payload any) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func getJSON(t *testing.T, url string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func waitForJobStatus(t *testing.T, baseURL string, jobID string, status string, timeout time.Duration) job {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		code, body := getJSON(t, baseURL+"/jobs/"+jobID)
		if code == http.StatusOK {
			var got job
			if err := json.Unmarshal(body, &got); err == nil && got.Status == status {
				return got
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach status %q within %s", jobID, status, timeout)
	return job{}
}

func TestHandleHealth(t *testing.T) {
	t.Run("reachable", func(t *testing.T) {
		mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
		_, srv := testGatewayAndServer(t, mock.url())

		code, body := getJSON(t, srv.URL+"/health")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var payload struct {
			Status            string `json:"status"`
			OpencodeReachable bool   `json:"opencode_reachable"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode health payload: %v", err)
		}
		if payload.Status != "ok" {
			t.Fatalf("unexpected status: %q", payload.Status)
		}
		if !payload.OpencodeReachable {
			t.Fatalf("expected opencode_reachable=true")
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		_, srv := testGatewayAndServer(t, "http://127.0.0.1:1")
		code, body := getJSON(t, srv.URL+"/health")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		var payload struct {
			Status            string `json:"status"`
			OpencodeReachable bool   `json:"opencode_reachable"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode health payload: %v", err)
		}
		if payload.Status != "ok" {
			t.Fatalf("unexpected status: %q", payload.Status)
		}
		if payload.OpencodeReachable {
			t.Fatalf("expected opencode_reachable=false")
		}
	})
}

func TestPostJobsValidation(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	tests := []struct {
		name    string
		payload jobRequest
	}{
		{name: "missing job_id", payload: jobRequest{Prompt: "hello", CallbackURL: "http://cb"}},
		{name: "missing prompt", payload: jobRequest{JobID: "job-1", CallbackURL: "http://cb"}},
		{name: "missing callback_url", payload: jobRequest{JobID: "job-1", Prompt: "hello"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := postJSON(t, srv.URL+"/jobs", tc.payload)
			if code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", code)
			}
		})
	}
}

func TestPostJobsValidLifecycleAndCallback(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"final answer"}]}`, 120*time.Millisecond)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "job-lifecycle-1",
		Prompt:      "solve this",
		Model:       "openai/gpt-5.4",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}
	var accepted struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &accepted); err != nil {
		t.Fatalf("decode accepted body: %v", err)
	}
	if accepted.JobID != req.JobID || accepted.Status != "queued" {
		t.Fatalf("unexpected accepted payload: %+v", accepted)
	}

	seenRunning := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, body := getJSON(t, srv.URL+"/jobs/"+req.JobID)
		var got job
		if err := json.Unmarshal(body, &got); err == nil {
			if got.Status == "running" {
				seenRunning = true
			}
			if got.Status == "completed" {
				if !seenRunning {
					t.Fatalf("expected to observe running status before completed")
				}
				if got.Result != "final answer" {
					t.Fatalf("unexpected result: %q", got.Result)
				}
				if strings.TrimSpace(got.SessionID) == "" {
					t.Fatalf("expected non-empty session_id")
				}
				break
			}
		}
		time.Sleep(15 * time.Millisecond)
	}

	completed := waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)
	if completed.Result != "final answer" {
		t.Fatalf("unexpected completed result: %q", completed.Result)
	}

	select {
	case cb := <-callback.ch:
		if cb.JobID != req.JobID {
			t.Fatalf("callback job_id mismatch: %q", cb.JobID)
		}
		if cb.Status != "completed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
		if cb.Model != req.Model {
			t.Fatalf("callback model mismatch: %q", cb.Model)
		}
		if cb.Mode != "run" {
			t.Fatalf("callback mode mismatch: %q", cb.Mode)
		}
		if len(cb.Format) != 0 {
			t.Fatalf("callback format should be empty, got: %s", string(cb.Format))
		}
		if cb.Result != "final answer" {
			t.Fatalf("callback result mismatch: %q", cb.Result)
		}
		if strings.TrimSpace(cb.CompletedAt) == "" {
			t.Fatalf("callback missing completed_at")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for callback")
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.sessionsCreated != 1 {
		t.Fatalf("expected 1 created session, got %d", mock.sessionsCreated)
	}
	if mock.messageCalls != 1 {
		t.Fatalf("expected 1 message call, got %d", mock.messageCalls)
	}
	if mock.lastProviderID != "openai" || mock.lastModelID != "gpt-5.4" {
		t.Fatalf("unexpected model split provider=%q model=%q", mock.lastProviderID, mock.lastModelID)
	}
	if mock.lastPrompt != req.Prompt {
		t.Fatalf("unexpected prompt forwarded: %q", mock.lastPrompt)
	}
}

func TestPostJobsIdempotentAndDuplicateJobID(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 50*time.Millisecond)
	_, srv := testGatewayAndServer(t, mock.url())

	first := jobRequest{
		JobID:          "job-dup-1",
		Prompt:         "first",
		CallbackURL:    "http://127.0.0.1:1/callback",
		IdempotencyKey: "idem-1",
	}
	code, _ := postJSON(t, srv.URL+"/jobs", first)
	if code != http.StatusAccepted {
		t.Fatalf("first create expected 202, got %d", code)
	}

	idempotent := jobRequest{
		JobID:          "job-dup-2",
		Prompt:         "second",
		CallbackURL:    "http://127.0.0.1:1/callback",
		IdempotencyKey: "idem-1",
	}
	code, body := postJSON(t, srv.URL+"/jobs", idempotent)
	if code != http.StatusOK {
		t.Fatalf("idempotent request expected 200, got %d", code)
	}
	var idemResp job
	if err := json.Unmarshal(body, &idemResp); err != nil {
		t.Fatalf("decode idempotent response: %v", err)
	}
	if idemResp.ID != first.JobID {
		t.Fatalf("expected idempotent response for %q, got %q", first.JobID, idemResp.ID)
	}

	dup := jobRequest{
		JobID:       "job-dup-1",
		Prompt:      "duplicate by id",
		CallbackURL: "http://127.0.0.1:1/callback",
	}
	code, body = postJSON(t, srv.URL+"/jobs", dup)
	if code != http.StatusOK {
		t.Fatalf("duplicate job_id request expected 200, got %d", code)
	}
	var dupResp job
	if err := json.Unmarshal(body, &dupResp); err != nil {
		t.Fatalf("decode duplicate response: %v", err)
	}
	if dupResp.ID != first.JobID {
		t.Fatalf("expected duplicate response for existing job_id=%q, got %q", first.JobID, dupResp.ID)
	}
}

func TestGetJobsAndGetJobByID(t *testing.T) {
	mock := newMockOpencode(t, http.StatusOK, `{"parts":[{"type":"text","text":"ok"}]}`, 0)
	_, srv := testGatewayAndServer(t, mock.url())

	for i := 1; i <= 2; i++ {
		req := jobRequest{
			JobID:       "list-job-" + strconvItoa(i),
			Prompt:      "prompt",
			CallbackURL: "http://127.0.0.1:1/callback",
		}
		code, _ := postJSON(t, srv.URL+"/jobs", req)
		if code != http.StatusAccepted {
			t.Fatalf("create list job %d expected 202, got %d", i, code)
		}
	}

	code, body := getJSON(t, srv.URL+"/jobs")
	if code != http.StatusOK {
		t.Fatalf("GET /jobs expected 200, got %d", code)
	}
	var summaries []jobSummary
	if err := json.Unmarshal(body, &summaries); err != nil {
		t.Fatalf("decode summaries: %v", err)
	}
	if len(summaries) < 2 {
		t.Fatalf("expected at least 2 jobs, got %d", len(summaries))
	}

	code, body = getJSON(t, srv.URL+"/jobs/list-job-1")
	if code != http.StatusOK {
		t.Fatalf("GET /jobs/{id} expected 200, got %d", code)
	}
	var found job
	if err := json.Unmarshal(body, &found); err != nil {
		t.Fatalf("decode job details: %v", err)
	}
	if found.ID != "list-job-1" {
		t.Fatalf("expected list-job-1, got %q", found.ID)
	}

	code, _ = getJSON(t, srv.URL+"/jobs/missing")
	if code != http.StatusNotFound {
		t.Fatalf("GET /jobs/missing expected 404, got %d", code)
	}
}

func TestResolveValuePassThroughNonOpRef(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fallback  string
		expected  string
		expectErr bool
	}{
		{name: "value set", value: "plain-secret", fallback: "op://vault/item/field", expected: "plain-secret"},
		{name: "fallback plain", value: "", fallback: "fallback-value", expected: "fallback-value"},
		{name: "empty both", value: "", fallback: "", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveValue(tc.value, tc.fallback)
			if (err != nil) != tc.expectErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestSplitModel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantProv  string
		wantModel string
		wantErr   bool
	}{
		{name: "valid", input: "openai/gpt-5.4", wantProv: "openai", wantModel: "gpt-5.4"},
		{name: "trimmed", input: "  anthropic / claude-3.7 ", wantProv: "anthropic", wantModel: "claude-3.7"},
		{name: "missing slash", input: "openai", wantErr: true},
		{name: "empty provider", input: "/gpt", wantErr: true},
		{name: "empty model", input: "openai/", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider, model, err := splitModel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider != tc.wantProv || model != tc.wantModel {
				t.Fatalf("want provider/model %q/%q, got %q/%q", tc.wantProv, tc.wantModel, provider, model)
			}
		})
	}
}

func TestExtractResultText(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "json envelope text parts",
			body: `{"parts":[{"type":"text","text":"hello"},{"type":"text","text":"world"}]}`,
			want: "hello\n\nworld",
		},
		{
			name: "ndjson fallback returns last text",
			body: "{\"parts\":[{\"type\":\"text\",\"text\":\"first\"}]}\n{\"parts\":[{\"type\":\"text\",\"text\":\"second\"}]}",
			want: "second",
		},
		{
			name: "raw fallback",
			body: "not json",
			want: "not json",
		},
		{
			name: "empty",
			body: "  ",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractResultText([]byte(tc.body))
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func strconvItoa(v int) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		return "-" + strconvItoa(-v)
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	return string(buf)
}
