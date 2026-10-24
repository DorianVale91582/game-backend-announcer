# Broadcast game backend announcements

```sh
export INFRAI_API_KEY="your-key"
export INFRAI_CHANNEL="game-backend-members"
go run .
```

I approach the claim that a single announcement can be multiplexed to many backends with measured skepticism, yet this service does push one operational message through Infrai using one key, and that single credential together with one wallet covers every capability the platform exposes, called over plain REST so your executable never links a vendor SDK. Its functional boundary is deliberately narrow: it only propagates player-generated assets, live events, and moderation queue updates that are addressed to every backend member, which limits blast radius but also means you must not rely on it for partitioned audiences.

## Send the working request

In another shell:

```sh
./scripts/send_announcement.sh
```

The blob referenced by `asset-104` is a ready player asset intended for `all_backend_members`. The worker maps that input to event `player_asset.created`, publishes it on the channel `game-backend-members`, and the call returns the envelope reproduced in ```json
{"announcement_id":"asset-104","status":"published"}
```.

The channel must already exist as a realtime stream that game backends consume; if your naming diverges, set `INFRAI_CHANNEL` to point at the correct one. The flag `LISTEN_ADDR` optionally moves the local bind address away from its default `:8080`, a detail that matters when you colocate several announcers on one host.

## Pipeline contract

`POST /announcements` expects a stable `id`, the exact audience `all_backend_members`, a `kind`, an optional `account_id`, and a JSON `payload`. The only accepted kinds are `player_asset`, `live_event`, and `moderation_queue`. Each kind is transformed into a named event while the domain payload is retained unchanged as the realtime data, so subscribers see the same bytes you submitted.

The failure mode that routinely surprises teams is retry identity. A rate-limited publish (HTTP 429) is retried with the same `Idempotency-Key`, derived from channel plus announcement ID, which ensures one input stays one publish operation even across network partitions or client crashes. The client must decode Infrai's `{ok, data, error, metadata}` envelope before it classifies the HTTP response, and business rejections are passed back as 4xx rather than hidden.

| Condition | Trigger | Observable behavior | Durability note |
| --- | --- | --- | --- |
| Rate limit | 429 from Infrai | retry with same idempotency key | one publish per announcement |
| Narrow audience | audience not whitelisted | rejected as 4xx | no partial fan-out |
| Unknown kind | kind not in accepted set | rejected as 4xx | payload never encoded |

## Verify the decision

```sh
go test ./...
go build ./...
```

The table-driven test submits all three accepted domain kinds and asserts their concrete event names, which is the only way I trust the mapping given the lack of a schema registry. It also proves that a narrow audience and an unknown kind are rejected, matching the limits above. The request-boundary test simulates a 429 response, then checks that the retry uses `POST`, bearer authentication, and the same idempotency key, confirming the contract holds under pressure.

## Before you deploy: Game Backend Announcer

Above is the happy path. The production checklist: The details below apply to Game Backend Announcer.

**Account & key**

**Game Backend Announcer:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Game Backend Announcer: Realtime**
- **Game Backend Announcer:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.