# LocalAgent

LocalAgent is a local coding backend for OpenCode and other OpenAI-compatible clients.

It combines:
- Go for the service layer
- Pebble-backed `github.com/fgrzl/kv` search and index storage
- Ollama with Qwen3-Coder 30B-A3B for local inference
- a workspace indexer that chunks files into searchable entities
- mux for the HTTP API layer

## What it does now

- Scans a local workspace and indexes text files into Pebble through `github.com/fgrzl/kv`
- Exposes keyword search at `/api/search`
- Exposes OpenAI-compatible chat completions at `/v1/chat/completions`
- Injects relevant indexed workspace context into chat prompts automatically
- Advertises a curated chat model catalog through `/v1/models` and proxies Ollama embeddings

## Run with Docker Desktop

The Compose stack runs both Rudy and Ollama and uses
`hf.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF:Q3_K_M` (Q3_K_M, about a 14.7 GB download).
The model weights occupy about 13.7 GiB of memory; allocate at least 24 GiB to Docker Desktop so there is room
for context and inference buffers.

```bash
docker compose up --build -d
```

Compose starts Ollama and pulls the model automatically before starting Rudy.
The API is available at `http://localhost:8080/v1` and, on a trusted LAN, at
`http://<host-ip>:8080/v1`. Allow at least 30 GB of free disk space for downloading
and importing the model.

The app container mounts:

- `./workspace` for the code you want indexed
- `./data` for Pebble data and the local index manifest

Put code in `workspace/`, then rebuild its index:

```bash
curl -fsS -X POST http://localhost:8080/api/index/rebuild
```

Verify inference through Rudy:

```bash
curl -fsS http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"hf.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF:Q3_K_M","messages":[{"role":"user","content":"Write a Go function that adds two integers."}],"max_tokens":256,"stream":false}'
```

## Backend configuration

Compose overrides the service's Ollama defaults with:

- `LOCALAGENT_OLLAMA_URL=http://ollama:11434`
- `LOCALAGENT_CHAT_MODEL=hf.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF:Q3_K_M`
- `LOCALAGENT_SUPPORTED_CHAT_MODELS=hf.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF:Q3_K_M`
- `LOCALAGENT_WORKSPACE_DIR=/workspace`
- `LOCALAGENT_DATA_DIR=/data`
- `LOCALAGENT_CONTEXT_MAX_BYTES=1024`
- `LOCALAGENT_REQUEST_TIMEOUT=300s`

`LOCALAGENT_OLLAMA_URL` accepts an OpenAI-compatible backend. Rudy appends
`/v1/models`, `/v1/chat/completions`, or `/v1/embeddings`.

Workspace retrieval currently uses keyword search and does not require embeddings.
The embeddings proxy needs a separately installed embedding model; Qwen3-Coder 30B-A3B is
a chat model.

## OpenCode

The project-level [opencode.json](opencode.json) points OpenCode at Rudy and selects
Qwen3-Coder 30B-A3B. Run `opencode` from this directory once the stack is running.
See [opencode/README.md](opencode/README.md) for client configuration and limitations.

## Helpful endpoints

- `GET /healthz`
- `GET /api/search?q=golang`
- `POST /api/index/rebuild`
- `POST /api/index/file`
- `POST /v1/chat/completions`

## Next steps

- Add file watching for incremental reindexing
- Improve snippets around matched text
- Add optional semantic/vector retrieval
