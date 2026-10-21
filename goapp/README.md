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
| Docker + buildx | only for the container image (`make image`); needs the containerd image store, the default in recent Docker |
| jq              | only for the container image targets: `sudo apt install jq` / `brew install jq` |

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
| `image`         | builds the container image and writes its SBOM to `sbom/image.spdx.json` |
| `image-base-sboms` | fetches the SBOMs published for the `FROM` images into `sbom/base-*.spdx.json` |
| `run`           | runs the app (`PORT` defaults to 18080 in the Makefile)               |
| `test`          | `go test ./...`                                                       |
| `clean`         | removes `bin/` and `sbom/`                                            |

## Container image

The `Dockerfile` is a multi-stage build: the first stage compiles a static
binary (`CGO_ENABLED=0`) on `golang`, and the release stage copies only that
binary onto `alpine`, running as a non-root user (uid 10001). Both base images
are pinned by digest, and both are Docker Official Images that publish an SPDX
SBOM attestation, so you can inspect what you are building on.

```
make image                # build guestbook:dev and write sbom/image.spdx.json
make image-base-sboms     # write sbom/base-golang_*.spdx.json and sbom/base-alpine_*.spdx.json
docker run --rm -p 8080:8080 guestbook:dev
```

Override the name or tag with `IMAGE=... TAG=...`.

- **Image SBOM.** `make image` builds with `--sbom=true`, so BuildKit scans the
  final image (BuildKit's scanner is Syft-based) and the make target saves the
  SPDX 2.3 document. It lists the Alpine packages (`musl`, `busybox`, ...) and
  the Go modules and standard library found in the binary (pgx, `stdlib`).
  Only the release stage is scanned: the `golang` build stage and its
  toolchain are not in the image, so they are not in its SBOM.
- **Base image SBOMs.** `make image-base-sboms` reads the attestations from the
  registry (`docker buildx imagetools inspect <image> --format '{{json .SBOM}}'`).
  It fails if a `FROM` image has none. Not every minimal image publishes one:
  `scratch` has none, and `gcr.io/distroless/static` and
  `cgr.dev/chainguard/static` returned no SBOM through this mechanism when I
  checked, which is why the release stage is Alpine.
- **Updating bases.** To move to newer base images, change the tags and
  re-resolve the digests with `docker buildx imagetools inspect <image>`.

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
