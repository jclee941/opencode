package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientCreateSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ses-123"}`))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	got, err := client.createSession(context.Background(), "agent-gateway: job-1")
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}
	if got != "ses-123" {
		t.Fatalf("expected session id ses-123, got %q", got)
	}
}

func TestClientCreateSessionNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	_, err := client.createSession(context.Background(), "agent-gateway: job-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	errText := err.Error()
	if !strings.Contains(errText, "status=500") || !strings.Contains(errText, "boom") {
		t.Fatalf("unexpected error text: %q", errText)
	}
}

func TestClientCreateSessionEmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":""}`))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	_, err := client.createSession(context.Background(), "agent-gateway: job-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
}

func TestClientSendMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses-1/message" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"parts":[{"type":"text","text":"result"}]}`))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	got, err := client.sendMessage(context.Background(), "ses-1", "prompt", "openai", "gpt-5.4")
	if err != nil {
		t.Fatalf("sendMessage returned error: %v", err)
	}
	if got != "result" {
		t.Fatalf("expected result text, got %q", got)
	}
}

func TestClientSendMessageNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	_, err := client.sendMessage(context.Background(), "ses-1", "prompt", "openai", "gpt-5.4")
	if err == nil {
		t.Fatalf("expected error")
	}
	errText := err.Error()
	if !strings.Contains(errText, "status=503") || !strings.Contains(errText, "unavailable") {
		t.Fatalf("unexpected error text: %q", errText)
	}
}

func TestClientSendMessageRequestShape(t *testing.T) {
	var captured struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"parts":[{"type":"text","text":"ok"}]}`))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	_, err := client.sendMessage(context.Background(), "ses-1", "hello world", "openai", "gpt-5.4")
	if err != nil {
		t.Fatalf("sendMessage returned error: %v", err)
	}

	if captured.ProviderID != "openai" || captured.ModelID != "gpt-5.4" {
		t.Fatalf("unexpected provider/model: %q/%q", captured.ProviderID, captured.ModelID)
	}
	if len(captured.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(captured.Parts))
	}
	if captured.Parts[0].Type != "text" || captured.Parts[0].Text != "hello world" {
		t.Fatalf("unexpected parts payload: type=%q text=%q", captured.Parts[0].Type, captured.Parts[0].Text)
	}
}

func TestClientPromptAsync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses-1/prompt_async" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	err := client.promptAsync(context.Background(), "ses-1", "prompt", "openai", "gpt-5.4")
	if err != nil {
		t.Fatalf("promptAsync returned error: %v", err)
	}
}

func TestClientPromptAsyncNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	err := client.promptAsync(context.Background(), "ses-1", "prompt", "openai", "gpt-5.4")
	if err == nil {
		t.Fatalf("expected error")
	}
	errText := err.Error()
	if !strings.Contains(errText, "status=500") {
		t.Fatalf("unexpected error text: %q", errText)
	}
}

func TestClientPromptAsyncRequestShape(t *testing.T) {
	var captured struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
		Model struct {
			ProviderID string `json:"providerID"`
			ModelID    string `json:"modelID"`
		} `json:"model"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	err := client.promptAsync(context.Background(), "ses-1", "hello world", "openai", "gpt-5.4")
	if err != nil {
		t.Fatalf("promptAsync returned error: %v", err)
	}

	if len(captured.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(captured.Parts))
	}
	if captured.Parts[0].Type != "text" || captured.Parts[0].Text != "hello world" {
		t.Fatalf("unexpected parts payload: type=%q text=%q", captured.Parts[0].Type, captured.Parts[0].Text)
	}
	if captured.Model.ProviderID != "openai" || captured.Model.ModelID != "gpt-5.4" {
		t.Fatalf("unexpected model payload: provider=%q model=%q", captured.Model.ProviderID, captured.Model.ModelID)
	}
}

func TestClientCheckReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/session" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newOpencodeClient(srv.URL, "pass", srv.Client())
	if !client.checkReachable(context.Background()) {
		t.Fatalf("expected reachable")
	}
}

func TestClientCheckReachableDown(t *testing.T) {
	client := newOpencodeClient("http://127.0.0.1:1", "pass", &http.Client{})
	if client.checkReachable(context.Background()) {
		t.Fatalf("expected unreachable")
	}
}
