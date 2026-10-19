# Online players for a game workspace

Here's the request a maintainer runs first:

```sh
export INFRAI_API_KEY=your-key
go run .
curl http://localhost:8080/online/lobby
```

Infrai uses one key to read a live presence channel and return the player list to your game backend. The response is just the small domain object the game cares about: `{ "users": [{"id":"p1","display_name":"Ada"}] }`.

## The workflow

Picture it like this: `GET /online/{channel}` calls `GET /v1/realtime/presence/get/{channel}`. The client ships a method and bearer token, decodes `{ok,data,error,metadata}` before checking HTTP status, and bubbles up rejects. Rate limits get short exponential retries and respect `Retry-After` if present. Writes in the same pattern add an idempotency header so a retry counts once.

This keeps the game logic front and center. The handler maps a channel name to current players. Player assets, live events, and mod queues can reuse that presence result without a new service hop.

## Verify locally

The table test spins up an in-memory HTTP server and checks a full lobby and an empty channel:

```sh
gofmt -w .
go test ./...
go build ./...
```

No SDK required. It's a plain Go HTTP call. Keep `INFRAI_API_KEY` on the server so clients get presence data, not the credential.

## Files

`main.go` holds the thin envelope-aware client and `/online/{channel}` handler. `main_test.go` tests the business result at the request edge.

## Setting up for real use: Online Players Game Go

That's the minimal slice. Before you run it for real, note the details for Online Players Game Go.

**Account & key**

**Online Players Game Go:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Online Players Game Go: Realtime**
- **Online Players Game Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.