package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

const (
	defaultBotTokenRef = "op://homelab/telegram/credential"
	defaultBotUserRef  = "op://homelab/telegram/username"
	defaultChatIDRef   = "op://homelab/telegram/chat_id"
)

var (
	ansiRegex         = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	ansiFragmentRegex = regexp.MustCompile(`\[[0-9;?]*[ -/]*[@-~]`)
	urlRegex          = regexp.MustCompile(`https://[^\s"'<>]+`)
	choiceParenRegex  = regexp.MustCompile(`\(([a-zA-Z0-9-]+)\)`)
	prompts           = []string{
		"Paste the redirect URL (or just the code) here:",
		"Paste the authorization code here:",
	}
	promptHints = []string{
		"paste",
		"authorization code",
		"redirect",
		"enter",
		"input",
		"select",
		"choose",
		"confirm",
		"yes/no",
		"y/n",
		"callback",
		"code",
		"url",
		"gemini",
		"cli",
		"login",
		"account",
	}
)

type telegramResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description,omitempty"`
	ResultRaw   json.RawMessage `json:"result,omitempty"`
}

type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message,omitempty"`
	CallbackQuery *struct {
		ID      string `json:"id"`
		Data    string `json:"data,omitempty"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message,omitempty"`
	} `json:"callback_query,omitempty"`
}

type inboundTelegramUpdate struct {
	Message *struct {
		Chat *struct {
			ID int64 `json:"id"`
		} `json:"chat,omitempty"`
		Text string `json:"text"`
	} `json:"message,omitempty"`
	CallbackQuery *struct {
		ID      string `json:"id"`
		Data    string `json:"data,omitempty"`
		Message *struct {
			Chat *struct {
				ID int64 `json:"id"`
			} `json:"chat,omitempty"`
		} `json:"message,omitempty"`
	} `json:"callback_query,omitempty"`
	Text   string `json:"text,omitempty"`
	ChatID string `json:"chatId,omitempty"`
}

type relayPayload struct {
	SessionID         string                `json:"sessionId"`
	Title             string                `json:"title"`
	Message           string                `json:"message"`
	URL               string                `json:"url"`
	Instructions      string                `json:"instructions"`
	InlineKeyboard    [][]map[string]string `json:"inlineKeyboard,omitempty"`
	InlineKeyboardAlt [][]map[string]string `json:"inline_keyboard,omitempty"`
	Choices           []string              `json:"choices,omitempty"`
	ChatID            string                `json:"chat_id,omitempty"`
	Text              string                `json:"text,omitempty"`
	DisablePreview    bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup       map[string]any        `json:"reply_markup,omitempty"`
	ReplyMarkupAlt    map[string]any        `json:"replyMarkup,omitempty"`
	ReplyMarkupJSON   string                `json:"reply_markup_json,omitempty"`
	Telegram          map[string]any        `json:"telegram,omitempty"`
	Data              map[string]any        `json:"data,omitempty"`
}

type menuOption struct {
	Label string
	Input string
}

type callbackStore struct {
	mu      sync.Mutex
	entries map[string][]callbackEntry
	nextSeq uint64
}

type callbackEntry struct {
	value   string
	updated time.Time
	seq     uint64
}

type sessionState struct {
	mu         sync.Mutex
	sessionID  string
	lastWindow string
}

func (s *sessionState) set(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = strings.TrimSpace(sessionID)
}

func (s *sessionState) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID
}

func (s *sessionState) setWindow(window string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastWindow = window
}

func (s *sessionState) getWindow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastWindow
}

func newCallbackStore() *callbackStore {
	return &callbackStore{entries: map[string][]callbackEntry{}}
}

func (s *callbackStore) put(sessionID string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return
	}
	s.entries[trimmed] = append(s.entries[trimmed], callbackEntry{value: value, updated: time.Now().UTC()})
	s.nextSeq++
	last := len(s.entries[trimmed]) - 1
	s.entries[trimmed][last].seq = s.nextSeq
}

func (s *callbackStore) get(sessionID string) (string, time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	queue, ok := s.entries[strings.TrimSpace(sessionID)]
	if !ok || len(queue) == 0 {
		return "", time.Time{}, false
	}
	last := queue[len(queue)-1]
	return last.value, last.updated, true
}

