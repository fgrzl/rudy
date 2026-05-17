# LocalAgent

LocalAgent is a local coding assistant stack for your machine.

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
- Proxies Ollama endpoints such as `/v1/models` and `/v1/embeddings`

## Environment

Defaults are chosen for local Docker use:

- `LOCALAGENT_HTTP_ADDR=:8080`
- `LOCALAGENT_OLLAMA_URL=http://ollama:11434`
- `LOCALAGENT_WORKSPACE_DIR=/workspace`
- `LOCALAGENT_DATA_DIR=/data`
- `LOCALAGENT_CHAT_MODEL=qwen2.5-coder:7b-instruct`
- `LOCALAGENT_EMBEDDING_MODEL=nomic-embed-text`

## Run with Docker

```bash
docker compose up --build
```

The app container mounts:
- `./workspace` for the code you want indexed
- `./data` for Pebble data and the local index manifest

## First model pull

After Ollama starts, pull the models you want:

```bash
docker compose exec ollama ollama pull qwen2.5-coder:7b-instruct
docker compose exec ollama ollama pull nomic-embed-text
```

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
