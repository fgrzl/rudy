# OpenCode

OpenCode is the primary client for coding against LocalAgent.
It talks to the local service over the OpenAI-compatible HTTP API.

## Run the stack

```bash
docker compose up --build
```

Pull the configured chat models manually before using OpenCode if they are not already present:

```bash
docker compose exec ollama ollama pull qwen2.5-coder:7b-instruct
docker compose exec ollama ollama pull qwen2.5-coder:14b-instruct
docker compose exec ollama ollama pull nomic-embed-text
```

## Configure OpenCode

Point OpenCode at the local service base URL:

```json
{
  "baseUrl": "http://localhost:8080/v1",
  "model": "qwen2.5-coder:7b-instruct"
}
```

Use any of the model IDs Rudy advertises unless you intentionally override it.

If your OpenCode setup expects the root API paths, Rudy also accepts `/chat/completions`, `/models`, and `/embeddings` without the `/v1` prefix.

## Service defaults

- `LOCALAGENT_HTTP_ADDR=:8080`
- `LOCALAGENT_OLLAMA_URL=http://ollama:11434`
- `LOCALAGENT_CHAT_MODEL=qwen2.5-coder:7b-instruct`
- `LOCALAGENT_EMBEDDING_MODEL=nomic-embed-text`
- `LOCALAGENT_WORKSPACE_DIR=/workspace`
- `LOCALAGENT_DATA_DIR=/data`

## Direct endpoints

OpenCode uses `/v1/chat/completions` for coding conversations.
The backend also exposes these helpers if you want to drive it directly:

- `GET /healthz`
- `GET /api/search?q=golang`
- `POST /api/index/rebuild`
- `POST /api/index/file`