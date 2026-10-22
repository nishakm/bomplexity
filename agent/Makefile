# Agent add-on for the demo cluster: `make agent-up` / `make agent-down`
# Needs the v1 demo running first (`make up` in ../kind).

CLUSTER_NAME ?= sbom-demo
NAMESPACE    ?= sbom-demo
AGENT_IMAGE  ?= guestbook-agent:v1
MCP_IMAGE    ?= guestbook-mcp:v1
KIND_DIR     ?= ../kind
ENV_FILE     ?= .env
URL          ?= http://localhost:8080
KUBECTL      := kubectl --context kind-$(CLUSTER_NAME) -n $(NAMESPACE)

.DEFAULT_GOAL := help
.PHONY: help agent-up agent-down images load secret reset-entries scorecard sbom-diff test

## help: list all targets
help:
	@awk '/^## [a-z-]+: /{ name=$$2; sub(/:$$/,"",name); d=$$0; sub(/^## [a-z-]+: /,"",d); printf "  %-14s %s\n", name, d }' $(MAKEFILE_LIST)

## test: go vet and unit tests
test:
	go vet ./...
	go test ./...

## images: build the agent and MCP server images
images:
	@echo "==> Building $(AGENT_IMAGE) and $(MCP_IMAGE)"
	docker build -q -t $(AGENT_IMAGE) -f Dockerfile .
	docker build -q -t $(MCP_IMAGE) -f Dockerfile.mcp .

## load: load both images into the kind cluster
load:
	@echo "==> Loading images into kind cluster '$(CLUSTER_NAME)'"
	kind load docker-image $(AGENT_IMAGE) $(MCP_IMAGE) --name $(CLUSTER_NAME)

## secret: create/update the llm-credentials Secret from GOOGLE_API_KEY (env or .env)
secret:
	@key="$${GOOGLE_API_KEY:-}"; \
	if [ -z "$$key" ] && [ -f $(ENV_FILE) ]; then set -a; . ./$(ENV_FILE); set +a; key="$${GOOGLE_API_KEY:-}"; fi; \
	if [ -z "$$key" ] || [ "$$key" = change-me ]; then \
	  echo "ERROR: GOOGLE_API_KEY is not set. Export it, or put it in $(ENV_FILE) (see .env.example)." >&2; exit 1; fi; \
	echo "==> Applying Secret llm-credentials"; \
	printf %s "$$key" | $(KUBECTL) create secret generic llm-credentials --from-file=LLM_API_KEY=/dev/stdin --dry-run=client -o yaml | $(KUBECTL) apply -f - >/dev/null

## agent-up: build, load, create the secret, apply agent.yaml and wait for the rollouts
agent-up:
	@command -v docker >/dev/null || { echo "ERROR: docker is not installed" >&2; exit 1; }
	@command -v kind >/dev/null || { echo "ERROR: kind is not installed" >&2; exit 1; }
	@$(KUBECTL) get deployment/guestbook >/dev/null 2>&1 || { \
	  echo "ERROR: the v1 demo is not running in '$(CLUSTER_NAME)'. Run 'make up' in $(KIND_DIR) first." >&2; exit 1; }
	@$(MAKE) --no-print-directory secret
	@$(MAKE) --no-print-directory images
	@$(MAKE) --no-print-directory load
	@echo "==> Applying agent.yaml"
	@existed=$$($(KUBECTL) get deployment/guestbook-agent -o name 2>/dev/null); \
	$(KUBECTL) apply -f agent.yaml; \
	if [ -n "$$existed" ]; then $(KUBECTL) rollout restart deployment/guestbook-mcp deployment/guestbook-agent; fi
	@# nginx mounts its config with subPath, so it needs a restart to see /ask.
	@$(KUBECTL) rollout restart deployment/nginx
	@echo "==> Waiting for rollouts"
	@$(KUBECTL) rollout status deployment/guestbook-mcp --timeout=120s
	@$(KUBECTL) rollout status deployment/guestbook-agent --timeout=120s
	@$(KUBECTL) rollout status deployment/nginx --timeout=120s
	@for i in $$(seq 1 60); do \
	  code=$$(curl -s -o /dev/null -w '%{http_code}' -X POST $(URL)/ask -d '{}'); \
	  [ "$$code" = 400 ] && { echo "==> Agent is up: POST $(URL)/ask"; exit 0; }; sleep 1; \
	done; echo "ERROR: $(URL)/ask did not become ready" >&2; exit 1

## agent-down: remove the agent and restore v1 nginx
agent-down:
	@echo "==> Removing agent resources"
	@$(KUBECTL) delete -f agent.yaml --ignore-not-found
	@$(KUBECTL) delete secret llm-credentials --ignore-not-found
	@echo "==> Restoring the v1 nginx config"
	@$(KUBECTL) apply -f $(KIND_DIR)/nginx.yaml
	@$(KUBECTL) rollout restart deployment/nginx
	@$(KUBECTL) rollout status deployment/nginx --timeout=120s

## reset-entries: truncate the guestbook and seed 4 friendly entries
reset-entries:
	@echo "==> Resetting entries"
	@$(KUBECTL) exec postgres-0 -- psql -q -U guestbook -d guestbook -c "TRUNCATE entries RESTART IDENTITY; \
	  INSERT INTO entries (name, message) VALUES \
	  ('Ada', 'Loved the talk! SBOMs for everything.'), \
	  ('Grace', 'Great demo, the cluster just worked.'), \
	  ('Linus', 'Hello from the third row, thanks BSidesPDX!'), \
	  ('Margaret', 'Please bring more stickers next year.');"
	@curl -fsS $(URL)/entries >/dev/null && echo "==> Seeded 4 entries"

## scorecard: component counts for the v1 app /sbom, the v1 images and the v2 images
scorecard:
	@APP_URL=$(URL) ./sbom-stages.sh scorecard

## sbom-diff: components in the v2 images that are not in v1
sbom-diff:
	@./sbom-stages.sh diff
