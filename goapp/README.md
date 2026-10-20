# guestbook: a Go app that reports its own SBOM

A tiny guestbook HTTP API, built to demonstrate how a Go program can produce a
Software Bill of Materials (SBOM) from the build information Go embeds in every
binary.

## API

| Method | Path       | Description                                                |
|--------|------------|------------------------------------------------------------|
| POST   | `/entries` | Add an entry: `{"name": "...", "message": "..."}`          |
| GET    | `/entries` | List entries                                               |
| GET    | `/healthz` | Returns 200                                                |
| GET    | `/sbom`    | The app's own SBOM as a plain-text tree of package URLs    |

Storage is Postgres (via `pgx`) when `DATABASE_URL` is set, otherwise an
in-memory store. `PORT` sets the listen port (default `8080`).

Dependencies are deliberate, because each one appears in the SBOM: only
`github.com/jackc/pgx/v5` (plus its transitive modules). Routing uses the
standard library's `http.ServeMux`.

## Prerequisites

| Tool            | Install                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| Go 1.25+        | <https://go.dev/dl/> (an older Go will auto-download the toolchain in `go.mod`) |
| make, curl      | usually preinstalled                                                            |

## Build and run

From this folder:

```
make build                # compile to bin/guestbook
make run                  # go run, in-memory store, port 18080
PORT=8080 make run        # pick a different port
./bin/guestbook           # run the built binary (port 8080 unless PORT is set)
```

To use Postgres instead of the in-memory store (the `entries` table is created
on startup):

```
DATABASE_URL=postgres://user:pass@localhost:5432/dbname ./bin/guestbook
```

Try it (use the port you started on; `make run` uses 18080):

```
curl -XPOST localhost:18080/entries -d '{"name":"ann","message":"hi"}'
curl localhost:18080/entries
curl localhost:18080/healthz
curl localhost:18080/sbom
```

## Make targets

| Target          | What it does                                                          |
|-----------------|-----------------------------------------------------------------------|
| `build`         | `go build` into `./bin/`                                              |
| `sbom`          | starts the app, fetches `/sbom` into `sbom/runtime.txt`, stops the app |
| `run`           | runs the app (`PORT` defaults to 18080 in the Makefile)               |
| `test`          | `go test ./...`                                                       |
| `clean`         | removes `bin/` and `sbom/`                                            |

## The SBOM

`GET /sbom` calls `runtime/debug.ReadBuildInfo()` on the running binary and
prints an ASCII tree of package URLs (purls): the main module, each dependency
module with its version, and the Go toolchain (`pkg:golang/std@<version>`):

```
pkg:golang/github.com/nisha/guestbook
+-- pkg:golang/github.com/jackc/pgpassfile@v1.0.0
+-- pkg:golang/github.com/jackc/pgservicefile@v0.0.0-20240606120523-5a60cdf6a761
+-- pkg:golang/github.com/jackc/pgx/v5@v5.11.0
+-- pkg:golang/github.com/jackc/puddle/v2@v2.2.2
+-- pkg:golang/golang.org/x/sync@v0.17.0
+-- pkg:golang/golang.org/x/text@v0.29.0
`-- pkg:golang/std@1.26.10
```

Things to know:

- **Flat, not a real tree.** Build info records which modules are in the binary
  but not which module imports which, so everything hangs directly off the main
  module. (In reality `puddle` and `x/text` each pull in `x/sync`.)
- **No main module version.** Build info reports `(devel)` for the main module,
  so its purl has no `@version`.
