package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type eventEnvelope struct {
	Directory string       `json:"directory"`
	Payload   eventPayload `json:"payload"`
}

type eventPayload struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

type sessionStatusProps struct {
	SessionID string `json:"sessionID"`
	Status    string `json:"status"`
}

type messageProps struct {
	SessionID string `json:"sessionID"`
}

func decodeSSEEvents(r io.Reader, handler func(eventEnvelope) bool) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	var dataBuffer bytes.Buffer

	flush := func() bool {
		if dataBuffer.Len() == 0 {
			return true
		}

		var env eventEnvelope
		if err := json.Unmarshal(dataBuffer.Bytes(), &env); err != nil {
			fmt.Fprintf(os.Stderr, "sse event decode failed: %v\n", err)
			dataBuffer.Reset()
			return true
		}
		dataBuffer.Reset()

		return handler(env)
	}

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "data: "):
			dataBuffer.WriteString(strings.TrimPrefix(line, "data: "))
		case strings.HasPrefix(line, ":"):
			continue
		case line == "":
			if !flush() {
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	if !flush() {
		return nil
	}

	return nil
}

func extractSessionID(env eventEnvelope) string {
	switch env.Payload.Type {
	case "session.status":
		var props sessionStatusProps
		if err := json.Unmarshal(env.Payload.Properties, &props); err != nil {
			return ""
		}
		return props.SessionID
	case "message.updated", "message.part.updated", "todo.updated", "file.edited":
		var props messageProps
		if err := json.Unmarshal(env.Payload.Properties, &props); err != nil {
			return ""
		}
		return props.SessionID
	default:
		return ""
	}
}
