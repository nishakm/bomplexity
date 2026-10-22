# agent

An "ask the guestbook" agent for the demo cluster. It is a separate app with
its own Go module, images and SBOMs; `goapp/` and `kind/` are not changed.

```
POST /ask {"question": "..."}  ->  {"answer": "..."}
```

| Path             | What it is                                                              |
|------------------|-------------------------------------------------------------------------|
| `main.go`, `ask.go`, `llm.go` | The agent (`guestbook-agent:v1`): `/ask`, the tool loop (max 5 rounds), an MCP client, and a plain `net/http` client for an OpenAI-compatible model API. |
| `mcp/`           | The MCP server (`guestbook-mcp:v1`): streamable HTTP on `:8081/mcp`, read-only tools `list_entries` (optional `limit`, default 20) and `entry_stats`. They only call the guestbook's `GET /entries`. |
| `Dockerfile`, `Dockerfile.mcp` | Same pinned golang/alpine bases as `goapp/`, non-root uid 10001. |
| `agent.yaml`     | All Kubernetes resources: both Deployments and Services, the `agent-prompt` ConfigMap, and a replacement `nginx-config` that routes `/ask` to the agent. |
| `sbom-stages.sh` | Component counts and diff behind `make scorecard` and `make sbom-diff` (uses syft on the images). |

## How it plugs in

```
curl :8080/ask -> nginx -> guestbook-agent -> model API (Gemini)
                               |
                               +-- MCP --> guestbook-mcp --GET /entries--> guestbook -> Postgres
```

## Run it

Needs the v1 demo running (`make up` in `../kind`) and a Google AI Studio key:
`cp .env.example .env` and set `GOOGLE_API_KEY` (git-ignored; it becomes the
`llm-credentials` Secret, never YAML).

```
make agent-up       # build both images, kind load, secret, apply, wait for rollouts
make agent-down     # remove the agent, restore the v1 nginx config
make reset-entries  # truncate the guestbook and seed 4 friendly entries
make scorecard      # component counts: v1 app /sbom, v1 images, v2 images
make sbom-diff      # components in the v2 images that v1 does not have
make test           # go vet + unit tests
```

## Settings (agent environment)

| Variable             | Default                                                   |
|----------------------|-----------------------------------------------------------|
| `LLM_API_KEY`        | none; `/ask` answers 503 if unset                          |
| `LLM_BASE_URL`       | `https://generativelanguage.googleapis.com/v1beta/openai` |
| `LLM_MODEL`          | `gemini-3.5-flash-lite`                                   |
| `SYSTEM_PROMPT_FILE` | none (mounted from the `agent-prompt` ConfigMap)          |
| `MCP_SERVER_URL`     | `http://guestbook-mcp:8081/mcp`                           |

The demo script is in `../docs/act3-demo.md`.
