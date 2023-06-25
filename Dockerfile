FROM alpine:3.18 as downloader

RUN apk --no-cache add curl
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o yt-dlp
RUN chmod a+rx yt-dlp


FROM golang:1.20.4-alpine3.16 AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

# Build binary
RUN CGO_ENABLED=0 go build -o gotube-dl


FROM python:3.11-alpine3.15

ENV LISTEN_ADDRESS="0.0.0.0"

ENV UID=934
ENV GID=934

# Add group and user
RUN addgroup -S --gid ${GID} gotube-dl && adduser -S --uid ${UID} gotube-dl -G gotube-dl

RUN apk add --no-cache ffmpeg

COPY --from=builder /app/gotube-dl /usr/bin/gotube-dl
COPY --from=downloader yt-dlp /usr/bin/yt-dlp
COPY yt-dlp.conf /etc/yt-dlp.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

USER gotube-dl

ENTRYPOINT ["/usr/bin/gotube-dl"]
