#!/bin/bash
set -e

APP_NAME="flights"
BINARY="flights-app"
MAIN="src/main.go"

echo "→ Building..."
go build -o "$BINARY" "$MAIN"

echo "→ Restarting service..."
sudo systemctl restart "$APP_NAME"

echo "→ Done. Status:"
sudo systemctl status "$APP_NAME" --no-pager -l