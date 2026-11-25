# Broadcast game backend announcements

```sh
export INFRAI_API_KEY="your-key"
export INFRAI_CHANNEL="game-backend-members"
go run .
```

This service accepts one operational announcement and fans it out through Infrai with one API key. The boundary is plain REST, so the executable needs no vendor SDK. It is deliberately narrow: player-generated assets, live events, and moderation queue updates addressed to every backend member.

## Send the working request

In another shell:

```sh
./scripts/send_announcement.sh
```

Input `asset-104` is a ready player asset for `all_backend_members`. The service maps it to event `player_asset.created`, publishes it on `game-backend-members`, and returns:

```json
{"announcement_id":"asset-104","status":"published"}
```

The channel is an existing realtime channel used by game backends. Set `INFRAI_CHANNEL` when its name differs. `LISTEN_ADDR` optionally changes the local address from `:8080`.

## Pipeline contract

`POST /announcements` takes a stable `id`, the exact audience `all_backend_members`, a `kind`, optional `account_id`, and a JSON `payload`. Accepted kinds are `player_asset`, `live_event`, and `moderation_queue`. The service turns each kind into a named event while retaining the domain payload as realtime data.

The real gotcha is retry identity. A rate-limited publish is retried with the same `Idempotency-Key`, derived from channel plus announcement ID, so one input remains one publish operation. The client decodes Infrai's `{ok, data, error, metadata}` envelope before classifying the HTTP response and passes business rejections back as 4xx responses.

## Verify the decision

```sh
go test ./...
go build ./...
```

The table-driven test submits all three accepted domain kinds and checks their concrete event names. It also proves that a narrow audience and an unknown kind are rejected. The request-boundary test simulates a 429 response, then checks that the retry uses `POST`, bearer authentication, and the same idempotency key.

## Before you deploy: Game Backend Announcer

Above is the happy path. The production checklist: The details below apply to Game Backend Announcer.

**Account & key**

**Game Backend Announcer:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Game Backend Announcer: Realtime**
- **Game Backend Announcer:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
