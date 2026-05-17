# LocalAgent

LocalAgent is a local coding backend for OpenCode and other OpenAI-compatible clients.

It combines:
- Go for the service layer
- Pebble-backed `github.com/fgrzl/kv` search and index storage
- Ollama for local models
- a workspace indexer that chunks files into searchable entities
- mux for the HTTP API layer

## What it does now

- Scans a local workspace and indexes text files into Pebble through `github.com/fgrzl/kv`
- Exposes keyword search at `/api/search`
- Exposes OpenAI-compatible chat completions at `/v1/chat/completions`
- Injects relevant indexed workspace context into chat prompts automatically
- Advertises a curated chat model catalog through `/v1/models` and proxies Ollama embeddings

## Environment

Defaults are chosen for local Docker use:

- `LOCALAGENT_HTTP_ADDR=:8080`
- `LOCALAGENT_OLLAMA_URL=http://ollama:11434`
- `LOCALAGENT_WORKSPACE_DIR=/workspace`
- `LOCALAGENT_DATA_DIR=/data`
- `LOCALAGENT_CHAT_MODEL=qwen2.5-coder:7b-instruct`
- `LOCALAGENT_SUPPORTED_CHAT_MODELS=qwen2.5-coder:7b-instruct,qwen2.5-coder:14b-instruct`
- `LOCALAGENT_EMBEDDING_MODEL=nomic-embed-text`
- `LOCALAGENT_SEARCH_HIT_LIMIT=8`
- `LOCALAGENT_CONTEXT_CHUNK_LIMIT=4`
- `LOCALAGENT_REQUEST_TIMEOUT=90s`

## Run with Docker

```bash
docker compose up --build
```

Pull the chat models Rudy advertises before starting the app if they are not already present in your Ollama volume:

```bash
docker compose exec ollama ollama pull qwen2.5-coder:7b-instruct
docker compose exec ollama ollama pull qwen2.5-coder:14b-instruct
docker compose exec ollama ollama pull nomic-embed-text
```

The app container mounts:
- `./workspace` for the code you want indexed
- `./data` for Pebble data and the local index manifest

## Model pulls

If you run Ollama outside Docker Compose, pull the models Rudy advertises before using OpenCode:

```bash
docker compose exec ollama ollama pull qwen2.5-coder:7b-instruct
docker compose exec ollama ollama pull qwen2.5-coder:14b-instruct
docker compose exec ollama ollama pull nomic-embed-text
```

## OpenCode

OpenCode is the primary coding client for this repo. Point it at `http://localhost:8080/v1` and use the same chat model as the service defaults.

See [opencode/README.md](opencode/README.md) for the exact setup.

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
