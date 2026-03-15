# opencode-agent-gateway

`opencode-agent-gateway` is a small Go HTTP gateway that accepts job requests, forwards prompts to an OpenCode `serve` instance, and exposes job lifecycle APIs for polling, progress checks, and abort.

It supports:
- synchronous job execution (`mode: "run"`)
- asynchronous job execution (`mode: "async"`) backed by OpenCode session events
- callback delivery on terminal states (`completed`, `failed`, `cancelled`)

## Prerequisites

- Go 1.22+
- A running OpenCode `serve` instance (default: `http://localhost:3456`)
- Optional: 1Password CLI (`op`) if you pass secrets as `op://...` references

## Configuration

The gateway reads CLI flags and a few environment defaults.

| Flag | Description | Default / Source |
|---|---|---|
| `--listen` | HTTP listen address | `127.0.0.1:7800` (`GATEWAY_LISTEN`) |
| `--opencode-url` | Base URL of OpenCode server | `http://localhost:3456` (`OPENCODE_URL`) |
| `--opencode-pass` | OpenCode password (operator-facing name) | See note below |
| `--default-model` | Fallback model in `provider/model` form | `openai/gpt-5.4` (`GATEWAY_DEFAULT_MODEL`) |
| `--max-concurrent` | Max concurrent worker count | `5` (`GATEWAY_MAX_CONCURRENT`) |
| `--callback-timeout` | Callback HTTP timeout | `30s` (currently fixed in code) |

Notes:
- Current flag in this codebase is `--opencode-password` (mapped from `OPENCODE_SERVER_PASSWORD`).
- Callback timeout is currently fixed to `30s` in `main.go`; there is no dedicated CLI flag yet.

## API Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Health response + OpenCode reachability |
| `GET` | `/jobs` | List jobs (summary view) |
| `POST` | `/jobs` | Create a job |
| `GET` | `/jobs/{id}` | Full job details |
| `GET` | `/jobs/{id}/status` | Lightweight status view |
| `GET` | `/jobs/{id}/progress` | Progress metadata |
| `POST` | `/jobs/{id}/abort` | Request job cancellation |

## Job Modes

### Sync mode (`mode: "run"`)
- Default when `mode` is omitted
- Uses OpenCode session message API and waits for the model response
- Sets job result to returned text

### Async mode (`mode: "async"`)
- Triggers OpenCode `prompt_async`
- Tracks session status events until terminal idle-after-busy progression
- Marks completion with gateway async completion message

## Callback Payload v2

On terminal state, gateway POSTs JSON to `callback_url`.

```json
{
  "job_id": "job-123",
  "status": "completed",
  "model": "openai/gpt-5.4",
  "mode": "run",
  "format": {"type":"json_schema","json_schema":{"name":"answer","schema":{"type":"object"}}},
  "session_id": "session-1",
  "result": "final answer",
  "error": "",
  "duration_ms": 842,
  "completed_at": "2026-03-15T01:23:45Z"
}
```

Field summary:
- `job_id`: gateway job ID
- `status`: `completed`, `failed`, or `cancelled`
- `model`: model used by the job
- `mode`: `run` or `async`
- `format`: optional structured output format passed to OpenCode
- `session_id`: OpenCode session identifier when available
- `result`: model/gateway terminal result text
- `error`: failure/cancellation reason on non-success states
- `duration_ms`: elapsed processing time in milliseconds
- `completed_at`: RFC3339 UTC timestamp

## Usage Examples

### Run gateway

Requested operator form:

```bash
go run . --opencode-url http://localhost:3456 --opencode-pass mypass
```

Current binary flag form:

```bash
go run . --opencode-url http://localhost:3456 --opencode-password mypass
```

### Create sync job

```bash
curl -sS -X POST http://127.0.0.1:7800/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "job_id": "job-sync-1",
    "prompt": "Summarize the latest changelog.",
    "model": "openai/gpt-5.4",
    "mode": "run",
    "callback_url": "http://127.0.0.1:9000/callback"
  }'
```

### Create async job

```bash
curl -sS -X POST http://127.0.0.1:7800/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "job_id": "job-async-1",
    "prompt": "Run the asynchronous workflow.",
    "mode": "async",
    "callback_url": "http://127.0.0.1:9000/callback"
  }'
```

### Check status and details

```bash
curl -sS http://127.0.0.1:7800/jobs/job-sync-1/status
curl -sS http://127.0.0.1:7800/jobs/job-sync-1
curl -sS http://127.0.0.1:7800/jobs/job-sync-1/progress
```

### Abort a job

```bash
curl -sS -X POST http://127.0.0.1:7800/jobs/job-async-1/abort
```

## Testing

Run the package tests:

```bash
go test ./...
```

Repository verification command used in CI/local checks:

```bash
GOWORK=off go test -count=1 -timeout 30s ./...
GOWORK=off go vet ./...
```