func (s *callbackStore) popAfter(sessionID string, afterSeq uint64) (string, time.Time, uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(sessionID)
	queue, ok := s.entries[key]
	if !ok || len(queue) == 0 {
		return "", time.Time{}, 0, false
	}
	for idx, entry := range queue {
		if afterSeq == 0 || entry.seq > afterSeq {
			s.entries[key] = append(queue[:idx], queue[idx+1:]...)
			if len(s.entries[key]) == 0 {
				delete(s.entries, key)
			}
			return entry.value, entry.updated, entry.seq, true
		}
	}
	return "", time.Time{}, 0, false
}

func (s *callbackStore) clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, strings.TrimSpace(sessionID))
}

func (s *callbackStore) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.entries))
	for key, queue := range s.entries {
		if len(queue) == 0 {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func main() {
	listenAddr := flag.String("listen", envOrDefault("AUTHFLOW_LISTEN", "127.0.0.1:7788"), "HTTP listen address")
	command := flag.String("command", envOrDefault("AUTHFLOW_COMMAND", "opencode auth login -p google"), "command to wrap")
	title := flag.String("title", envOrDefault("AUTHFLOW_TITLE", "Google login"), "Telegram message title")
	sessionFlag := flag.String("session", envOrDefault("AUTHFLOW_SESSION_ID", ""), "fixed session id for callback relay")
	botTokenFlag := flag.String("bot-token", envOrDefault("TELEGRAM_BOT_TOKEN", ""), "Telegram bot token or op:// reference")
	chatIDFlag := flag.String("chat-id", envOrDefault("TELEGRAM_CHAT_ID", ""), "Telegram chat id or op:// reference")
	flag.Parse()

	oauthWebhookURL := strings.TrimSpace(envOrDefault("N8N_TELEGRAM_OAUTH_WEBHOOK_URL", ""))
	eventWebhookURL := strings.TrimSpace(envOrDefault("N8N_TELEGRAM_EVENT_WEBHOOK_URL", ""))

	botFallback := defaultBotTokenRef
	botToken, err := resolveValue(*botTokenFlag, botFallback)
	if err != nil {
		fail(err)
	}
	user, err := resolveValue("", defaultBotUserRef)
	if err != nil {
		user = ""
	}
	chatFallback := defaultChatIDRef
	chatID, err := resolveValue(*chatIDFlag, chatFallback)
	if err != nil {
		fail(err)
	}
	if botToken == "" {
		fail(fmt.Errorf("missing Telegram delivery configuration"))
	}
	if botToken != "" && chatID == "" {
		chatID, err = discoverChatID(botToken, user)
		if err != nil {
			fail(err)
		}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	callbacks := newCallbackStore()
	state := &sessionState{}
	if botToken != "" && chatID != "" {
		go pollTelegramUpdates(client, botToken, chatID, callbacks, state)
	}
	server := &http.Server{Addr: *listenAddr, Handler: router(callbacks, chatID, botToken, client, oauthWebhookURL, eventWebhookURL, *title, state), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	fmt.Printf("auth bridge listening on %s\n", *listenAddr)

	stdin, stdout, stderr, cmd, err := startCommand(normalizeAuthCommand(*command))
	if err != nil {
		fail(err)
	}
	writer := &lockedWriter{w: stdin}
	go func() { _, _ = io.Copy(writer, os.Stdin) }()

	sessionID := strings.TrimSpace(*sessionFlag)
	if sessionID == "" {
		sessionID = newSessionID()
	}
	state.set(sessionID)
	callbacks.clear(sessionID)
	bootstrap := relayPayload{SessionID: sessionID, Title: *title, Message: "Bridge connected. Buttons are active now. Start Google login in browser, then tap matching option buttons here."}
	if isGoogleAuthCommand(*command) {
		bootstrap.Message = "Bridge connected. Press Continue to reveal the live menu, then use the generated buttons."
		bootstrap.Choices = []string{"Continue"}
	}
	_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, bootstrap)
	go drainCallbacks(callbacks, writer, sessionID)
	go forwardStream(os.Stdout, stdout, client, botToken, chatID, oauthWebhookURL, eventWebhookURL, sessionID, *title, state)
	go forwardStream(os.Stderr, stderr, client, botToken, chatID, oauthWebhookURL, eventWebhookURL, sessionID, *title, state)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		_ = cmd.Process.Kill()
	}()

	if err := cmd.Wait(); err != nil {
		fail(err)
	}
}

func router(callbacks *callbackStore, chatID string, botToken string, client *http.Client, oauthWebhookURL string, eventWebhookURL string, title string, state *sessionState) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
		if sessionID == "" {
			http.Error(w, "missing session", http.StatusBadRequest)
			return
		}
		value, updatedAt, ok := waitForCallback(callbacks, sessionID, r.URL.Query().Get("wait"))
		if !ok {
			http.Error(w, "callback not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"sessionId": sessionID, "value": value, "updatedAt": updatedAt.Format(time.RFC3339)})
	})
	mux.HandleFunc("/callbacks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"sessions": callbacks.list()})
	})
	mux.HandleFunc("/telegram/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		chatIDValue, text, callbackQueryID, err := decodeTelegramUpdate(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if chatID != "" && chatIDValue != "" && chatIDValue != chatID {
			http.Error(w, "chat not allowed", http.StatusForbidden)
			return
		}
		handled := handleTelegramMessage(text, callbacks, state)
		if !handled {
			http.Error(w, "callback command not found", http.StatusBadRequest)
			return
		}
		if callbackQueryID != "" {
			payload := relayPayload{SessionID: state.get(), Title: title, Message: fmt.Sprintf("Applied: %s", humanizeCallbackValue(text))}
			_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, payload)
		}
		go func() {
			time.Sleep(700 * time.Millisecond)
			if payload := buildPayloadFromState(state, title); payload != nil {
				_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, *payload)
			}
		}()
		if callbackQueryID != "" {
			_ = answerCallbackQuery(&http.Client{Timeout: 20 * time.Second}, botToken, callbackQueryID)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stored"))
	})
	return mux
}

