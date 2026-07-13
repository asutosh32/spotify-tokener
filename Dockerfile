FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /build

COPY go.mod ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -o spotify-tokener github.com/topi314/spotify-tokener

FROM chromedp/headless-shell

EXPOSE 8080

COPY --from=build /build/spotify-tokener /bin/spotify-tokener

ENTRYPOINT ["/bin/spotify-tokener"]
