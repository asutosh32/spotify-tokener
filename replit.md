# spotify-tokener

A Go service that fetches Spotify anonymous and account tokens using a headless Chromium browser, bypassing Spotify's TOTP requirement.

## How to run on Replit

The **"Start application"** workflow runs the service:

```
SPOTIFY_TOKENER_ADDR=0.0.0.0:5000
SPOTIFY_TOKENER_CHROME_PATH=/nix/store/.../bin/chromium
go run -mod=mod .
```

The token endpoint is at `/api/token`. The first cold-start request may fail while Chromium warms up — subsequent requests return cached tokens.

## Snapdeploy deployment

The `snapdeploy-upload.zip` contains:
- `app` — startup script that chmod's `/headless-shell` if present, finds the Chrome binary, then launches `./spotify-tokener`
- `spotify-tokener` — pre-built Linux amd64 binary

To redeploy after changes to the Go source, rebuild the binary:
```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o spotify-tokener .
zip -X snapdeploy-upload.zip app spotify-tokener
```
Then push to GitHub and trigger a redeploy on Snapdeploy.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SPOTIFY_TOKENER_ADDR` | `0.0.0.0:8080` | Listen address |
| `SPOTIFY_TOKENER_CHROME_PATH` | auto-detected | Path to Chrome/Chromium executable |
| `SPOTIFY_TOKENER_LOG_LEVEL` | `INFO` | Log level |

## User preferences

- Keep the project's existing Go structure and Dockerfile as-is.