func startCommand(command string) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exec.Cmd, error) {
	cmd := exec.Command("script", "-qfec", command, "/dev/null")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, err
	}
	return stdin, stdout, stderr, cmd, nil
}

func forwardStream(dst io.Writer, src io.Reader, client *http.Client, botToken string, chatID string, oauthWebhookURL string, eventWebhookURL string, sessionID string, title string, state *sessionState) {
	buf := make([]byte, 1024)
	var seenURL string
	var lastPrompt string
	var lastMenuSignature string
	window := ""
	for {
		n, err := src.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			_, _ = dst.Write(chunk)
			clean := ansiRegex.ReplaceAllString(string(chunk), "")
			window += clean
			if state != nil {
				state.setWindow(window)
			}
			if len(window) > 8192 {
				window = window[len(window)-8192:]
			}
			if seenURL == "" {
				if url := firstURL(window); url != "" {
					seenURL = url
					payload := relayPayload{SessionID: sessionID, Title: title, Message: "Tap the login button below and complete sign-in in browser. Then use Telegram buttons for prompt inputs.", URL: url}
					_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, payload)
				}
			}
			menuOptions := extractVisibleMenuOptions(window)
			menuSignature := signatureForMenuOptions(menuOptions)
			if menuSignature != "" && menuSignature != lastMenuSignature {
				payload := relayPayload{SessionID: sessionID, Title: title, Message: "Select from the live menu below.", InlineKeyboard: buildMenuOptionKeyboard(sessionID, menuOptions)}
				_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, payload)
				lastMenuSignature = menuSignature
			}
			prompt := detectPrompt(window)
			if prompt != "" && prompt != lastPrompt {
				message := fmt.Sprintf("Auth prompt detected:\n%s\n\nUse the buttons when available, or send the requested value directly.", prompt)
				payload := relayPayload{SessionID: sessionID, Title: title, Message: message, URL: seenURL, Choices: extractPromptChoices(prompt)}
				_ = deliverPayload(client, botToken, chatID, oauthWebhookURL, eventWebhookURL, payload)
				lastPrompt = prompt
			}
		}
		if err != nil {
			return
		}
	}
}

func firstURL(text string) string {
	return strings.TrimSpace(urlRegex.FindString(text))
}

func detectPrompt(text string) string {
	normalizedWindow := normalizePromptText(text)
	lines := strings.Split(normalizedWindow, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		for _, prompt := range prompts {
			if strings.Contains(line, prompt) {
				return prompt
			}
		}
		if len(choiceParenRegex.FindAllStringSubmatch(line, -1)) >= 2 {
			return line
		}
		lower := strings.ToLower(line)
		if strings.HasSuffix(line, ":") || strings.HasSuffix(line, "?") {
			for _, hint := range promptHints {
				if strings.Contains(lower, hint) {
					return line
				}
			}
		}
	}
	return ""
}

