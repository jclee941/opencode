package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type config struct {
	listenAddr     string
	opencodeURL    string
	opencodePass   string
	defaultModel   string
	maxConcurrent  int
	callbackTimout time.Duration
}

type job struct {
	ID             string          `json:"job_id"`
	Prompt         string          `json:"prompt"`
	Repo           string          `json:"repo"`
	Model          string          `json:"model"`
	Mode           string          `json:"mode"`
	Format         json.RawMessage `json:"format,omitempty"`
	CallbackURL    string          `json:"callback_url"`
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"`
	SessionID      string          `json:"session_id,omitempty"`
	Result         string          `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	DurationMs     int64           `json:"duration_ms,omitempty"`
}

type jobSummary struct {
	ID        string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type jobRequest struct {
	JobID          string          `json:"job_id"`
	Prompt         string          `json:"prompt"`
	Repo           string          `json:"repo"`
	Model          string          `json:"model"`
	Mode           string          `json:"mode"`
	Format         json.RawMessage `json:"format,omitempty"`
	CallbackURL    string          `json:"callback_url"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type callbackPayload struct {
	JobID       string          `json:"job_id"`
	Status      string          `json:"status"`
	Model       string          `json:"model,omitempty"`
	Mode        string          `json:"mode,omitempty"`
	Format      json.RawMessage `json:"format,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	Result      string          `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	DurationMs  int64           `json:"duration_ms"`
	CompletedAt string          `json:"completed_at"`
}

type jobStore struct {
	mu             sync.Mutex
	jobs           map[string]*job
	idempotencyMap map[string]string
}

func newJobStore() *jobStore {
	return &jobStore{jobs: make(map[string]*job), idempotencyMap: make(map[string]string)}
}

func cloneJob(in *job) *job {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Format) > 0 {
		out.Format = append(json.RawMessage(nil), in.Format...)
	}
	if in.StartedAt != nil {
		t := *in.StartedAt
		out.StartedAt = &t
	}
	if in.CompletedAt != nil {
		t := *in.CompletedAt
		out.CompletedAt = &t
	}
	return &out
}

func (s *jobStore) createOrGet(req jobRequest, defaultModel string) (*job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idKey := strings.TrimSpace(req.IdempotencyKey)
	if idKey != "" {
		if existingID, ok := s.idempotencyMap[idKey]; ok {
			if existing, ok := s.jobs[existingID]; ok {
				return cloneJob(existing), false, nil
			}
		}
	}

	jobID := strings.TrimSpace(req.JobID)
	if jobID == "" {
		return nil, false, errors.New("job_id is required")
	}
	if existing, ok := s.jobs[jobID]; ok {
		return cloneJob(existing), false, nil
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = defaultModel
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "run"
	}

	now := time.Now().UTC()
	created := &job{
		ID:             jobID,
		Prompt:         strings.TrimSpace(req.Prompt),
		Repo:           strings.TrimSpace(req.Repo),
		Model:          model,
		Mode:           mode,
		Format:         append(json.RawMessage(nil), req.Format...),
		CallbackURL:    strings.TrimSpace(req.CallbackURL),
		IdempotencyKey: idKey,
		Status:         "queued",
		CreatedAt:      now,
	}
	s.jobs[jobID] = created
	if idKey != "" {
		s.idempotencyMap[idKey] = jobID
	}
	return cloneJob(created), true, nil
}

func (s *jobStore) get(id string) (*job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(v), true
}

func (s *jobStore) listSummaries() []jobSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	summaries := make([]jobSummary, 0, len(s.jobs))
	for _, j := range s.jobs {
		summaries = append(summaries, jobSummary{ID: j.ID, Status: j.Status, CreatedAt: j.CreatedAt})
	}
	sort.Slice(summaries, func(i, k int) bool {
		return summaries[i].CreatedAt.After(summaries[k].CreatedAt)
	})
	return summaries
}

func (s *jobStore) update(id string, mutate func(*job)) (*job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	mutate(j)
	return cloneJob(j), true
}

type gateway struct {
	cfg            config
	store          *jobStore
	client         *opencodeClient
	callbackClient *http.Client
	sem            chan struct{}
	ctx            context.Context
	wg             sync.WaitGroup
}

func newGateway(ctx context.Context, cfg config) *gateway {
	return &gateway{
		cfg:            cfg,
		store:          newJobStore(),
		client:         newOpencodeClient(cfg.opencodeURL, cfg.opencodePass, &http.Client{}),
		callbackClient: &http.Client{Timeout: cfg.callbackTimout},
		sem:            make(chan struct{}, cfg.maxConcurrent),
		ctx:            ctx,
	}
}

func (g *gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	reachable := g.client.checkReachable(r.Context())
	writeJSON(w, map[string]any{"status": "ok", "opencode_reachable": reachable})
}

