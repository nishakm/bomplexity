BIN          := bin/guestbook
SBOM_DIR     := sbom
RUNTIME_SBOM := $(SBOM_DIR)/runtime.txt
PORT         ?= 18080
IMAGE        ?= guestbook
TAG          ?= dev
PLATFORM     ?= linux/$(shell go env GOARCH)
IMAGE_SBOM   := $(SBOM_DIR)/image.spdx.json

.PHONY: build sbom image image-base-sboms run test clean

## build: compile to ./bin
build:
	@mkdir -p bin
	go build -trimpath -o $(BIN) .

$(BIN):
	$(MAKE) build

## sbom: start the app, fetch /sbom, write its purls (one per line) to ./sbom/runtime.txt, stop the app
sbom: $(BIN)
	@mkdir -p $(SBOM_DIR)
	@env -u DATABASE_URL PORT=$(PORT) $(BIN) >/dev/null 2>&1 & pid=$$!; \
	trap 'kill $$pid 2>/dev/null; wait $$pid 2>/dev/null' EXIT; \
	for i in $$(seq 1 50); do \
	  curl -fsS -o /dev/null http://localhost:$(PORT)/healthz 2>/dev/null && break; \
	  kill -0 $$pid 2>/dev/null || { echo "app exited early (is port $(PORT) in use?)" >&2; exit 1; }; \
	  sleep 0.1; \
	done; \
	curl -fsS http://localhost:$(PORT)/sbom -o $(RUNTIME_SBOM).tmp; \
	status=$$?; \
	if [ $$status -eq 0 ]; then \
	  tail -n +2 $(RUNTIME_SBOM).tmp | cut -c5- > $(RUNTIME_SBOM) && echo "wrote $(RUNTIME_SBOM)"; \
	fi; \
	rm -f $(RUNTIME_SBOM).tmp; exit $$status

## image: build the container image and write its SBOM to ./sbom/image.spdx.json
image:
	@mkdir -p $(SBOM_DIR)
	@rm -rf $(SBOM_DIR)/.image-export
	docker buildx build --sbom=true --provenance=false -t $(IMAGE):$(TAG) \
	  --output type=docker --output type=local,dest=$(SBOM_DIR)/.image-export .
	@# BuildKit wraps the SPDX document in an in-toto statement; keep the document.
	@jq '.predicate' $(SBOM_DIR)/.image-export/sbom.spdx.json > $(IMAGE_SBOM)
	@rm -rf $(SBOM_DIR)/.image-export
	@jq -r '"wrote $(IMAGE_SBOM): \(.spdxVersion), \(.packages | length) packages"' $(IMAGE_SBOM)

## image-base-sboms: fetch the SBOMs published for each FROM image in the Dockerfile
image-base-sboms:
	@mkdir -p $(SBOM_DIR)
	@for img in $$(awk 'toupper($$1)=="FROM" {print $$2}' Dockerfile | sort -u); do \
	  out=$(SBOM_DIR)/base-$$(echo $${img%%@*} | tr '/:' '__').spdx.json; \
	  docker buildx imagetools inspect $$img --format '{{json .SBOM}}' \
	    | jq '.["$(PLATFORM)"].SPDX' > $$out; \
	  jq -e . $$out >/dev/null && [ "$$(jq 'type' $$out)" = '"object"' ] \
	    || { echo "no SBOM published for $$img ($(PLATFORM))" >&2; rm -f $$out; exit 1; }; \
	  jq -r --arg img "$${img%%@*}" --arg f "$$out" '"wrote \($$f): \($$img), \(.spdxVersion), \(.packages | length) packages"' $$out; \
	done

## run: run the app (in-memory store unless DATABASE_URL is set)
run:
	PORT=$(PORT) go run .

test:
	go test ./...

clean:
	rm -rf bin $(SBOM_DIR)
