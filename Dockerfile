# syntax=docker/dockerfile:1.7

# Stage builder: compila o servidor a partir das dependências declaradas em
# go.mod.
FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath

WORKDIR /src

# Cache de módulos: copia só os manifestos primeiro para aproveitar layers
# entre builds quando só o código fonte muda.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copia o restante do código.
COPY . .

# Compila o servidor em /out.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /out/server ./cmd/server

# Stage runtime: imagem mínima estática para rodar o servidor em produção.
# As migrations são aplicadas pelo próprio servidor no boot quando
# PRODUCTION=true (ver cmd/server/main.go).
FROM gcr.io/distroless/static-debian12 AS runtime

WORKDIR /app

# O servidor lê os templates em runtime via os.DirFS("web/views"), então
# precisamos preservar o caminho relativo dentro do container.
COPY --from=builder /out/server /app/server
COPY web/views /app/web/views

EXPOSE 4000

USER nonroot:nonroot

ENTRYPOINT ["/app/server"]