func normalizePromptText(text string) string {
	raw := strings.TrimSpace(ansiFragmentRegex.ReplaceAllString(text, " "))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if r == '\r' || r == 0 {
			b.WriteByte(' ')
			continue
		}
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	cleaned := b.String()
	lines := strings.Split(cleaned, "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if joined == "" {
		return ""
	}
	if strings.Count(joined, "\n") > len(joined)/4 {
		compact := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			if !unicode.IsPrint(r) {
				return -1
			}
			return unicode.ToLower(r)
		}, joined)
		if strings.Contains(compact, "loginmethod") || (strings.Contains(compact, "addaccount") && strings.Contains(compact, "checkquotas")) {
			return strings.ReplaceAll(joined, "\n", " ")
		}
	}
	return joined
}

func newSessionID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func deliverPayload(client *http.Client, botToken string, chatID string, oauthWebhookURL string, eventWebhookURL string, payload relayPayload) error {
	if len(payload.InlineKeyboard) == 0 {
		payload.InlineKeyboard = buildInlineKeyboard(payload)
	}
	payload.Text = buildMessage(payload)
	payload.ChatID = strings.TrimSpace(chatID)
	payload.DisablePreview = true
	if len(payload.InlineKeyboard) > 0 {
		payload.ReplyMarkup = map[string]any{"inline_keyboard": payload.InlineKeyboard}
		payload.ReplyMarkupAlt = payload.ReplyMarkup
		payload.InlineKeyboardAlt = payload.InlineKeyboard
		if raw, err := json.Marshal(payload.ReplyMarkup); err == nil {
			payload.ReplyMarkupJSON = string(raw)
		}
	}
	payload.Telegram = map[string]any{
		"chat_id":                  payload.ChatID,
		"text":                     payload.Text,
		"disable_web_page_preview": true,
	}
	if payload.ReplyMarkup != nil {
		payload.Telegram["reply_markup"] = payload.ReplyMarkup
	}
	payload.Data = map[string]any{
		"chat_id":                  payload.ChatID,
		"text":                     payload.Text,
		"disable_web_page_preview": true,
	}
	if payload.ReplyMarkup != nil {
		payload.Data["reply_markup"] = payload.ReplyMarkup
	}
	webhookURL := eventWebhookURL
	if strings.TrimSpace(payload.URL) != "" {
		webhookURL = oauthWebhookURL
	} else if strings.TrimSpace(webhookURL) == "" && strings.TrimSpace(oauthWebhookURL) != "" {
		webhookURL = oauthWebhookURL
	}
	if webhookURL != "" {
		webhookErr := sendWebhookPayload(client, webhookURL, payload)
		if botToken != "" && chatID != "" {
			telegramErr := sendTelegramPayload(client, botToken, chatID, payload)
			if webhookErr != nil && telegramErr != nil {
				return fmt.Errorf("webhook and telegram delivery failed: webhook=%v telegram=%v", webhookErr, telegramErr)
			}
			return nil
		}
		return webhookErr
	}
	if botToken == "" || chatID == "" {
		return fmt.Errorf("missing Telegram delivery configuration")
	}
	return sendTelegramPayload(client, botToken, chatID, payload)
}