func (g *gateway) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		g.handleCreateJob(w, r)
	case http.MethodGet:
		writeJSON(w, g.store.listSummaries())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *gateway) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, 1<<20)
	var req jobRequest
	if err := json.NewDecoder(limited).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := validateFormat(req.Format); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.JobID) == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CallbackURL) == "" {
		http.Error(w, "callback_url is required", http.StatusBadRequest)
		return
	}

	createdJob, created, err := g.store.createOrGet(req, g.cfg.defaultModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !created {
		fmt.Fprintf(os.Stderr, "idempotent hit: job_id=%s idempotency_key=%s status=%s\n", createdJob.ID, createdJob.IdempotencyKey, createdJob.Status)
		writeJSON(w, createdJob)
		return
	}

	fmt.Fprintf(os.Stderr, "job queued: job_id=%s repo=%s model=%s\n", createdJob.ID, createdJob.Repo, createdJob.Model)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"job_id": createdJob.ID, "status": "queued"})

	g.wg.Add(1)
	go func(jobID string) {
		defer g.wg.Done()
		if createdJob.Mode == "async" {
			g.runJobAsync(jobID)
			return
		}
		g.runJob(jobID)
	}(createdJob.ID)
}

func (g *gateway) handleJobByID(w http.ResponseWriter, r *http.Request) {
	id, subresource, ok := parseJobReadPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch subresource {
	case "":
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		g.handleJobDetails(w, r, id)
	case "status":
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		g.handleJobStatus(w, r, id)
	case "progress":
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		g.handleJobProgress(w, r, id)
	case "abort":
		g.handleAbort(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (g *gateway) runJob(jobID string) {
	select {
	case g.sem <- struct{}{}:
	case <-g.ctx.Done():
		fmt.Fprintf(os.Stderr, "skip job, shutdown in progress: job_id=%s\n", jobID)
		cancelled := time.Now().UTC()
		updated, ok := g.store.update(jobID, func(j *job) {
			j.Status = "cancelled"
			j.Error = "gateway shutdown before execution"
			j.CompletedAt = &cancelled
		})
		if ok {
			if err := g.postCallback(updated); err != nil {
				fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
			}
		}
		return
	}
	defer func() { <-g.sem }()

	started := time.Now().UTC()
	current, ok := g.store.update(jobID, func(j *job) {
		j.Status = "running"
		j.StartedAt = &started
	})
	if !ok {
		fmt.Fprintf(os.Stderr, "job not found while starting: job_id=%s\n", jobID)
		return
	}

	providerID, modelID, err := splitModel(current.Model)
	if err != nil {
		g.failJob(current, err)
		return
	}

	sessionID, err := g.client.createSession(g.ctx, "agent-gateway: "+current.ID)
	if err != nil {
		g.failJob(current, err)
		return
	}
	current, _ = g.store.update(current.ID, func(j *job) {
		j.SessionID = sessionID
	})

	messageCtx, messageCancel := context.WithTimeout(g.ctx, 20*time.Minute)
	result, err := g.client.sendMessage(messageCtx, sessionID, current.Prompt, providerID, modelID, current.Format)
	messageCancel()
	if err != nil {
		g.failJob(current, err)
		return
	}

	completed := time.Now().UTC()
	durationMs := completed.Sub(started).Milliseconds()
	updated, _ := g.store.update(current.ID, func(j *job) {
		j.Status = "completed"
		j.Result = result
		j.Error = ""
		j.CompletedAt = &completed
		j.DurationMs = durationMs
	})
	fmt.Fprintf(os.Stderr, "job completed: job_id=%s session_id=%s duration_ms=%d\n", updated.ID, updated.SessionID, updated.DurationMs)

	if err := g.postCallback(updated); err != nil {
		fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
	}
}

func (g *gateway) failJob(existing *job, cause error) {
	completed := time.Now().UTC()
	durationMs := int64(0)
	if existing != nil && existing.StartedAt != nil {
		durationMs = completed.Sub(*existing.StartedAt).Milliseconds()
	}
	updated, ok := g.store.update(existing.ID, func(j *job) {
		j.Status = "failed"
		j.Error = cause.Error()
		j.CompletedAt = &completed
		j.DurationMs = durationMs
	})
	if !ok {
		fmt.Fprintf(os.Stderr, "job not found while failing: job_id=%s err=%v\n", existing.ID, cause)
		return
	}
	fmt.Fprintf(os.Stderr, "job failed: job_id=%s err=%v\n", updated.ID, cause)
	if err := g.postCallback(updated); err != nil {
		fmt.Fprintf(os.Stderr, "callback failed: job_id=%s err=%v\n", updated.ID, err)
	}
}

func (g *gateway) postCallback(j *job) error {
	if j == nil || strings.TrimSpace(j.CallbackURL) == "" {
		return nil
	}
	completedAt := ""
	if j.CompletedAt != nil {
		completedAt = j.CompletedAt.UTC().Format(time.RFC3339)
	}
	payload := callbackPayload{
		JobID:       j.ID,
		Status:      j.Status,
		Model:       j.Model,
		Mode:        j.Mode,
		Format:      append(json.RawMessage(nil), j.Format...),
		SessionID:   j.SessionID,
		Result:      j.Result,
		Error:       j.Error,
		DurationMs:  j.DurationMs,
		CompletedAt: completedAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, j.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.callbackClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	fmt.Fprintf(os.Stderr, "callback delivered: job_id=%s status=%s\n", j.ID, j.Status)
	return nil
}

func splitModel(model string) (string, string, error) {
	v := strings.TrimSpace(model)
	parts := strings.SplitN(v, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid model format %q, expected provider/model", model)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func extractResultText(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}

	// OpenCode message endpoint returns JSON with structure:
	// { "info": {...}, "parts": [ {"type":"text","text":"..."}, ... ] }
	// We want to collect all parts with type="text" and concatenate their text fields.
	var envelope struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal([]byte(trimmed), &envelope); err == nil && len(envelope.Parts) > 0 {
		var texts []string
		for _, p := range envelope.Parts {
			if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
				texts = append(texts, strings.TrimSpace(p.Text))
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n\n")
		}
	}

	// Fallback for multi-line NDJSON (streaming responses).
	lines := strings.Split(trimmed, "\n")
	var lastText string
	for _, line := range lines {
		v := strings.TrimSpace(line)
		if v == "" {
			continue
		}
		var lineEnv struct {
			Parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"parts"`
		}
		if err := json.Unmarshal([]byte(v), &lineEnv); err == nil {
			for _, p := range lineEnv.Parts {
				if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
					lastText = strings.TrimSpace(p.Text)
				}
			}
		}
	}
	if lastText != "" {
		return lastText
	}

	// Final fallback: return raw body trimmed.
	return trimmed
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

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	listenAddr := flag.String("listen", envOrDefault("GATEWAY_LISTEN", "127.0.0.1:7800"), "HTTP listen address")
	opencodeURL := flag.String("opencode-url", envOrDefault("OPENCODE_URL", "http://localhost:3456"), "OpenCode server URL")
	opencodePasswordFlag := flag.String("opencode-password", envOrDefault("OPENCODE_SERVER_PASSWORD", ""), "OpenCode Basic Auth password or op:// reference")
	defaultModel := flag.String("default-model", envOrDefault("GATEWAY_DEFAULT_MODEL", "openai/gpt-5.4"), "Default model provider/model")
	maxConcurrent := flag.Int("max-concurrent", 5, "Maximum concurrent job workers")
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_MAX_CONCURRENT")); raw != "" {
		if parsed, err := strconvAtoi(raw); err == nil {
			*maxConcurrent = parsed
		}
	}
	flag.Parse()

	resolvedPassword, err := resolveValue(*opencodePasswordFlag, "")
	if err != nil {
		fail(err)
	}

	if strings.TrimSpace(*defaultModel) == "" {
		fail(errors.New("default model cannot be empty"))
	}
	if _, _, err := splitModel(*defaultModel); err != nil {
		fail(fmt.Errorf("invalid default model: %w", err))
	}
	if *maxConcurrent <= 0 {
		fail(errors.New("max-concurrent must be >= 1"))
	}

	cfg := config{
		listenAddr:     strings.TrimSpace(*listenAddr),
		opencodeURL:    strings.TrimRight(strings.TrimSpace(*opencodeURL), "/"),
		opencodePass:   resolvedPassword,
		defaultModel:   strings.TrimSpace(*defaultModel),
		maxConcurrent:  *maxConcurrent,
		callbackTimout: 30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gateway := newGateway(ctx, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", gateway.handleHealth)
	mux.HandleFunc("/jobs", gateway.handleJobs)
	mux.HandleFunc("/jobs/", gateway.handleJobByID)

	server := &http.Server{Addr: cfg.listenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	fmt.Fprintf(os.Stderr, "opencode-agent-gateway listening on %s opencode=%s max_concurrent=%d\n", cfg.listenAddr, cfg.opencodeURL, cfg.maxConcurrent)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Fprintln(os.Stderr, "shutdown signal received")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "server shutdown error: %v\n", err)
	}
	gateway.wg.Wait()
	fmt.Fprintln(os.Stderr, "shutdown complete")
}

func strconvAtoi(v string) (int, error) {
	n := 0
	if v == "" {
		return 0, errors.New("empty")
	}
	for i, r := range v {
		if i == 0 && r == '+' {
			continue
		}
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer %q", v)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
