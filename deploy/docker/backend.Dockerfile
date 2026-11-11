# Build context: repository root
#   docker build -f deploy/docker/backend.Dockerfile -t connect4-backend .

FROM golang:1.23-alpine AS build
WORKDIR /src

# TODO: once go.sum is committed, copy go.mod/go.sum first and run
# `go mod download` in its own layer so dependencies are cached between builds.
COPY backend/ .
RUN go mod tidy \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
# Numeric UID so Kubernetes can verify runAsNonRoot.
USER 65532:65532
ENTRYPOINT ["/server"]