func sendWebhookPayload(client *http.Client, webhookURL string, payload relayPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("n8n webhook error: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

func sendTelegramPayload(client *http.Client, botToken string, chatID string, payload relayPayload) error {
	text := strings.TrimSpace(payload.Text)
	if text == "" {
		text = buildMessage(payload)
	}
	msg := map[string]any{"chat_id": chatID, "text": text, "disable_web_page_preview": true}
	keyboard := payload.InlineKeyboard
	if len(keyboard) == 0 {
		keyboard = buildInlineKeyboard(payload)
	}
	if len(keyboard) > 0 {
		msg["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	resp, err := client.Post(fmt.Sprintf("%s/bot%s/sendMessage", apiBase(), botToken), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result telegramResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("sendMessage API error: %s", result.Description)
	}
	return nil
}

func buildInlineKeyboard(payload relayPayload) [][]map[string]string {
	rows := [][]map[string]string{}
	if strings.TrimSpace(payload.URL) != "" {
		rows = append(rows, []map[string]string{{"text": payload.Title, "url": strings.TrimSpace(payload.URL)}})
	}
	if strings.TrimSpace(payload.SessionID) != "" {
		rows = append(rows, []map[string]string{{
			"text":                             "Fill callback command",
			"switch_inline_query_current_chat": fmt.Sprintf("/callback %s ", strings.TrimSpace(payload.SessionID)),
		}})
		rows = append(rows, []map[string]string{
			{"text": "Up", "callback_data": buildButtonCallback(payload.SessionID, "nav:up")},
			{"text": "Down", "callback_data": buildButtonCallback(payload.SessionID, "nav:down")},
			{"text": "Select", "callback_data": buildButtonCallback(payload.SessionID, "__ENTER__")},
		})
		if len(payload.Choices) > 0 {
			rows = append(rows, buildChoiceButtonRow(payload.SessionID, payload.Choices))
		} else {
			rows = append(rows, []map[string]string{
				{"text": "1", "callback_data": buildButtonCallback(payload.SessionID, "1")},
				{"text": "2", "callback_data": buildButtonCallback(payload.SessionID, "2")},
				{"text": "3", "callback_data": buildButtonCallback(payload.SessionID, "3")},
			})
		}
		rows = append(rows, []map[string]string{{"text": "Continue", "callback_data": buildButtonCallback(payload.SessionID, "__ENTER__")}})
		if needsBinaryChoice(payload.Message) {
			rows = append(rows, []map[string]string{
				{"text": "Yes", "callback_data": buildButtonCallback(payload.SessionID, "yes")},
				{"text": "No", "callback_data": buildButtonCallback(payload.SessionID, "no")},
			})
		}
	}
	return rows
}

func isGoogleAuthCommand(command string) bool {
	trimmed := strings.TrimSpace(normalizeAuthCommand(command))
	return strings.Contains(trimmed, "opencode auth login -p google")
}

func buildMenuOptionKeyboard(sessionID string, options []menuOption) [][]map[string]string {
	rows := make([][]map[string]string, 0, (len(options)+1)/2)
	for i := 0; i < len(options); i += 2 {
		row := []map[string]string{}
		for j := i; j < len(options) && j < i+2; j++ {
			option := options[j]
			if option.Label == "" || option.Input == "" {
				continue
			}
			row = append(row, map[string]string{"text": option.Label, "callback_data": buildButtonCallback(sessionID, option.Input)})
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func extractVisibleMenuOptions(text string) []menuOption {
	normalized := normalizePromptText(text)
	if normalized == "" {
		return nil
	}
	lines := strings.Split(normalized, "\n")
	blocks := [][]string{}
	current := []string{}
	flush := func() {
		if len(current) >= 2 {
			blocks = append(blocks, append([]string(nil), current...))
		}
		current = nil
	}
	for _, raw := range lines {
		line := sanitizeMenuLine(raw)
		if isMenuCandidate(line) {
			current = append(current, line)
			continue
		}
		flush()
	}
	flush()
	if len(blocks) == 0 {
		return nil
	}
	block := blocks[len(blocks)-1]
	options := make([]menuOption, 0, len(block))
	for idx, line := range block {
		options = append(options, menuOption{Label: line, Input: fmt.Sprintf("numpad:%d", idx+1)})
	}
	return options
}

func sanitizeMenuLine(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, ">*.-+ ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	return strings.TrimSpace(trimmed)
}

func isMenuCandidate(line string) bool {
	if line == "" || len(line) > 80 {
		return false
	}
	lower := strings.ToLower(line)
	for _, banned := range []string{"http://", "https://", "session:", "bridge connected", "auth prompt detected", "reply directly", "explicit target command", "fill callback command", "select from the live menu below", "login method"} {
		if strings.Contains(lower, banned) {
			return false
		}
	}
	if strings.Contains(line, ":") || strings.Contains(line, "[") || strings.Contains(line, "]") {
		return false
	}
	letters := 0
	for _, r := range line {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	return letters >= 3
}

func signatureForMenuOptions(options []menuOption) string {
	parts := make([]string, 0, len(options))
	for _, option := range options {
		if option.Label != "" {
			parts = append(parts, option.Label)
		}
	}
	return strings.Join(parts, "|")
}

func buildPayloadFromState(state *sessionState, title string) *relayPayload {
	if state == nil {
		return nil
	}
	sessionID := state.get()
	window := state.getWindow()
	if sessionID == "" || window == "" {
		return nil
	}
	if options := extractVisibleMenuOptions(window); len(options) > 0 {
		payload := relayPayload{SessionID: sessionID, Title: title, Message: "Select from the live menu below.", InlineKeyboard: buildMenuOptionKeyboard(sessionID, options)}
		return &payload
	}
	if prompt := detectPrompt(window); prompt != "" {
		payload := relayPayload{SessionID: sessionID, Title: title, Message: fmt.Sprintf("Auth prompt detected:\n%s\n\nUse the buttons when available, or send the requested value directly.", prompt), Choices: extractPromptChoices(prompt)}
		return &payload
	}
	return nil
}

func humanizeCallbackValue(raw string) string {
	_, value, ok := parseButtonCallback(strings.TrimSpace(raw))
	if !ok {
		return "input"
	}
	switch value {
	case "__ENTER__":
		return "select"
	case "nav:up":
		return "up"
	case "nav:down":
		return "down"
	default:
		return value
	}
}

func buildChoiceButtonRow(sessionID string, choices []string) []map[string]string {
	row := make([]map[string]string, 0, len(choices))
	for _, choice := range choices {
		value := strings.TrimSpace(choice)
		if value == "" {
			continue
		}
		row = append(row, map[string]string{"text": value, "callback_data": buildButtonCallback(sessionID, value)})
	}
	return row
}

func extractPromptChoices(prompt string) []string {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return nil
	}
	seen := map[string]bool{}
	choices := make([]string, 0, 8)
	appendChoice := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			return
		}
		seen[key] = true
		choices = append(choices, trimmed)
	}
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		raw := strings.TrimSpace(text[start+1 : end])
		if raw != "" {
			for _, part := range strings.Split(raw, "/") {
				appendChoice(part)
			}
		}
	}
	for _, match := range choiceParenRegex.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			appendChoice(match[1])
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return choices
}

func needsBinaryChoice(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "yes/no") || strings.Contains(lower, "y/n") || strings.Contains(lower, "confirm")
}

func buildButtonCallback(sessionID string, value string) string {
	return "cb:" + strings.TrimSpace(sessionID) + ":" + strings.TrimSpace(value)
}

func parseButtonCallback(raw string) (string, string, bool) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "cb:") {
		return "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "cb:"), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	sessionID := strings.TrimSpace(parts[0])
	payload := strings.TrimSpace(parts[1])
	if sessionID == "" || payload == "" {
		return "", "", false
	}
	return sessionID, payload, true
}

func buildMessage(payload relayPayload) string {
	parts := []string{strings.TrimSpace(payload.Title)}
	if strings.TrimSpace(payload.SessionID) != "" {
		parts = append(parts, fmt.Sprintf("session: %s", payload.SessionID))
		parts = append(parts, "use buttons first for prompt inputs")
		parts = append(parts, fmt.Sprintf("explicit target command: /callback %s <value>", payload.SessionID))
	}
	if strings.TrimSpace(payload.Message) != "" {
		parts = append(parts, strings.TrimSpace(payload.Message))
	}
	return strings.Join(parts, "\n\n")
}

func normalizeAuthCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "opencodode auth login") {
		return strings.Replace(trimmed, "opencodode", "opencode", 1)
	}
	return command
}

