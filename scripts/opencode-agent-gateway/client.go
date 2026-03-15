package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func hasFormat(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) > 0
}

func firstFormat(formats []json.RawMessage) json.RawMessage {
	if len(formats) == 0 {
		return nil
	}
	return formats[0]
}

type opencodeClient struct {
	baseURL    string
	password   string
	httpClient *http.Client
}

func newOpencodeClient(baseURL, password string, httpClient *http.Client) *opencodeClient {
	client := httpClient
	if client == nil {
		client = &http.Client{}
	}
	return &opencodeClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		password:   password,
		httpClient: client,
	}
}

func (c *opencodeClient) createSession(ctx context.Context, title string) (string, error) {
	body, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		return "", err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, c.baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create session failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("create session decode failed: %w", err)
	}
	if strings.TrimSpace(payload.ID) == "" {
		return "", fmt.Errorf("create session response missing id")
	}
	return payload.ID, nil
}

func (c *opencodeClient) sendMessage(ctx context.Context, sessionID, prompt, providerID, modelID string, format ...json.RawMessage) (string, error) {
	formatPayload := firstFormat(format)
	requestBody := map[string]any{
		"parts":      []map[string]string{{"type": "text", "text": prompt}},
		"providerID": providerID,
		"modelID":    modelID,
	}
	if hasFormat(formatPayload) {
		requestBody["format"] = formatPayload
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	endpoint := c.baseURL + "/session/" + sessionID + "/message"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("send message failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	result := extractResultText(data)
	if strings.TrimSpace(result) == "" {
		result = strings.TrimSpace(string(data))
	}
	return result, nil
}

func (c *opencodeClient) promptAsync(ctx context.Context, sessionID, prompt, providerID, modelID string, format ...json.RawMessage) error {
	formatPayload := firstFormat(format)
	requestBody := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": prompt}},
		"model": map[string]string{
			"providerID": providerID,
			"modelID":    modelID,
		},
	}
	if hasFormat(formatPayload) {
		requestBody["format"] = formatPayload
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	endpoint := c.baseURL + "/session/" + sessionID + "/prompt_async"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("prompt async failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("prompt async failed: expected status=204 got=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	return nil
}

func (c *opencodeClient) connectEvents(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/event", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("connect events failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	return resp.Body, nil
}

func (c *opencodeClient) getSessionStatus(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/session/status", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get session status failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("get session status decode failed: %w", err)
	}

	statusBySession := make(map[string]string, len(raw))
	for sessionID, value := range raw {
		switch typed := value.(type) {
		case string:
			statusBySession[sessionID] = strings.TrimSpace(typed)
		case map[string]any:
			if status, ok := typed["status"].(string); ok {
				statusBySession[sessionID] = strings.TrimSpace(status)
			}
		}
	}

	return statusBySession, nil
}

func (c *opencodeClient) checkReachable(parent context.Context) bool {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/session", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health request build error: %v\n", err)
		return false
	}
	req.SetBasicAuth("opencode", c.password)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health request failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// getSessionMessages retrieves the message history for a session and returns
// the last assistant message text. This is used by async job completion to
// retrieve the actual AI response instead of using a placeholder.
func (c *opencodeClient) getSessionMessages(ctx context.Context, sessionID string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+"/session/"+sessionID+"/messages", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("opencode", c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("get session messages failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	// Response is an array of MessageWithRole objects: [{"role":"user","content":"..."}, {"role":"assistant","content":"..."}]
	var messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &messages); err != nil {
		return "", fmt.Errorf("get session messages decode failed: %w", err)
	}

	// Walk backwards to find the last assistant message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "assistant" {
			continue
		}
		// Content may be a string or structured object with parts
		var text string
		if err := json.Unmarshal(messages[i].Content, &text); err == nil && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text), nil
		}
		// Try extracting from parts structure (same as extractResultText)
		result := extractResultText(messages[i].Content)
		if strings.TrimSpace(result) != "" {
			return result, nil
		}
	}

	return "", nil
}
