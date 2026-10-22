# syntax=docker/dockerfile:1

# The agent: POST /ask -> model + MCP tools. Same pinned base images as
# ../goapp/Dockerfile (both publish SBOM attestations).
FROM golang:1.26.10@sha256:5c3f3f28a273c9b61698b2d34f18375929790ab1f1d3835d032d21e2b31eae0c AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/agent .

FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0
RUN adduser -D -H -u 10001 guestbook
COPY --from=build /out/agent /usr/local/bin/agent
USER 10001
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/agent"]
