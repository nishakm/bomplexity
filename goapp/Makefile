BIN        := bin/guestbook
SBOM_DIR   := sbom
PORT       ?= 18080

# cyclonedx-gomod: prefer PATH, fall back to the default `go install` location.
CDXGOMOD   ?= $(or $(shell command -v cyclonedx-gomod),$(shell go env GOPATH)/bin/cyclonedx-gomod)

SRC_SBOM     := $(SBOM_DIR)/source.cdx.json
BIN_SBOM     := $(SBOM_DIR)/binary.cdx.json
RUNTIME_SBOM := $(SBOM_DIR)/runtime.cdx.json

.PHONY: build sbom-embedded sbom-runtime sbom-diff run test clean

## build: compile to ./bin and generate both CycloneDX SBOMs into ./sbom
build:
	@mkdir -p bin $(SBOM_DIR)
	go build -trimpath -o $(BIN) .
	$(CDXGOMOD) bin -json -output $(BIN_SBOM) $(BIN)
	$(CDXGOMOD) app -json -output $(SRC_SBOM) -main . .
	@echo "SBOMs written: $(BIN_SBOM) $(SRC_SBOM)"

## sbom-embedded: module list Go embedded in the binary
sbom-embedded: $(BIN)
	go version -m $(BIN)

$(BIN):
	$(MAKE) build

## sbom-runtime: start the app, fetch /sbom, stop the app
sbom-runtime: $(BIN)
	@mkdir -p $(SBOM_DIR)
	@env -u DATABASE_URL PORT=$(PORT) $(BIN) >/dev/null 2>&1 & pid=$$!; \
	trap 'kill $$pid 2>/dev/null; wait $$pid 2>/dev/null' EXIT; \
	for i in $$(seq 1 50); do \
	  curl -fsS -o /dev/null http://localhost:$(PORT)/healthz 2>/dev/null && break; \
	  kill -0 $$pid 2>/dev/null || { echo "app exited early (is port $(PORT) in use?)" >&2; exit 1; }; \
	  sleep 0.1; \
	done; \
	curl -fsS http://localhost:$(PORT)/sbom -o $(RUNTIME_SBOM) && echo "wrote $(RUNTIME_SBOM)"

## sbom-diff: compare the runtime SBOM (flat) with the source SBOM (graph)
sbom-diff:
	@for f in $(RUNTIME_SBOM) $(SRC_SBOM); do \
	  [ -f $$f ] || { echo "missing $$f: run 'make build sbom-runtime' first" >&2; exit 1; }; \
	done
	@jq -n -r -f sbom-diff.jq --slurpfile rt $(RUNTIME_SBOM) --slurpfile src $(SRC_SBOM) \
	  --arg rtfile $(RUNTIME_SBOM) --arg srcfile $(SRC_SBOM)

## run: run the app (in-memory store unless DATABASE_URL is set)
run:
	PORT=$(PORT) go run .

test:
	go test ./...

clean:
	rm -rf bin $(SBOM_DIR)
