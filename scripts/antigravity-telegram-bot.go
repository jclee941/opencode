// antigravity-telegram-bot.go provides Telegram-based interactive authentication
// management for Antigravity OAuth accounts.
//
// Features:
//   - Monitors antigravity-accounts.json for verification/cooldown state changes
//   - Telegram bot commands: /status, /accounts, /verify, /enable, /disable, /clearverify
//   - Optional HTTP endpoint for OAuth URL relay from opencode login flow
//
// Environment variables:
//
//	TELEGRAM_BOT_TOKEN   — Telegram Bot API token (required)
//	TELEGRAM_CHAT_ID     — Telegram chat ID to send notifications (required)
//	ACCOUNTS_FILE        — Path to antigravity-accounts.json (default: ~/.config/opencode/antigravity-accounts.json)
//	OAUTH_RELAY_PORT     — Port for OAuth URL relay HTTP endpoint (default: disabled)
//	POLL_INTERVAL        — File poll interval in seconds (default: 5)
//
// Usage:
//
//	TELEGRAM_BOT_TOKEN=xxx TELEGRAM_CHAT_ID=yyy go run scripts/antigravity-telegram-bot.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------
// Accounts JSON schema (V4) — mirrors opencode-antigravity-auth types
// ---------------------------------------------------------------------------

type AccountStorage struct {
	Version             int               `json:"version"`
	Accounts            []AccountMetadata `json:"accounts"`
	ActiveIndex         int               `json:"activeIndex"`
	ActiveIndexByFamily map[string]int    `json:"activeIndexByFamily,omitempty"`
}

type AccountMetadata struct {
	Email                      string                 `json:"email,omitempty"`
	RefreshToken               string                 `json:"refreshToken"`
	ProjectID                  string                 `json:"projectId,omitempty"`
	ManagedProjectID           string                 `json:"managedProjectId,omitempty"`
	AddedAt                    int64                  `json:"addedAt"`
	LastUsed                   int64                  `json:"lastUsed"`
	Enabled                    *bool                  `json:"enabled,omitempty"`
	RateLimitResetTimes        map[string]interface{} `json:"rateLimitResetTimes,omitempty"`
	CoolingDownUntil           *int64                 `json:"coolingDownUntil,omitempty"`
	CooldownReason             string                 `json:"cooldownReason,omitempty"`
	Fingerprint                map[string]interface{} `json:"fingerprint,omitempty"`
	VerificationRequired       *bool                  `json:"verificationRequired,omitempty"`
	VerificationRequiredAt     *int64                 `json:"verificationRequiredAt,omitempty"`
	VerificationRequiredReason string                 `json:"verificationRequiredReason,omitempty"`
	VerificationURL            string                 `json:"verificationUrl,omitempty"`
	CachedQuota                map[string]interface{} `json:"cachedQuota,omitempty"`
	CachedQuotaUpdatedAt       *int64                 `json:"cachedQuotaUpdatedAt,omitempty"`
}

func (a *AccountMetadata) isEnabled() bool {
	return a.Enabled == nil || *a.Enabled
}

func (a *AccountMetadata) needsVerification() bool {
	return a.VerificationRequired != nil && *a.VerificationRequired
}

func (a *AccountMetadata) isCoolingDown() bool {
	if a.CoolingDownUntil == nil {
		return false
	}
	return *a.CoolingDownUntil > time.Now().UnixMilli()
}

func (a *AccountMetadata) displayName(idx int) string {
	if a.Email != "" {
		return fmt.Sprintf("[%d] %s", idx, a.Email)
	}
	return fmt.Sprintf("[%d] (no email)", idx)
}

// ---------------------------------------------------------------------------
// Telegram Bot API (minimal, stdlib-only)
// ---------------------------------------------------------------------------

type TelegramBot struct {
	token  string
	chatID string
	client *http.Client
	offset int
}

type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type TelegramMessage struct {
	MessageID int           `json:"message_id"`
	Chat      TelegramChat  `json:"chat"`
	Text      string        `json:"text"`
	From      *TelegramUser `json:"from,omitempty"`
}

