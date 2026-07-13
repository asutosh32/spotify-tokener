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

FROM chromedp/headless-shell

USER root

RUN chmod +x /headless-shell

EXPOSE 8080
# Railway injects $PORT at runtime; the app reads it automatically.

COPY --from=build /build/spotify-tokener /bin/spotify-tokener

ENTRYPOINT ["/bin/spotify-tokener"]