func waitForCallback(store *callbackStore, sessionID string, waitValue string) (string, time.Time, bool) {
	value, updatedAt, _, ok := store.popAfter(sessionID, 0)
	if ok {
		return value, updatedAt, true
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(waitValue) + "s")
	if err != nil || parsed <= 0 {
		return "", time.Time{}, false
	}
	deadline := time.Now().Add(parsed)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		value, updatedAt, _, ok = store.popAfter(sessionID, 0)
		if ok {
			return value, updatedAt, true
		}
	}
	return "", time.Time{}, false
}

func waitForCallbackAfter(store *callbackStore, sessionID string, waitValue string, afterSeq uint64) (string, uint64, bool) {
	parsed, err := time.ParseDuration(strings.TrimSpace(waitValue) + "s")
	if err != nil || parsed <= 0 {
		return "", 0, false
	}
	deadline := time.Now().Add(parsed)
	value, _, nextSeq, ok := store.popAfter(sessionID, afterSeq)
	if ok {
		return value, nextSeq, true
	}
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		value, _, nextSeq, ok = store.popAfter(sessionID, afterSeq)
		if ok {
			return value, nextSeq, true
		}
	}
	return "", 0, false
}

func drainCallbacks(store *callbackStore, stdin *lockedWriter, sessionID string) {
	for {
		value, _, ok := waitForCallback(store, sessionID, "86400")
		if !ok {
			continue
		}
		payload := value
		if payload == "" {
			payload = "\r"
		} else if strings.Contains(payload, "\x1b[") {
			if !strings.HasSuffix(payload, "\n") {
				payload += "\n"
			}
		} else {
			payload += "\n"
		}
		_, _ = stdin.Write([]byte(payload))
	}
}

