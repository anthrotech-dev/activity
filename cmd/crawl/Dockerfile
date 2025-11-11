FROM golang:latest AS builder
WORKDIR /work

ARG VERSION

RUN --mount=type=bind,source=./go.mod,target=/work/go.mod,ro \
    --mount=type=bind,source=./go.sum,target=/work/go.sum,ro \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

RUN --mount=type=bind,source=.,target=/work \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    VERSION=${VERSION:-$(git describe 2>/dev/null || echo "unset")} \
    BUILD_MACHINE="$(uname -srmo)" \
    BUILD_TIME="$(date)" \
    GO_VERSION="$(go version)" \
    go build -o /crawl ./cmd/crawl

FROM gcr.io/distroless/base:nonroot
ENV HOME=/home/nonroot
USER nonroot:nonroot

COPY --from=builder /crawl /crawl

CMD ["/crawl"]
