FROM --platform=linux/amd64 golang:1.21-bullseye AS builder

RUN apt-get update && apt-get install -y gcc musl-tools && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -o x-ui -ldflags="-w -s" .


FROM --platform=linux/amd64 debian:11-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/local/x-ui
COPY --from=builder /app/x-ui .
COPY bin/ ./bin/

CMD ["./x-ui"]
