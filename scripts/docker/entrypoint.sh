#!/bin/bash
set -e

echo "Waiting for PostgreSQL..."
until PGPASSWORD="${POSTGRES_PASSWORD:-video_sync}" psql \
    -h "${DB_HOST:-postgres}" \
    -p "${DB_PORT:-5432}" \
    -U "${POSTGRES_USER:-video_sync}" \
    -d "${POSTGRES_DB:-video_sync}" \
    -c '\q' 2>/dev/null; do
    echo "PostgreSQL is not ready, retrying..."
    sleep 2
done
echo "PostgreSQL is ready"

echo "Checking database initialization..."
TABLE_COUNT=$(
    PGPASSWORD="${POSTGRES_PASSWORD:-video_sync}" psql \
        -h "${DB_HOST:-postgres}" \
        -p "${DB_PORT:-5432}" \
        -U "${POSTGRES_USER:-video_sync}" \
        -d "${POSTGRES_DB:-video_sync}" \
        -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';"
)

if [ "$TABLE_COUNT" -eq "0" ]; then
    echo "Initializing database schema..."
    PGPASSWORD="${POSTGRES_PASSWORD:-video_sync}" psql \
        -h "${DB_HOST:-postgres}" \
        -p "${DB_PORT:-5432}" \
        -U "${POSTGRES_USER:-video_sync}" \
        -d "${POSTGRES_DB:-video_sync}" \
        -f /app/bili-sync-schema.sql || true
    echo "Database initialization complete"
else
    echo "Database schema already exists"
fi

chmod 755 /downloads /metadata /var/log/video-sync

echo "Starting video-sync..."
exec /app/video-sync -config /app/configs/config.yaml
