package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type capturedPayload struct {
	SessionID      string                `json:"sessionId"`
	Message        string                `json:"message"`
	InlineKeyboard [][]map[string]string `json:"inlineKeyboard"`
}

func TestExtractVisibleMenuOptions(t *testing.T) {
	input := "\x1b[2J\r\nAdd account\r\nCheck quotas\r\nVerify one account\r\nVerify all accounts\r\nGemini CLI Login\r\nConfigure models in opencode.json\r\n"
	options := extractVisibleMenuOptions(input)
	if len(options) != 6 {
		t.Fatalf("expected 6 options, got %d", len(options))
	}
	if options[0].Label != "Add account" || options[0].Input != "numpad:1" {
		t.Fatalf("unexpected first option: %+v", options[0])
	}
	if options[5].Label != "Configure models in opencode.json" || options[5].Input != "numpad:6" {
		t.Fatalf("unexpected last option: %+v", options[5])
	}
}

func TestExtractVisibleMenuOptionsIgnoresNonMenuLines(t *testing.T) {
	input := "Google login\nhttps://accounts.google.com/o/oauth2/auth?x=1\nsession: live1\n"
	options := extractVisibleMenuOptions(input)
	if len(options) != 0 {
		t.Fatalf("expected no menu options, got %+v", options)
	}
}

func TestCallbackStorePopAfterPreservesOrder(t *testing.T) {
	store := newCallbackStore()
	store.put("s1", "first")
	time.Sleep(2 * time.Millisecond)
	store.put("s1", "second")

	v1, _, seq1, ok := store.popAfter("s1", 0)
	if !ok || v1 != "first" {
		t.Fatalf("expected first callback, got %q ok=%v", v1, ok)
	}
	v2, _, seq2, ok := store.popAfter("s1", seq1)
	if !ok || v2 != "second" {
		t.Fatalf("expected second callback, got %q ok=%v", v2, ok)
	}
	if seq2 <= seq1 {
		t.Fatalf("expected increasing seq, got %d then %d", seq1, seq2)
	}
}

func TestHandleTelegramMessageNumpadMapsToArrowSequence(t *testing.T) {
	store := newCallbackStore()
	state := &sessionState{}
	state.set("live1")
	if !handleTelegramMessage("cb:live1:numpad:3", store, state) {
		t.Fatalf("expected callback to be handled")
	}
	value, _, ok := store.get("live1")
	if !ok {
		t.Fatalf("expected stored callback")
	}
	if value != "\x1b[B\x1b[B\r" {
		t.Fatalf("unexpected mapped value: %q", value)
	}
}

func TestHandleTelegramMessageNavButtonsMapToArrowKeys(t *testing.T) {
	store := newCallbackStore()
	state := &sessionState{}
	state.set("live1")
	if !handleTelegramMessage("cb:live1:nav:down", store, state) {
		t.Fatalf("expected nav down callback to be handled")
	}
	value, _, ok := store.get("live1")
	if !ok || value != "\x1b[B" {
		t.Fatalf("unexpected nav down mapping: %q ok=%v", value, ok)
	}
	if !handleTelegramMessage("cb:live1:nav:up", store, state) {
		t.Fatalf("expected nav up callback to be handled")
	}
	value, _, ok = store.get("live1")
	if !ok || value != "\x1b[A" {
		t.Fatalf("unexpected nav up mapping: %q ok=%v", value, ok)
	}
}

func TestForwardStreamEmitsDynamicMenuPayloads(t *testing.T) {
	var mu sync.Mutex
	payloads := []capturedPayload{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload capturedPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		mu.Lock()
		payloads = append(payloads, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	reader, writer := io.Pipe()
	var dst bytes.Buffer
	done := make(chan struct{})
	go func() {
		forwardStream(&dst, reader, server.Client(), "", "", "", server.URL, "sess1", "Google login", nil)
		close(done)
	}()

	_, _ = writer.Write([]byte("Login method\nOAuth with Google\nManually enter API Key\n"))
	time.Sleep(200 * time.Millisecond)
	_, _ = writer.Write([]byte("Add account\nCheck quotas\nVerify one account\nVerify all accounts\nGemini CLI Login\nConfigure models in opencode.json\n"))
	_ = writer.Close()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) < 2 {
		t.Fatalf("expected at least 2 payloads, got %d", len(payloads))
	}
	first := payloads[0].InlineKeyboard
	if len(first) == 0 || len(first[0]) < 2 {
		t.Fatalf("expected first dynamic keyboard row, got %+v", first)
	}
	if first[0][0]["text"] != "OAuth with Google" || first[0][1]["text"] != "Manually enter API Key" {
		t.Fatalf("unexpected first-stage keyboard: %+v", first)
	}
	last := payloads[len(payloads)-1].InlineKeyboard
	joined := ""
	for _, row := range last {
		for _, btn := range row {
			joined += btn["text"] + "|"
		}
	}
	for _, expected := range []string{"Add account", "Check quotas", "Verify one account", "Verify all accounts", "Gemini CLI Login", "Configure models in opencode.json"} {
		if !bytes.Contains([]byte(joined), []byte(expected)) {
			t.Fatalf("missing %q in second-stage keyboard: %s", expected, joined)
		}
	}
}

func TestBuildPayloadFromStateUsesLatestLogWindow(t *testing.T) {
	state := &sessionState{}
	state.set("sess1")
	state.setWindow("Add account\nCheck quotas\nVerify one account\n")
	payload := buildPayloadFromState(state, "Google login")
	if payload == nil {
		t.Fatalf("expected payload from state")
	}
	if len(payload.InlineKeyboard) == 0 || payload.InlineKeyboard[0][0]["text"] != "Add account" {
		t.Fatalf("unexpected payload keyboard: %+v", payload.InlineKeyboard)
	}
}

func TestHumanizeCallbackValue(t *testing.T) {
	if got := humanizeCallbackValue("cb:s1:nav:down"); got != "down" {
		t.Fatalf("expected down, got %q", got)
	}
	if got := humanizeCallbackValue("cb:s1:__ENTER__"); got != "select" {
		t.Fatalf("expected select, got %q", got)
	}
}