func handleTelegramMessage(text string, callbacks *callbackStore, state *sessionState) bool {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return false
	}
	activeSessionID := ""
	if state != nil {
		activeSessionID = state.get()
	}
	if sessionID, value, ok := parseButtonCallback(raw); ok {
		if value == "__ENTER__" {
			value = "\r"
		}
		if value == "nav:up" {
			value = "\x1b[A"
		}
		if value == "nav:down" {
			value = "\x1b[B"
		}
		if strings.HasPrefix(value, "numpad:") {
			if n, err := strconv.Atoi(strings.TrimPrefix(value, "numpad:")); err == nil && n > 0 {
				value = ""
				for i := 1; i < n; i++ {
					value += "\x1b[B"
				}
				value += "\r"
			}
		}
		callbacks.put(sessionID, value)
		return true
	}
	if strings.HasPrefix(raw, "/input ") {
		sessionID := activeSessionID
		if sessionID == "" {
			return false
		}
		value := strings.TrimSpace(strings.TrimPrefix(raw, "/input "))
		if value == "" {
			return false
		}
		callbacks.put(sessionID, normalizeTelegramInput(value))
		return true
	}
	parts := strings.Fields(raw)
	if len(parts) < 3 {
		if !strings.HasPrefix(raw, "/") {
			sessionID := activeSessionID
			if sessionID == "" {
				return false
			}
			callbacks.put(sessionID, normalizeTelegramInput(raw))
			return true
		}
		return false
	}
	if parts[0] != "/callback" && parts[0] != "/oauth" {
		if !strings.HasPrefix(raw, "/") {
			sessionID := activeSessionID
			if sessionID == "" {
				return false
			}
			callbacks.put(sessionID, normalizeTelegramInput(raw))
			return true
		}
		return false
	}
	callbacks.put(strings.TrimSpace(parts[1]), normalizeTelegramInput(strings.TrimSpace(strings.Join(parts[2:], " "))))
	return true
}

func normalizeTelegramInput(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		if parsed, err := url.Parse(raw); err == nil {
			code := strings.TrimSpace(parsed.Query().Get("code"))
			if code != "" {
				return code
			}
		}
	}
	if strings.Contains(raw, "code=") && strings.Contains(raw, "&") {
		if query, err := url.ParseQuery(raw); err == nil {
			code := strings.TrimSpace(query.Get("code"))
			if code != "" {
				return code
			}
		}
	}
	return raw
}

func decodeTelegramUpdate(body io.Reader) (string, string, string, error) {
	data, err := io.ReadAll(io.LimitReader(body, 16384))
	if err != nil {
		return "", "", "", err
	}
	var payload inboundTelegramUpdate
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", "", err
	}
	text := strings.TrimSpace(payload.Text)
	chatID := strings.TrimSpace(payload.ChatID)
	callbackQueryID := ""
	if payload.CallbackQuery != nil {
		callbackQueryID = strings.TrimSpace(payload.CallbackQuery.ID)
		if payload.CallbackQuery.Message != nil && payload.CallbackQuery.Message.Chat != nil {
			chatID = fmt.Sprintf("%d", payload.CallbackQuery.Message.Chat.ID)
		}
		if payload.CallbackQuery.Data != "" {
			text = strings.TrimSpace(payload.CallbackQuery.Data)
		}
	}
	if payload.Message != nil {
		if payload.Message.Chat != nil {
			chatID = fmt.Sprintf("%d", payload.Message.Chat.ID)
		}
		if payload.Message.Text != "" {
			text = strings.TrimSpace(payload.Message.Text)
		}
	}
	if text == "" {
		return "", "", "", fmt.Errorf("missing message text")
	}
	return chatID, text, callbackQueryID, nil
}

