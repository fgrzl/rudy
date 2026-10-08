# OpenCode

OpenCode connects to Rudy at `http://localhost:8080/v1`. Rudy adds indexed
workspace context and forwards requests to the Compose-managed Ollama service using Qwen3-Coder 30B-A3B.

## Run the stack

```bash
docker compose up --build -d
curl -fsS http://localhost:8080/v1/models
opencode
```

The root [opencode.json](../opencode.json) configures the `rudy` provider and selects
`rudy/hf.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF:Q3_K_M`. OpenCode must be installed separately.

For another project, copy that configuration into its root, or merge the provider
into `~/.config/opencode/opencode.json`. Set the workspace mount in `compose.yml`
to the code you want Rudy to index, then rebuild the index:

```bash
curl -fsS -X POST http://localhost:8080/api/index/rebuild
```

The model runs with a 16,384-token context and one inference slot. OpenCode
advertises 14,336 tokens to reserve 2,048 tokens for Rudy's retrieval and message
framing. Rudy caps retrieved context at 1,024 UTF-8 bytes. OpenCode enables automatic
compaction, tool-output pruning, and a 2,048-token compaction reserve. Both normal
chat and compaction use the local model.

Rudy preserves tool-call IDs, reasoning fields, and multipart message content,
and forwards streaming responses incrementally. Requests have a five-minute
upstream deadline. Model capacity, token estimates, large individual tool outputs,
and Qwen3-Coder 30B-A3B's tool-use ability can still limit longer coding sessions.

After changing these settings, restart OpenCode. For an existing overflowing
session, use `/compact` or start a fresh session with `/new`.

To use Ollama directly, expose its port in `compose.yml` and point the provider at
`http://localhost:11434/v1`; this bypasses Rudy's workspace retrieval.
