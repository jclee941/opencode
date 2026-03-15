package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDecodeSSEEventsSessionStatus(t *testing.T) {
	input := strings.Join([]string{
		`data: {"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses-1","status":"idle"}}}`,
		"",
		`data: {"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses-1","status":"busy"}}}`,
		"",
	}, "\n")

	var events []eventEnvelope
	err := decodeSSEEvents(strings.NewReader(input), func(env eventEnvelope) bool {
		events = append(events, env)
		return true
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	var first sessionStatusProps
	if err := unmarshalStatus(events[0], &first); err != nil {
		t.Fatalf("failed to decode first status props: %v", err)
	}
	if first.SessionID != "ses-1" || first.Status != "idle" {
		t.Fatalf("unexpected first status payload: %+v", first)
	}

	var second sessionStatusProps
	if err := unmarshalStatus(events[1], &second); err != nil {
		t.Fatalf("failed to decode second status props: %v", err)
	}
	if second.SessionID != "ses-1" || second.Status != "busy" {
		t.Fatalf("unexpected second status payload: %+v", second)
	}
}

func TestDecodeSSEEventsHeartbeat(t *testing.T) {
	input := strings.Join([]string{
		":",
		": keep-alive",
		"",
	}, "\n")

	count := 0
	err := decodeSSEEvents(strings.NewReader(input), func(eventEnvelope) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 dispatched events, got %d", count)
	}
}

func TestDecodeSSEEventsMultiLineData(t *testing.T) {
	input := strings.Join([]string{
		`data: {"directory":"/tmp","payload":`,
		`data: {"type":"message.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
	}, "\n")

	count := 0
	err := decodeSSEEvents(strings.NewReader(input), func(env eventEnvelope) bool {
		count++
		if env.Payload.Type != "message.updated" {
			t.Fatalf("expected message.updated, got %q", env.Payload.Type)
		}
		if got := extractSessionID(env); got != "ses-1" {
			t.Fatalf("expected ses-1, got %q", got)
		}
		return true
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestDecodeSSEEventsBlankLineFlush(t *testing.T) {
	input := strings.Join([]string{
		`data: {"directory":"/tmp/a","payload":{"type":"message.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
		`data: {"directory":"/tmp/b","payload":{"type":"message.part.updated","properties":{"sessionID":"ses-2"}}}`,
		"",
	}, "\n")

	var sessions []string
	err := decodeSSEEvents(strings.NewReader(input), func(env eventEnvelope) bool {
		sessions = append(sessions, extractSessionID(env))
		return true
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 flushed events, got %d", len(sessions))
	}
	if sessions[0] != "ses-1" || sessions[1] != "ses-2" {
		t.Fatalf("unexpected session IDs: %v", sessions)
	}
}

func TestDecodeSSEEventsMalformedJSON(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}
	os.Stderr = w

	input := strings.Join([]string{
		`data: {"directory":"/tmp","payload":`,
		"",
		`data: {"directory":"/tmp","payload":{"type":"message.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
	}, "\n")

	dispatched := 0
	decodeErr := decodeSSEEvents(strings.NewReader(input), func(eventEnvelope) bool {
		dispatched++
		return true
	})
	_ = w.Close()
	os.Stderr = oldStderr

	logged, _ := io.ReadAll(r)
	_ = r.Close()

	if decodeErr != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", decodeErr)
	}
	if dispatched != 1 {
		t.Fatalf("expected decoder to continue and dispatch 1 event, got %d", dispatched)
	}
	if !strings.Contains(string(logged), "sse event decode failed") {
		t.Fatalf("expected malformed event log, got %q", string(logged))
	}
}

func TestDecodeSSEEventsHandlerStop(t *testing.T) {
	input := strings.Join([]string{
		`data: {"directory":"/tmp/1","payload":{"type":"message.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
		`data: {"directory":"/tmp/2","payload":{"type":"message.updated","properties":{"sessionID":"ses-2"}}}`,
		"",
	}, "\n")

	seen := 0
	err := decodeSSEEvents(strings.NewReader(input), func(eventEnvelope) bool {
		seen++
		return false
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if seen != 1 {
		t.Fatalf("expected stop after 1 event, got %d", seen)
	}
}

func TestDecodeSSEEventsMultipleEvents(t *testing.T) {
	input := strings.Join([]string{
		`data: {"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses-1","status":"busy"}}}`,
		"",
		`data: {"directory":"/tmp","payload":{"type":"message.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
		`data: {"directory":"/tmp","payload":{"type":"todo.updated","properties":{"sessionID":"ses-1"}}}`,
		"",
		`data: {"directory":"/tmp","payload":{"type":"file.edited","properties":{"sessionID":"ses-1"}}}`,
		"",
	}, "\n")

	var types []string
	err := decodeSSEEvents(strings.NewReader(input), func(env eventEnvelope) bool {
		types = append(types, env.Payload.Type)
		return true
	})
	if err != nil {
		t.Fatalf("decodeSSEEvents returned error: %v", err)
	}
	if len(types) != 4 {
		t.Fatalf("expected 4 events, got %d", len(types))
	}
	expected := []string{"session.status", "message.updated", "todo.updated", "file.edited"}
	for i := range expected {
		if types[i] != expected[i] {
			t.Fatalf("unexpected event order at index %d: got=%q want=%q", i, types[i], expected[i])
		}
	}
}

func TestExtractSessionIDStatus(t *testing.T) {
	env := eventEnvelope{
		Payload: eventPayload{
			Type:       "session.status",
			Properties: []byte(`{"sessionID":"ses-99","status":"idle"}`),
		},
	}
	if got := extractSessionID(env); got != "ses-99" {
		t.Fatalf("expected ses-99, got %q", got)
	}
}

func TestExtractSessionIDMessage(t *testing.T) {
	types := []string{"message.updated", "message.part.updated"}
	for _, eventType := range types {
		env := eventEnvelope{
			Payload: eventPayload{
				Type:       eventType,
				Properties: []byte(`{"sessionID":"ses-42"}`),
			},
		}
		if got := extractSessionID(env); got != "ses-42" {
			t.Fatalf("type=%s expected ses-42, got %q", eventType, got)
		}
	}
}

func TestExtractSessionIDUnknown(t *testing.T) {
	env := eventEnvelope{
		Payload: eventPayload{
			Type:       "server.heartbeat",
			Properties: []byte(`{"sessionID":"ses-1"}`),
		},
	}
	if got := extractSessionID(env); got != "" {
		t.Fatalf("expected empty session id for unknown type, got %q", got)
	}
}

func unmarshalStatus(env eventEnvelope, out *sessionStatusProps) error {
	if env.Payload.Type != "session.status" {
		return io.EOF
	}
	return jsonUnmarshal(env.Payload.Properties, out)
}

func jsonUnmarshal(data []byte, v any) error {
	return unmarshalJSON(data, v)
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
