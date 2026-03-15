package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockAsyncOpencode struct {
	server *httptest.Server

	mu                    sync.Mutex
	healthStatusCode      int
	promptAsyncStatusCode int
	sessionsCreated       int
	messageCalls          int
	promptAsyncCalls      int
	lastSessionID         string
	messageBody           string
	sessionStatus         map[string]string
	subscribers           map[int]chan string
	nextSubscriberID      int
}

func newMockAsyncOpencode(t *testing.T) *mockAsyncOpencode {
	t.Helper()
	m := &mockAsyncOpencode{
		healthStatusCode:      http.StatusOK,
		promptAsyncStatusCode: http.StatusNoContent,
		messageBody:           `{"parts":[{"type":"text","text":"sync result"}]}`,
		sessionStatus:         make(map[string]string),
		subscribers:           make(map[int]chan string),
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
			m.lastSessionID = sessionID
			m.sessionStatus[sessionID] = "idle"
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": sessionID})
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/session/") && strings.HasSuffix(r.URL.Path, "/message"):
			m.mu.Lock()
			m.messageCalls++
			body := m.messageBody
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/session/") && strings.HasSuffix(r.URL.Path, "/prompt_async"):
			m.mu.Lock()
			m.promptAsyncCalls++
			statusCode := m.promptAsyncStatusCode
			m.mu.Unlock()

			w.WriteHeader(statusCode)
			if statusCode != http.StatusNoContent {
				_, _ = w.Write([]byte("prompt failed"))
			}
			return
		case r.Method == http.MethodGet && r.URL.Path == "/event":
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			id, ch := m.addSubscriber()
			defer m.removeSubscriber(id)
			flusher.Flush()

			for {
				select {
				case <-r.Context().Done():
					return
				case msg := <-ch:
					_, _ = io.WriteString(w, msg)
					flusher.Flush()
				}
			}
		case r.Method == http.MethodGet && r.URL.Path == "/session/status":
			m.mu.Lock()
			payload := make(map[string]map[string]string, len(m.sessionStatus))
			for sessionID, status := range m.sessionStatus {
				payload[sessionID] = map[string]string{"status": status}
			}
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockAsyncOpencode) url() string {
	return m.server.URL
}

func (m *mockAsyncOpencode) addSubscriber() (int, chan string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextSubscriberID
	m.nextSubscriberID++
	ch := make(chan string, 16)
	m.subscribers[id] = ch
	return id, ch
}

func (m *mockAsyncOpencode) removeSubscriber(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscribers, id)
}

func (m *mockAsyncOpencode) waitForPromptAsyncCalls(t *testing.T, want int, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		calls := m.promptAsyncCalls
		sessionID := m.lastSessionID
		m.mu.Unlock()
		if calls >= want && strings.TrimSpace(sessionID) != "" {
			return sessionID
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("prompt_async calls did not reach %d within %s", want, timeout)
	return ""
}

func (m *mockAsyncOpencode) waitForSubscriberCount(t *testing.T, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		count := len(m.subscribers)
		m.mu.Unlock()
		if count >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("subscriber count did not reach %d within %s", want, timeout)
}

func (m *mockAsyncOpencode) setPromptAsyncStatusCode(statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.promptAsyncStatusCode = statusCode
}

func (m *mockAsyncOpencode) counts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.messageCalls, m.promptAsyncCalls
}

func (m *mockAsyncOpencode) sendSessionStatus(t *testing.T, sessionID, status string) {
	t.Helper()
	props, err := json.Marshal(sessionStatusProps{SessionID: sessionID, Status: status})
	if err != nil {
		t.Fatalf("marshal session status props: %v", err)
	}
	env, err := json.Marshal(eventEnvelope{Payload: eventPayload{Type: "session.status", Properties: props}})
	if err != nil {
		t.Fatalf("marshal event envelope: %v", err)
	}

	m.mu.Lock()
	m.sessionStatus[sessionID] = status
	subscribers := make([]chan string, 0, len(m.subscribers))
	for _, ch := range m.subscribers {
		subscribers = append(subscribers, ch)
	}
	m.mu.Unlock()

	message := "data: " + string(env) + "\n\n"
	for _, ch := range subscribers {
		ch <- message
	}
}

