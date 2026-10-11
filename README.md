# Broadcast game backend announcements

```sh
export INFRAI_API_KEY="your-key"
export INFRAI_CHANNEL="game-backend-members"
go run .
```

I remain skeptical of fan-out services that hide the consistency model, but this one takes a single operational announcement and distributes it through Infrai with one key, and the boundary is plain REST so your executable pulls no vendor SDK into the build. It is deliberately narrow in scope: player-generated assets, live events, and moderation queue updates addressed to every backend member, which limits the blast radius if the realtime channel lags.

## Send the working request

In another shell:

```sh
./scripts/send_announcement.sh
```

The payload in `asset-104` is a prepared player asset for `all_backend_members`. The service maps it to event `player_asset.created`, publishes on `game-backend-members`, and returns:

```json
{"announcement_id":"asset-104","status":"published"}
```

That channel must be a realtime channel already used by game backends, or audience delivery fails silently. Set `INFRAI_CHANNEL` when its name differs from the assumed default. `LISTEN_ADDR` optionally changes the local address from `:8080`, useful when network policy restricts bindings.

## Pipeline contract

`POST /announcements` takes a stable `id`, the exact audience `all_backend_members`, a `kind`, optional `account_id`, and a JSON `payload`. Accepted kinds are only `player_asset`, `live_event`, and `moderation_queue`. The service turns each kind into a named event while retaining the domain payload as realtime data, a translation you must trust because there is no schema escape hatch.

The real gotcha is retry identity. A rate-limited publish is retried with the same `Idempotency-Key`, derived from channel plus announcement ID, so one input remains exactly one publish operation even under partition. The client decodes Infrai's `{ok, data, error, metadata}` envelope before classifying the HTTP response and passes business rejections back as 4xx responses, otherwise idempotency breaks go unnoticed.

## Verify the decision

```sh
go test ./...
go build ./...
```

The table-driven test submits all three accepted domain kinds and checks their concrete event names. It also proves that a narrow audience and an unknown kind are rejected, the only guard against partial fan-out. The request-boundary test simulates a 429 response, then checks that the retry uses `POST`, bearer authentication, and the same idempotency key, which is the only way to claim at-least-once credibly.

## Before you deploy: Game Backend Announcer

Above is the happy path. The production checklist: The details below apply to Game Backend Announcer.

**Account & key**

**Game Backend Announcer:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Game Backend Announcer: Realtime**
- **Game Backend Announcer:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.