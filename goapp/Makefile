BIN          := bin/guestbook
SBOM_DIR     := sbom
RUNTIME_SBOM := $(SBOM_DIR)/runtime.txt
PORT         ?= 18080

.PHONY: build sbom run test clean

## build: compile to ./bin
build:
	@mkdir -p bin
	go build -trimpath -o $(BIN) .

$(BIN):
	$(MAKE) build

## sbom: start the app, fetch /sbom into ./sbom/runtime.txt, stop the app
sbom: $(BIN)
	@mkdir -p $(SBOM_DIR)
	@env -u DATABASE_URL PORT=$(PORT) $(BIN) >/dev/null 2>&1 & pid=$$!; \
	trap 'kill $$pid 2>/dev/null; wait $$pid 2>/dev/null' EXIT; \
	for i in $$(seq 1 50); do \
	  curl -fsS -o /dev/null http://localhost:$(PORT)/healthz 2>/dev/null && break; \
	  kill -0 $$pid 2>/dev/null || { echo "app exited early (is port $(PORT) in use?)" >&2; exit 1; }; \
	  sleep 0.1; \
	done; \
	curl -fsS http://localhost:$(PORT)/sbom -o $(RUNTIME_SBOM) && echo "wrote $(RUNTIME_SBOM)"

## run: run the app (in-memory store unless DATABASE_URL is set)
run:
	PORT=$(PORT) go run .

test:
	go test ./...

clean:
	rm -rf bin $(SBOM_DIR)