type TelegramChat struct {
	ID int64 `json:"id"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type telegramResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Desc   string          `json:"description,omitempty"`
}

func newTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: 35 * time.Second},
	}
}

func (b *TelegramBot) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)
}

func (b *TelegramBot) sendMessage(chatID string, text string, parseMode string) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": parseMode,
	}
	body, _ := json.Marshal(payload)
	resp, err := b.client.Post(b.apiURL("sendMessage"), "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sendMessage: %w", err)
	}
	defer resp.Body.Close()
	var r telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("sendMessage decode: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("sendMessage API error: %s", r.Desc)
	}
	return nil
}

func (b *TelegramBot) notify(text string) {
	if err := b.sendMessage(b.chatID, text, "HTML"); err != nil {
		log.Printf("telegram notify error: %v", err)
	}
}

func (b *TelegramBot) getUpdates() ([]TelegramUpdate, error) {
	payload := map[string]interface{}{
		"offset":  b.offset,
		"timeout": 30,
	}
	body, _ := json.Marshal(payload)
	resp, err := b.client.Post(b.apiURL("getUpdates"), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("getUpdates: %w", err)
	}
	defer resp.Body.Close()
	var r telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("getUpdates decode: %w", err)
	}
	if !r.OK {
		return nil, fmt.Errorf("getUpdates API error: %s", r.Desc)
	}
	var updates []TelegramUpdate
	if err := json.Unmarshal(r.Result, &updates); err != nil {
		return nil, fmt.Errorf("getUpdates unmarshal: %w", err)
	}
	return updates, nil
}

// ---------------------------------------------------------------------------
// Accounts file operations
// ---------------------------------------------------------------------------

type AccountsManager struct {
	path string
	mu   sync.Mutex
}

func newAccountsManager(path string) *AccountsManager {
	return &AccountsManager{path: path}
}

func (m *AccountsManager) load() (*AccountStorage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadUnlocked()
}

func (m *AccountsManager) loadUnlocked() (*AccountStorage, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, fmt.Errorf("read accounts: %w", err)
	}
	var storage AccountStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("parse accounts: %w", err)
	}
	return &storage, nil
}

func (m *AccountsManager) save(storage *AccountStorage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal accounts: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(m.path, data, 0644); err != nil {
		return fmt.Errorf("write accounts: %w", err)
	}
	return nil
}

func (m *AccountsManager) setEnabled(index int, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	storage, err := m.loadUnlocked()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(storage.Accounts) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(storage.Accounts)-1)
	}
	storage.Accounts[index].Enabled = &enabled
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.path, data, 0644)
}

func (m *AccountsManager) clearVerification(index int, enable bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	storage, err := m.loadUnlocked()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(storage.Accounts) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(storage.Accounts)-1)
	}
	acct := &storage.Accounts[index]
	f := false
	acct.VerificationRequired = &f
	acct.VerificationURL = ""
	acct.VerificationRequiredReason = ""
	acct.VerificationRequiredAt = nil
	acct.CoolingDownUntil = nil
	acct.CooldownReason = ""
	if enable {
		t := true
		acct.Enabled = &t
	}
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.path, data, 0644)
}

// ---------------------------------------------------------------------------
// File watcher (polling-based, cross-platform)
// ---------------------------------------------------------------------------

type FileWatcher struct {
	path     string
	interval time.Duration
	lastMod  time.Time
	lastSize int64
	onChange func()
	stop     chan struct{}
}

func newFileWatcher(path string, interval time.Duration, onChange func()) *FileWatcher {
	return &FileWatcher{
		path:     path,
		interval: interval,
		onChange: onChange,
		stop:     make(chan struct{}),
	}
}

func (w *FileWatcher) start() {
	info, err := os.Stat(w.path)
	if err == nil {
		w.lastMod = info.ModTime()
		w.lastSize = info.Size()
	}
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(w.path)
				if err != nil {
					continue
				}
				if info.ModTime() != w.lastMod || info.Size() != w.lastSize {
					w.lastMod = info.ModTime()
					w.lastSize = info.Size()
					w.onChange()
				}
			case <-w.stop:
				return
			}
		}
	}()
}

func (w *FileWatcher) close() {
	close(w.stop)
}

// ---------------------------------------------------------------------------
// Bot command handlers
// ---------------------------------------------------------------------------

type BotHandler struct {
	bot    *TelegramBot
	accts  *AccountsManager
	chatID string
}

func newBotHandler(bot *TelegramBot, accts *AccountsManager, chatID string) *BotHandler {
	return &BotHandler{bot: bot, accts: accts, chatID: chatID}
}

func (h *BotHandler) handleCommand(chatID string, text string) {
	// Only respond to authorized chat
	if chatID != h.chatID {
		log.Printf("ignoring message from unauthorized chat: %s", chatID)
		return
	}

	text = strings.TrimSpace(text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	// Strip @botname suffix from command
	if at := strings.Index(cmd, "@"); at > 0 {
		cmd = cmd[:at]
	}
	args := parts[1:]

	switch cmd {
	case "/start", "/help":
		h.cmdHelp()
	case "/status":
		h.cmdStatus()
	case "/accounts":
		h.cmdAccounts()
	case "/verify":
		h.cmdVerify(args)
	case "/enable":
		h.cmdEnable(args)
	case "/disable":
		h.cmdDisable(args)
	case "/clearverify":
		h.cmdClearVerify(args)
	case "/quota":
		h.cmdQuota()
	default:
		// Ignore non-commands
	}
}

func (h *BotHandler) cmdHelp() {
	msg := `<b>Antigravity Auth Bot</b>

<b>Commands:</b>
/status — Overview of account health
/accounts — List all accounts with status
/quota — Show quota info for enabled accounts
/verify — Show accounts needing verification
/clearverify &lt;index&gt; — Clear verification flag and re-enable account
/enable &lt;index&gt; — Enable an account
/disable &lt;index&gt; — Disable an account
/help — Show this help`
	h.bot.notify(msg)
}

func (h *BotHandler) cmdStatus() {
	storage, err := h.accts.load()
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}

	total := len(storage.Accounts)
	enabled := 0
	needsVerify := 0
	coolingDown := 0
	for i := range storage.Accounts {
		a := &storage.Accounts[i]
		if a.isEnabled() {
			enabled++
		}
		if a.needsVerification() {
			needsVerify++
		}
		if a.isCoolingDown() {
			coolingDown++
		}
	}

	var sb strings.Builder
	sb.WriteString("<b>📊 Account Status</b>\n\n")
	sb.WriteString(fmt.Sprintf("Total: %d\n", total))
	sb.WriteString(fmt.Sprintf("Enabled: %d\n", enabled))
	sb.WriteString(fmt.Sprintf("Active: [%d]", storage.ActiveIndex))
	if storage.ActiveIndex >= 0 && storage.ActiveIndex < total {
		a := &storage.Accounts[storage.ActiveIndex]
		if a.Email != "" {
			sb.WriteString(fmt.Sprintf(" %s", a.Email))
		}
	}
	sb.WriteString("\n")

	if needsVerify > 0 {
		sb.WriteString(fmt.Sprintf("\n⚠️ <b>Verification needed: %d</b>\n", needsVerify))
	}
	if coolingDown > 0 {
		sb.WriteString(fmt.Sprintf("❄️ Cooling down: %d\n", coolingDown))
	}

	if len(storage.ActiveIndexByFamily) > 0 {
		sb.WriteString("\n<b>Active by family:</b>\n")
		families := make([]string, 0, len(storage.ActiveIndexByFamily))
		for f := range storage.ActiveIndexByFamily {
			families = append(families, f)
		}
		sort.Strings(families)
		for _, f := range families {
			idx := storage.ActiveIndexByFamily[f]
			sb.WriteString(fmt.Sprintf("  %s → [%d]", f, idx))
			if idx >= 0 && idx < total {
				if e := storage.Accounts[idx].Email; e != "" {
					sb.WriteString(fmt.Sprintf(" %s", e))
				}
			}
			sb.WriteString("\n")
		}
	}

	h.bot.notify(sb.String())
}

func (h *BotHandler) cmdAccounts() {
	storage, err := h.accts.load()
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}

	var sb strings.Builder
	sb.WriteString("<b>📋 All Accounts</b>\n\n")

	for i := range storage.Accounts {
		a := &storage.Accounts[i]
		status := "✅"
		if !a.isEnabled() {
			status = "⛔"
		}
		if a.needsVerification() {
			status = "⚠️"
		}
		if a.isCoolingDown() {
			status = "❄️"
		}
		if i == storage.ActiveIndex {
			status = "🟢"
		}

		email := a.Email
		if email == "" {
			email = "(no email)"
		}
		sb.WriteString(fmt.Sprintf("%s <code>[%d]</code> %s", status, i, email))
		if a.needsVerification() {
			sb.WriteString(" — needs verify")
		}
		if a.isCoolingDown() {
			remaining := time.Until(time.UnixMilli(*a.CoolingDownUntil)).Truncate(time.Second)
			sb.WriteString(fmt.Sprintf(" — cooldown %s", remaining))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n🟢 active  ✅ enabled  ⛔ disabled  ⚠️ verify  ❄️ cooldown")
	h.bot.notify(sb.String())
}

func (h *BotHandler) cmdVerify(args []string) {
	storage, err := h.accts.load()
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}

	var sb strings.Builder
	sb.WriteString("<b>🔐 Accounts Needing Verification</b>\n\n")

	found := false
	for i := range storage.Accounts {
		a := &storage.Accounts[i]
		if !a.needsVerification() {
			continue
		}
		found = true
		sb.WriteString(fmt.Sprintf("<b>[%d] %s</b>\n", i, a.Email))
		if a.VerificationRequiredReason != "" {
			sb.WriteString(fmt.Sprintf("  Reason: %s\n", a.VerificationRequiredReason))
		}
		if a.CooldownReason != "" {
			sb.WriteString(fmt.Sprintf("  Cooldown: %s\n", a.CooldownReason))
		}
		if a.VerificationURL != "" {
			sb.WriteString(fmt.Sprintf("  URL: %s\n", a.VerificationURL))
		}
		if a.VerificationRequiredAt != nil {
			t := time.UnixMilli(*a.VerificationRequiredAt)
			sb.WriteString(fmt.Sprintf("  Since: %s\n", t.Format("2006-01-02 15:04:05")))
		}
		sb.WriteString(fmt.Sprintf("\nClear with: /clearverify %d\n\n", i))
	}

	if !found {
		sb.WriteString("✅ No accounts need verification.")
	}

	h.bot.notify(sb.String())
}

func (h *BotHandler) cmdClearVerify(args []string) {
	if len(args) == 0 {
		h.bot.notify("Usage: /clearverify &lt;index&gt;\nClears verification flag and re-enables the account.")
		return
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Invalid index: %s", args[0]))
		return
	}

	if err := h.accts.clearVerification(idx, true); err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}

	storage, _ := h.accts.load()
	email := ""
	if storage != nil && idx >= 0 && idx < len(storage.Accounts) {
		email = storage.Accounts[idx].Email
	}
	h.bot.notify(fmt.Sprintf("✅ Cleared verification for [%d] %s\nAccount re-enabled.", idx, email))
}

func (h *BotHandler) cmdEnable(args []string) {
	if len(args) == 0 {
		h.bot.notify("Usage: /enable &lt;index&gt;")
		return
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Invalid index: %s", args[0]))
		return
	}

	if err := h.accts.setEnabled(idx, true); err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}
	h.bot.notify(fmt.Sprintf("✅ Account [%d] enabled.", idx))
}

func (h *BotHandler) cmdDisable(args []string) {
	if len(args) == 0 {
		h.bot.notify("Usage: /disable &lt;index&gt;")
		return
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Invalid index: %s", args[0]))
		return
	}

	if err := h.accts.setEnabled(idx, false); err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}
	h.bot.notify(fmt.Sprintf("⛔ Account [%d] disabled.", idx))
}

func (h *BotHandler) cmdQuota() {
	storage, err := h.accts.load()
	if err != nil {
		h.bot.notify(fmt.Sprintf("❌ Error: %v", err))
		return
	}

	var sb strings.Builder
	sb.WriteString("<b>📈 Quota Status</b>\n\n")

	found := false
	for i := range storage.Accounts {
		a := &storage.Accounts[i]
		if !a.isEnabled() || a.CachedQuota == nil {
			continue
		}
		found = true
		sb.WriteString(fmt.Sprintf("<b>[%d] %s</b>\n", i, a.Email))

		families := make([]string, 0, len(a.CachedQuota))
		for f := range a.CachedQuota {
			families = append(families, f)
		}
		sort.Strings(families)

		for _, fam := range families {
			q, ok := a.CachedQuota[fam].(map[string]interface{})
			if !ok {
				continue
			}
			remaining := 0.0
			if v, ok := q["remainingFraction"].(float64); ok {
				remaining = v
			}
			bar := quotaBar(remaining)
			sb.WriteString(fmt.Sprintf("  %s %s %.0f%%\n", fam, bar, remaining*100))
		}
		sb.WriteString("\n")
	}

	if !found {
		sb.WriteString("No quota data for enabled accounts.")
	}

	h.bot.notify(sb.String())
}

func quotaBar(fraction float64) string {
	total := 10
	filled := int(fraction * float64(total))
	if filled > total {
		filled = total
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", total-filled)
}

// ---------------------------------------------------------------------------
// Change detector — compares snapshots to detect verification state changes
// ---------------------------------------------------------------------------

type accountSnapshot struct {
	email                string
	enabled              bool
	verificationRequired bool
	coolingDown          bool
	cooldownReason       string
}

func takeSnapshot(storage *AccountStorage) []accountSnapshot {
	snap := make([]accountSnapshot, len(storage.Accounts))
	for i := range storage.Accounts {
		a := &storage.Accounts[i]
		snap[i] = accountSnapshot{
			email:                a.Email,
			enabled:              a.isEnabled(),
			verificationRequired: a.needsVerification(),
			coolingDown:          a.isCoolingDown(),
			cooldownReason:       a.CooldownReason,
		}
	}
	return snap
}

func detectChanges(prev, curr []accountSnapshot, storage *AccountStorage) []string {
	var msgs []string

	maxLen := len(curr)
	if len(prev) > maxLen {
		maxLen = len(prev)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(prev) {
			// New account added
			msgs = append(msgs, fmt.Sprintf("🆕 New account [%d] %s added", i, curr[i].email))
			continue
		}
		if i >= len(curr) {
			// Account removed
			msgs = append(msgs, fmt.Sprintf("🗑️ Account [%d] %s removed", i, prev[i].email))
			continue
		}

		p, c := prev[i], curr[i]

		if !p.verificationRequired && c.verificationRequired {
			url := ""
			if i < len(storage.Accounts) {
				url = storage.Accounts[i].VerificationURL
			}
			msg := fmt.Sprintf("⚠️ <b>Verification required</b> for [%d] %s", i, c.email)
			if url != "" {
				msg += fmt.Sprintf("\n🔗 %s", url)
			}
			msg += fmt.Sprintf("\n\nClear with: /clearverify %d", i)
			msgs = append(msgs, msg)
		}
		if p.verificationRequired && !c.verificationRequired {
			msgs = append(msgs, fmt.Sprintf("✅ Verification cleared for [%d] %s", i, c.email))
		}

		if p.enabled && !c.enabled {
			msgs = append(msgs, fmt.Sprintf("⛔ Account [%d] %s disabled", i, c.email))
		}
		if !p.enabled && c.enabled {
			msgs = append(msgs, fmt.Sprintf("✅ Account [%d] %s enabled", i, c.email))
		}

		if !p.coolingDown && c.coolingDown {
			msgs = append(msgs, fmt.Sprintf("❄️ Account [%d] %s entered cooldown (%s)", i, c.email, c.cooldownReason))
		}
		if p.coolingDown && !c.coolingDown {
			msgs = append(msgs, fmt.Sprintf("🔥 Account [%d] %s cooldown expired", i, c.email))
		}
	}

	return msgs
}

// ---------------------------------------------------------------------------
// OAuth URL relay HTTP endpoint
// ---------------------------------------------------------------------------

func startOAuthRelay(port string, bot *TelegramBot, chatID string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth-url", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		var payload struct {
			URL          string `json:"url"`
			Instructions string `json:"instructions"`
			Method       string `json:"method"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			// Try plain text URL
			payload.URL = strings.TrimSpace(string(body))
		}

		if payload.URL == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		msg := fmt.Sprintf("🔑 <b>OAuth Login Required</b>\n\n%s", payload.URL)
		if payload.Instructions != "" {
			msg += fmt.Sprintf("\n\n📋 %s", payload.Instructions)
		}
		bot.notify(msg)

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Printf("OAuth relay listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("OAuth relay server error: %v", err)
	}
}

