# Demo cluster: `make up` / `make smoke` / `make down`
# Assumes the app image has already been built (see ../goapp).

CLUSTER_NAME ?= sbom-demo
NAMESPACE    ?= sbom-demo
IMAGE        ?= guestbook:dev
KIND_CONFIG  ?= cluster.yaml
MANIFESTS    ?= .
ENV_FILE     ?= .env
SMOKE_URL    ?= http://localhost:8080
# Always talk to the demo cluster, whatever the current kubectl context is.
KUBECTL      := kubectl --context kind-$(CLUSTER_NAME) -n $(NAMESPACE)

.DEFAULT_GOAL := help
.PHONY: help up cluster load namespace secret deploy smoke status down undeploy

## help: list all targets
help:
	@awk '/^## [a-z-]+: /{ name=$$2; sub(/:$$/,"",name); d=$$0; sub(/^## [a-z-]+: /,"",d); printf "  %-12s %s\n", name, d }' $(MAKEFILE_LIST)

## up: bring up the whole demo from nothing (cluster, load, namespace, secret, deploy)
up:
	@echo "==> Bringing up the demo: cluster '$(CLUSTER_NAME)', namespace '$(NAMESPACE)', image $(IMAGE)"
	@$(MAKE) --no-print-directory cluster
	@$(MAKE) --no-print-directory load
	@$(MAKE) --no-print-directory namespace
	@$(MAKE) --no-print-directory secret
	@$(MAKE) --no-print-directory deploy
	@echo "==> Demo is up at $(SMOKE_URL). Check it with: make smoke"

## cluster: create the kind cluster (skipped if it already exists)
cluster:
	@if kind get clusters 2>/dev/null | grep -qx '$(CLUSTER_NAME)'; then \
	  echo "==> Cluster '$(CLUSTER_NAME)' already exists, skipping creation"; \
	else \
	  echo "==> Creating kind cluster '$(CLUSTER_NAME)' from $(KIND_CONFIG)"; \
	  kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG) --wait 120s; \
	fi
	@echo "==> Waiting for the cluster node to be Ready"
	@kubectl --context kind-$(CLUSTER_NAME) wait --for=condition=Ready node --all --timeout=120s

## load: load the already-built app image into the kind cluster
load:
	@docker image inspect $(IMAGE) >/dev/null 2>&1 || { \
	  echo "ERROR: image $(IMAGE) not found. Build it first, e.g.: docker build -t $(IMAGE) ../goapp" >&2; exit 1; }
	@echo "==> Loading $(IMAGE) into kind cluster '$(CLUSTER_NAME)'"
	@kind load docker-image $(IMAGE) --name $(CLUSTER_NAME)

## namespace: create the namespace if it does not exist
namespace:
	@echo "==> Ensuring namespace '$(NAMESPACE)' exists"
	@kubectl --context kind-$(CLUSTER_NAME) create namespace $(NAMESPACE) --dry-run=client -o yaml \
	  | kubectl --context kind-$(CLUSTER_NAME) apply -f -

## secret: create/update the db-credentials Secret from .env
secret:
	@test -f $(ENV_FILE) || { \
	  echo "ERROR: $(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)  (then set a password)" >&2; exit 1; }
	@for k in POSTGRES_PASSWORD DATABASE_URL; do \
	  grep -q "^$$k=." $(ENV_FILE) || { echo "ERROR: $(ENV_FILE) has no value for $$k (see .env.example)" >&2; exit 1; }; \
	done
	@echo "==> Applying Secret db-credentials from $(ENV_FILE)"
	@$(KUBECTL) create secret generic db-credentials --from-env-file=$(ENV_FILE) --dry-run=client -o yaml \
	  | $(KUBECTL) apply -f -

## deploy: apply Postgres, wait for it, then apply the app and nginx and wait for them
deploy:
	@echo "==> Deploying Postgres"
	@$(KUBECTL) apply -f $(MANIFESTS)/postgres.yaml
	@echo "==> Waiting for Postgres to be ready (up to 3 min)"
	@$(KUBECTL) rollout status statefulset/postgres --timeout=180s
	@# The pod's probe checks the Unix socket, which is up during first-time init
	@# before Postgres accepts TCP connections (which is what the app uses).
	@echo "==> Waiting for Postgres to accept TCP connections"
	@$(KUBECTL) exec postgres-0 -- sh -c 'for i in $$(seq 1 60); do pg_isready -q -h 127.0.0.1 -U guestbook && exit 0; sleep 1; done; echo "Postgres never accepted TCP connections" >&2; exit 1'
	@echo "==> Deploying the app and nginx"
	@$(KUBECTL) apply -f $(MANIFESTS)/app.yaml -f $(MANIFESTS)/nginx.yaml
	@$(KUBECTL) set image deployment/guestbook guestbook=$(IMAGE)
	@echo "==> Waiting for the app to roll out (up to 2 min)"
	@$(KUBECTL) rollout status deployment/guestbook --timeout=120s
	@echo "==> Waiting for nginx to roll out (up to 2 min)"
	@$(KUBECTL) rollout status deployment/nginx --timeout=120s
	@echo "==> Waiting for the app to answer through nginx at $(SMOKE_URL)"
	@for i in $$(seq 1 60); do \
	  curl -fsS -o /dev/null $(SMOKE_URL)/healthz 2>/dev/null && exit 0; sleep 1; \
	done; echo "ERROR: no answer from $(SMOKE_URL)/healthz after 60s (is host port 8080 free?)" >&2; exit 1

## smoke: post an entry through nginx, read it back, fail loudly on any problem
smoke:
	@echo "==> Smoke test through $(SMOKE_URL)"
	@msg="smoke-$$(date +%s)"; \
	echo "    POST $(SMOKE_URL)/entries"; \
	curl -fsS --retry 5 --retry-connrefused --retry-delay 2 -X POST -H 'Content-Type: application/json' \
	  -d "{\"name\":\"smoke\",\"message\":\"$$msg\"}" $(SMOKE_URL)/entries >/dev/null \
	  || { echo "SMOKE FAILED: could not POST an entry" >&2; exit 1; }; \
	echo "    GET  $(SMOKE_URL)/entries"; \
	curl -fsS --retry 5 --retry-connrefused --retry-delay 2 $(SMOKE_URL)/entries | grep -q "$$msg" \
	  || { echo "SMOKE FAILED: the posted entry was not read back" >&2; exit 1; }; \
	echo "==> Smoke test PASSED (entry $$msg round-tripped)"

## status: show pods, services and PVCs in the namespace
status:
	@echo "==> Pods, services and PVCs in namespace '$(NAMESPACE)'"
	@$(KUBECTL) get pods,svc,pvc

## down: delete the kind cluster entirely (fine if it is already gone)
down:
	@echo "==> Deleting kind cluster '$(CLUSTER_NAME)'"
	@kind delete cluster --name $(CLUSTER_NAME)

## undeploy: delete the namespace but keep the cluster (fast reset)
undeploy:
	@if kind get clusters 2>/dev/null | grep -qx '$(CLUSTER_NAME)'; then \
	  echo "==> Deleting namespace '$(NAMESPACE)' (cluster '$(CLUSTER_NAME)' is kept)"; \
	  kubectl --context kind-$(CLUSTER_NAME) delete namespace $(NAMESPACE) --ignore-not-found --timeout=120s; \
	else \
	  echo "==> Cluster '$(CLUSTER_NAME)' does not exist, nothing to undeploy"; \
	fi