func TestAsyncJobLifecycle(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "async-job-lifecycle",
		Prompt:      "do async work",
		Mode:        "async",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	sessionID := mock.waitForPromptAsyncCalls(t, 1, 2*time.Second)
	mock.waitForSubscriberCount(t, 1, 2*time.Second)
	mock.sendSessionStatus(t, sessionID, "busy")
	mock.sendSessionStatus(t, sessionID, "idle")

	completed := waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)
	if completed.Result != asyncCompletionResult {
		t.Fatalf("unexpected async result: %q", completed.Result)
	}
	if completed.SessionID != sessionID {
		t.Fatalf("unexpected session id: %q", completed.SessionID)
	}

	select {
	case cb := <-callback.ch:
		if cb.JobID != req.JobID {
			t.Fatalf("callback job mismatch: %q", cb.JobID)
		}
		if cb.Status != "completed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
		if cb.Model != "openai/gpt-5.4" {
			t.Fatalf("callback model mismatch: %q", cb.Model)
		}
		if cb.Mode != "async" {
			t.Fatalf("callback mode mismatch: %q", cb.Mode)
		}
		if len(cb.Format) != 0 {
			t.Fatalf("callback format should be empty, got: %s", string(cb.Format))
		}
		if cb.Result != asyncCompletionResult {
			t.Fatalf("callback result mismatch: %q", cb.Result)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for callback")
	}
}

func TestAsyncJobFallbackSync(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "sync-job-default-mode",
		Prompt:      "do sync work",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	completed := waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)
	if completed.Result != "sync result" {
		t.Fatalf("unexpected sync result: %q", completed.Result)
	}

	messageCalls, promptAsyncCalls := mock.counts()
	if messageCalls != 1 {
		t.Fatalf("expected 1 message call, got %d", messageCalls)
	}
	if promptAsyncCalls != 0 {
		t.Fatalf("expected 0 prompt_async calls, got %d", promptAsyncCalls)
	}

	select {
	case cb := <-callback.ch:
		if cb.Status != "completed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for callback")
	}
}

func TestAsyncJobPromptAsyncFailure(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	mock.setPromptAsyncStatusCode(http.StatusInternalServerError)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "async-job-prompt-failure",
		Prompt:      "do async work",
		Mode:        "async",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	failed := waitForJobStatus(t, srv.URL, req.JobID, "failed", 2*time.Second)
	if !strings.Contains(failed.Error, "prompt async failed") {
		t.Fatalf("unexpected failure error: %q", failed.Error)
	}

	_, promptAsyncCalls := mock.counts()
	if promptAsyncCalls != 1 {
		t.Fatalf("expected 1 prompt_async call, got %d", promptAsyncCalls)
	}

	select {
	case cb := <-callback.ch:
		if cb.Status != "failed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
		if !strings.Contains(cb.Error, "prompt async failed") {
			t.Fatalf("callback error mismatch: %q", cb.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for callback")
	}
}

func TestAsyncJobSSEIdleAfterBusy(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "async-job-idle-after-busy",
		Prompt:      "watch sse",
		Mode:        "async",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	sessionID := mock.waitForPromptAsyncCalls(t, 1, 2*time.Second)
	mock.waitForSubscriberCount(t, 1, 2*time.Second)
	mock.sendSessionStatus(t, sessionID, "busy")
	mock.sendSessionStatus(t, sessionID, "idle")

	completed := waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)
	if completed.Result != asyncCompletionResult {
		t.Fatalf("unexpected async result: %q", completed.Result)
	}

	select {
	case cb := <-callback.ch:
		if cb.Status != "completed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for callback")
	}
}

func TestAsyncJobCallbackOnce(t *testing.T) {
	mock := newMockAsyncOpencode(t)
	callback := newCallbackRecorder(t)
	_, srv := testGatewayAndServer(t, mock.url())

	req := jobRequest{
		JobID:       "async-job-callback-once",
		Prompt:      "complete once",
		Mode:        "async",
		CallbackURL: callback.url(),
	}
	code, body := postJSON(t, srv.URL+"/jobs", req)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", code, strings.TrimSpace(string(body)))
	}

	sessionID := mock.waitForPromptAsyncCalls(t, 1, 2*time.Second)
	mock.waitForSubscriberCount(t, 1, 2*time.Second)
	mock.sendSessionStatus(t, sessionID, "busy")
	mock.sendSessionStatus(t, sessionID, "idle")

	_ = waitForJobStatus(t, srv.URL, req.JobID, "completed", 2*time.Second)

	select {
	case cb := <-callback.ch:
		if cb.Status != "completed" {
			t.Fatalf("callback status mismatch: %q", cb.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for first callback")
	}

	select {
	case cb := <-callback.ch:
		t.Fatalf("unexpected duplicate callback: %+v", cb)
	case <-time.After(250 * time.Millisecond):
	}
}
