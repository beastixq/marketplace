FROM golang:1.25.0-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/api/ ./cmd/api/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -buildvcs=false -o /out/marketplace-api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app && chown 10001:10001 /app
ARG TEST_COMMIT_SHA=unknown
ARG API_SOURCE_SHA256=unknown
ARG API_DIRTY=unknown
LABEL org.opencontainers.image.revision="${TEST_COMMIT_SHA}" \
    io.marketplace.source.sha256="${API_SOURCE_SHA256}" \
    io.marketplace.source.dirty="${API_DIRTY}"
COPY --from=build /out/marketplace-api /usr/local/bin/marketplace-api
WORKDIR /app
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/marketplace-api"]
CMD ["-config", "/etc/marketplace/config.yaml"]
