#!/bin/sh
set -e

# Start the Go API server in the background (CWD=/app so goose finds ./migrations)
cd /app
./server &
SERVER_PID=$!

# Forward SIGTERM/SIGINT to the Go process before nginx exits
trap 'kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null; exit 0' TERM INT

# Run nginx in the foreground — container lives as long as nginx does
exec nginx -g "daemon off;"
