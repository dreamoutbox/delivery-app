#!/bin/sh
set -e

# Drops the existing database (terminating active connections) and runs dev-db-seed.sh to recreate and re-seed it.

# Load .env if present
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-example}"
DB_NAME="${DB_NAME:-delivery_app}"

echo "Resetting database '${DB_NAME}' on ${DB_HOST}:${DB_PORT}..."

# Drop existing database with force to terminate any open connections (e.g. running dev server)
if command -v psql >/dev/null 2>&1; then
  echo "Dropping database '${DB_NAME}'..."
  PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"${DB_NAME}\" WITH (FORCE);"
elif command -v docker >/dev/null 2>&1; then
  CONTAINER_ID=$(docker ps -q --filter "ancestor=postgres:18.1" --filter "status=running" 2>/dev/null | head -n 1)
  if [ -z "$CONTAINER_ID" ]; then
    CONTAINER_ID=$(docker ps -q --filter "name=postgres" --filter "status=running" 2>/dev/null | head -n 1)
  fi

  if [ -n "$CONTAINER_ID" ]; then
    echo "Dropping database '${DB_NAME}' via Docker container..."
    docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_ID" psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"${DB_NAME}\" WITH (FORCE);"
  else
    echo "Error: Could not connect via psql or find running postgres container."
    exit 1
  fi
else
  echo "Error: Neither psql CLI nor Docker found."
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec "${SCRIPT_DIR}/dev-db-seed.sh"
