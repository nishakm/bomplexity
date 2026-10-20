# guestbook: a Go app and its SBOMs

A tiny guestbook HTTP API, built to demonstrate three ways of producing a
Software Bill of Materials (SBOM) using only Go's own tooling.

## API

| Method | Path       | Description                                                |
|--------|------------|------------------------------------------------------------|
| POST   | `/entries` | Add an entry: `{"name": "...", "message": "..."}`          |
| GET    | `/entries` | List entries                                               |
| GET    | `/healthz` | Returns 200                                                |
| GET    | `/sbom`    | The app's own CycloneDX SBOM, built at runtime             |

Storage is Postgres (via `pgx`) when `DATABASE_URL` is set, otherwise an
in-memory store. `PORT` sets the listen port (default `8080`).

Dependencies are deliberate, because each one appears in the SBOM: only
`github.com/jackc/pgx/v5` (plus its transitive modules). Routing uses the
standard library's `http.ServeMux`.

## Prerequisites

| Tool              | Install                                                                          |
|-------------------|----------------------------------------------------------------------------------|
| Go 1.25+          | <https://go.dev/dl/> (an older Go will auto-download the toolchain in `go.mod`)  |
| cyclonedx-gomod   | `go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest` (or `brew install cyclonedx/cyclonedx/cyclonedx-gomod`) |
| jq                | `sudo apt install jq` / `brew install jq`                                        |
| make, curl, git   | usually preinstalled                                                             |

`cyclonedx-gomod` installs into `$(go env GOPATH)/bin`; the Makefile finds it
there even if that directory is not on your `PATH`. Its `app` mode needs this
folder to be a git repository (it derives the main module's version from git).

## Make targets

| Target          | What it does                                                                  |
|-----------------|-------------------------------------------------------------------------------|
| `build`         | `go build` into `./bin/`, then writes both CycloneDX SBOMs into `./sbom/`     |
| `sbom-embedded` | prints `go version -m ./bin/guestbook`                                        |
| `sbom-runtime`  | starts the app, fetches `/sbom` into `sbom/runtime.cdx.json`, stops the app   |
| `sbom-diff`     | summarises runtime vs source SBOM (counts, dependency graph present or not)   |
| `run`           | runs the app (`PORT` defaults to 18080 in the Makefile)                       |
| `test`          | `go test ./...`                                                               |
| `clean`         | removes `bin/` and `sbom/`                                                    |

## The SBOMs

1. **Embedded, `go version -m ./bin/guestbook`.** Not CycloneDX: this is the
   raw module list the Go linker stamped into the binary (`dep` lines with
   `h1:` checksums, plus VCS and build settings). It is the source of truth the
   next two are derived from.
2. **Binary, `sbom/binary.cdx.json`**, from `cyclonedx-gomod bin`. It reads the
   same embedded build info and reformats it as CycloneDX. Flat component
   list, no relationships.
3. **Source, `sbom/source.cdx.json`**, from `cyclonedx-gomod app`. It loads the
   module with the Go toolchain, evaluates build constraints, and records the
   `dependencies` graph: which module requires which.
4. **Runtime, `sbom/runtime.cdx.json`**, from `GET /sbom`. The app calls
   `runtime/debug.ReadBuildInfo()` on itself and emits CycloneDX: main module,
   each dependency with version and SHA-256 (the `h1:` go.sum value decoded to
   hex, original kept in a `go:sum` property), and the Go toolchain as a
   `platform` component.

### Why they differ

- **Graph vs. flat list.** Build info records *which modules are in the binary*
  but not *who pulled them in*. The binary and runtime SBOMs therefore have no
  `dependencies` section. Only the source-based SBOM can see `go.mod` and the
  package import graph.
- **Toolchain.** The runtime SBOM lists the Go toolchain as a component;
  `cyclonedx-gomod` only adds the standard library when passed `-std`.
- **Version of the main module.** Build info and the tools derive it from VCS
  stamps; it will be a pseudo-version such as `v0.0.0-<time>-<rev>`, with
  `+dirty` if the tree has uncommitted changes.
- **Provenance.** Source SBOMs describe what *would* be built; binary and
  runtime SBOMs describe what *was* built. For attestation, prefer the latter
  and keep the former for dependency analysis.
