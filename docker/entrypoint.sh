#!/bin/sh
set -e

# Default to nobody:users (99:100) — Unraid standard
PUID=${PUID:-99}
PGID=${PGID:-100}

echo "SubVault: Starting with PUID=$PUID PGID=$PGID"

# Create group if it doesn't exist
if ! getent group "$PGID" > /dev/null 2>&1; then
    addgroup --gid "$PGID" subvault
fi

# Create user if it doesn't exist
if ! getent passwd "$PUID" > /dev/null 2>&1; then
    GROUP_NAME=$(getent group "$PGID" | cut -d: -f1)
    adduser --uid "$PUID" --ingroup "$GROUP_NAME" --disabled-password --no-create-home --gecos "" subvault
fi

# Fix ownership of data directory
chown -R "$PUID:$PGID" /app/data

# Drop privileges and run the application
exec gosu "$PUID:$PGID" ./subvault
