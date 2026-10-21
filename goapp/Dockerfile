# syntax=docker/dockerfile:1

# Build stage: compile a static binary. Base images are pinned by digest and
# both publish SBOM attestations (see `make image-base-sboms`).
FROM golang:1.26.10@sha256:5c3f3f28a273c9b61698b2d34f18375929790ab1f1d3835d032d21e2b31eae0c AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/guestbook .

# Release stage: minimal Alpine, non-root, only the binary.
FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0
RUN adduser -D -H -u 10001 guestbook
COPY --from=build /out/guestbook /usr/local/bin/guestbook
USER guestbook
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/guestbook"]