func pollTelegramUpdates(client *http.Client, botToken string, allowedChatID string, callbacks *callbackStore, state *sessionState) {
	endpoint := fmt.Sprintf("%s/bot%s/getUpdates", apiBase(), botToken)
	offset := 0
	for {
		updates, next, err := getTelegramUpdates(client, endpoint, offset)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		offset = next
		for _, update := range updates {
			chatIDValue, text, callbackQueryID, ok := decodeTelegramPollingUpdate(update)
			if !ok {
				continue
			}
			if chatIDValue != allowedChatID {
				continue
			}
			if handled := handleTelegramMessage(text, callbacks, state); handled && callbackQueryID != "" {
				_ = answerCallbackQuery(client, botToken, callbackQueryID)
			}
		}
	}
}

func decodeTelegramPollingUpdate(update telegramUpdate) (string, string, string, bool) {
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message != nil {
			chatID := fmt.Sprintf("%d", update.CallbackQuery.Message.Chat.ID)
			text := strings.TrimSpace(update.CallbackQuery.Data)
			if chatID != "" && text != "" {
				return chatID, text, strings.TrimSpace(update.CallbackQuery.ID), true
			}
		}
	}
	if update.Message != nil {
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)
		if chatID != "" && text != "" {
			return chatID, text, "", true
		}
	}
	return "", "", "", false
}

func answerCallbackQuery(client *http.Client, botToken string, callbackQueryID string) error {
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(callbackQueryID) == "" {
		return nil
	}
	body, err := json.Marshal(map[string]any{
		"callback_query_id": strings.TrimSpace(callbackQueryID),
		"text":              "applied",
		"show_alert":        false,
	})
	if err != nil {
		return err
	}
	resp, err := client.Post(fmt.Sprintf("%s/bot%s/answerCallbackQuery", apiBase(), botToken), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("answerCallbackQuery API error: %s", strings.TrimSpace(string(data)))
	}
	return nil
}

func getTelegramUpdates(client *http.Client, endpoint string, offset int) ([]telegramUpdate, int, error) {
	body, _ := json.Marshal(map[string]any{"offset": offset, "timeout": 30})
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, offset, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var envelope telegramResponse
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, offset, err
	}
	if !envelope.OK {
		return nil, offset, fmt.Errorf("getUpdates API error: %s", envelope.Description)
	}
	var updates []telegramUpdate
	if len(envelope.ResultRaw) > 0 {
		if err := json.Unmarshal(envelope.ResultRaw, &updates); err != nil {
			return nil, offset, err
		}
	}
	next := offset
	for _, update := range updates {
		if update.UpdateID >= next {
			next = update.UpdateID + 1
		}
	}
	return updates, next, nil
}

func discoverChatID(botToken string, user string) (string, error) {
	client := &http.Client{Timeout: 35 * time.Second}
	endpoint := fmt.Sprintf("%s/bot%s/getUpdates?timeout=1", apiBase(), botToken)
	for range 10 {
		resp, err := client.Get(endpoint)
		if err != nil {
			return "", err
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var env telegramResponse
		if err := json.Unmarshal(data, &env); err != nil {
			return "", err
		}
		if !env.OK {
			return "", fmt.Errorf("getUpdates API error: %s", env.Description)
		}
		var updates []telegramUpdate
		if len(env.ResultRaw) > 0 {
			if err := json.Unmarshal(env.ResultRaw, &updates); err == nil {
				for i := len(updates) - 1; i >= 0; i-- {
					if updates[i].Message != nil {
						return fmt.Sprintf("%d", updates[i].Message.Chat.ID), nil
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	if user != "" {
		return "", fmt.Errorf("no Telegram chat id discovered - open https://t.me/%s and send /start first", user)
	}
	return "", fmt.Errorf("no Telegram chat id discovered")
}

func resolveValue(value string, fallbackRef string) (string, error) {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		candidate = strings.TrimSpace(fallbackRef)
	}
	if candidate == "" {
		return "", nil
	}
	if !strings.HasPrefix(candidate, "op://") {
		return candidate, nil
	}
	cmd := exec.Command("op", "read", candidate)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve %s: %s", candidate, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return fallback
}

func apiBase() string {
	return strings.TrimRight(envOrDefault("TELEGRAM_API_BASE", "https://api.telegram.org"), "/")
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
