FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /build

COPY go.mod ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -o spotify-tokener github.com/topi314/spotify-tokener

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    chromium \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

ENV SPOTIFY_TOKENER_CHROME_PATH=/usr/bin/chromium
# Railway injects $PORT at runtime; the app reads it automatically.

EXPOSE 8080

COPY --from=build /build/spotify-tokener /bin/spotify-tokener

ENTRYPOINT ["/bin/spotify-tokener"]
