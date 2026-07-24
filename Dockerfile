FROM golang:1.25 AS builder


WORKDIR /app


COPY go.mod go.sum ./

RUN go mod download


COPY . .


RUN CGO_ENABLED=1 go build \
    -o sticker-bot \
    ./cmd/bot



FROM debian:bookworm-slim


RUN apt update && \
    apt install -y \
    ffmpeg \
    ca-certificates \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*


WORKDIR /app


COPY --from=builder /app/sticker-bot .


RUN mkdir -p storage/sessions


CMD ["./sticker-bot"]