// discoverChatID polls getUpdates until someone sends a message to the bot,
// then returns that chat ID. Used when TELEGRAM_CHAT_ID is not set.
func discoverChatID(token string) string {
	fmt.Println("TELEGRAM_CHAT_ID not set. Waiting for a message to the bot...")
	fmt.Println("Send any message to your bot in Telegram to auto-detect your chat ID.")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=30", token)
	for {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("getUpdates error: %v, retrying...", err)
			time.Sleep(3 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var result struct {
			OK     bool `json:"ok"`
			Result []struct {
				Message struct {
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
				} `json:"message"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("parse error: %v, retrying...", err)
			time.Sleep(3 * time.Second)
			continue
		}
		if result.OK && len(result.Result) > 0 {
			cid := result.Result[len(result.Result)-1].Message.Chat.ID
			if cid != 0 {
				chatID := fmt.Sprintf("%d", cid)
				log.Printf("discovered chat ID: %s", chatID)
				fmt.Printf("\n✅ Chat ID discovered: %s\n", chatID)
				fmt.Println("Add to .env: TELEGRAM_CHAT_ID=" + chatID)
				return chatID
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	accountsPath := os.Getenv("ACCOUNTS_FILE")
	oauthPort := os.Getenv("OAUTH_RELAY_PORT")
	pollStr := os.Getenv("POLL_INTERVAL")

	if token == "" {
		fmt.Fprintln(os.Stderr, "TELEGRAM_BOT_TOKEN is required")
		os.Exit(1)
	}

	// Auto-detect chat ID if not provided
	if chatID == "" {
		chatID = discoverChatID(token)
	}

	if accountsPath == "" {
		home, _ := os.UserHomeDir()
		accountsPath = filepath.Join(home, ".config", "opencode", "antigravity-accounts.json")
	}

	pollInterval := 5 * time.Second
	if pollStr != "" {
		if secs, err := strconv.Atoi(pollStr); err == nil && secs > 0 {
			pollInterval = time.Duration(secs) * time.Second
		}
	}

	log.Printf("starting antigravity telegram bot")
	log.Printf("  accounts: %s", accountsPath)
	log.Printf("  poll interval: %s", pollInterval)

	bot := newTelegramBot(token, chatID)
	accts := newAccountsManager(accountsPath)
	handler := newBotHandler(bot, accts, chatID)

	// Validate accounts file is readable
	storage, err := accts.load()
	if err != nil {
		log.Fatalf("cannot read accounts file: %v", err)
	}
	log.Printf("  loaded %d accounts, active: [%d]", len(storage.Accounts), storage.ActiveIndex)

	// Take initial snapshot for change detection
	prevSnapshot := takeSnapshot(storage)

	// Start file watcher
	watcher := newFileWatcher(accountsPath, pollInterval, func() {
		storage, err := accts.load()
		if err != nil {
			log.Printf("file watcher: load error: %v", err)
			return
		}
		currSnapshot := takeSnapshot(storage)
		changes := detectChanges(prevSnapshot, currSnapshot, storage)
		for _, msg := range changes {
			bot.notify(msg)
		}
		prevSnapshot = currSnapshot
	})
	watcher.start()
	defer watcher.close()

	// Start OAuth relay if configured
	if oauthPort != "" {
		go startOAuthRelay(oauthPort, bot, chatID)
	}

	// Send startup notification
	enabledCount := 0
	verifyCount := 0
	for i := range storage.Accounts {
		if storage.Accounts[i].isEnabled() {
			enabledCount++
		}
		if storage.Accounts[i].needsVerification() {
			verifyCount++
		}
	}
	startupMsg := fmt.Sprintf("🤖 <b>Antigravity Bot Online</b>\n\n%d accounts (%d enabled)", len(storage.Accounts), enabledCount)
	if verifyCount > 0 {
		startupMsg += fmt.Sprintf("\n⚠️ %d accounts need verification — /verify", verifyCount)
	}
	bot.notify(startupMsg)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start polling for Telegram updates
	log.Printf("  polling for telegram updates...")
	go func() {
		backoff := time.Second
		for {
			updates, err := bot.getUpdates()
			if err != nil {
				log.Printf("poll error: %v (retry in %s)", err, backoff)
				time.Sleep(backoff)
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			backoff = time.Second

			for _, u := range updates {
				if u.UpdateID >= bot.offset {
					bot.offset = u.UpdateID + 1
				}
				if u.Message != nil && u.Message.Text != "" {
					chatIDStr := strconv.FormatInt(u.Message.Chat.ID, 10)
					handler.handleCommand(chatIDStr, u.Message.Text)
				}
			}
		}
	}()

	<-sigCh
	log.Println("shutting down...")
	bot.notify("🔴 Antigravity Bot shutting down")
}

// Compile-time check: this file uses only stdlib.
var _ = regexp.Compile
