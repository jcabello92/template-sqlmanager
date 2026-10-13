# --- FASE 1: BASE ---
FROM golang:1.27.2-alpine3.24 AS base
WORKDIR /app
RUN apk add --no-cache git build-base

# --- FASE 2: DESARROLLO (dev) ---
FROM base AS dev
RUN go install github.com/air-verse/air@latest
CMD ["sh", "-c", "if [ -f go.mod ]; then air; else echo 'Proyecto no inicializado. Manteniendo contenedor vivo...' && tail -f /dev/null; fi"]

# --- FASE 3: CONSTRUCCIÓN (build) ---
FROM base AS build
COPY ./volume/app/src .
RUN echo 'package main; import "fmt"; import "time"; func main() { fmt.Println("Plantilla Go"); time.Sleep(86400 * time.Second) }' > dummy.go && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/app dummy.go
RUN if [ -f go.mod ]; then go mod download && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/app .; fi

# --- FASE 4: PRODUCCIÓN (prod) ---
FROM alpine:latest AS prod
WORKDIR /app
COPY --from=build /app/app ./app
CMD ["./app"]