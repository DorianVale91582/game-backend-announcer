#!/bin/sh
set -eu

curl --fail-with-body \
  --request POST \
  --header 'Content-Type: application/json' \
  --data '{"id":"asset-104","audience":"all_backend_members","kind":"player_asset","account_id":"studio-7","payload":{"asset_id":"asset-104","state":"ready"}}' \
  "${ANNOUNCER_URL:-http://localhost:8080}/announcements"
