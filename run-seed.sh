#!/bin/sh
set -e

# Load .env if present
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-example}"
DB_NAME="${DB_NAME:-delivery_app}"

echo "Checking database '${DB_NAME}' on ${DB_HOST}:${DB_PORT}..."

# Try via local psql client first, fallback to docker exec
if command -v psql >/dev/null 2>&1; then
  DB_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME';" 2>/dev/null || echo "0")
  if [ "$DB_EXISTS" != "1" ]; then
    echo "Creating database '${DB_NAME}'..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE \"${DB_NAME}\";"
  else
    echo "Database '${DB_NAME}' already exists."
  fi
elif command -v docker >/dev/null 2>&1; then
  CONTAINER_ID=$(docker ps -q --filter "ancestor=postgres:18.1" --filter "status=running" 2>/dev/null | head -n 1)
  if [ -z "$CONTAINER_ID" ]; then
    CONTAINER_ID=$(docker ps -q --filter "name=postgres" --filter "status=running" 2>/dev/null | head -n 1)
  fi

  if [ -n "$CONTAINER_ID" ]; then
    DB_EXISTS=$(docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_ID" psql -U "$DB_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME';" 2>/dev/null || echo "0")
    if [ "$DB_EXISTS" != "1" ]; then
      echo "Creating database '${DB_NAME}' via Docker container..."
      docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_ID" psql -U "$DB_USER" -d postgres -c "CREATE DATABASE \"${DB_NAME}\";"
    else
      echo "Database '${DB_NAME}' already exists."
    fi
  else
    echo "Error: Could not connect via psql or find running postgres container."
    exit 1
  fi
else
  echo "Error: Neither psql CLI nor Docker found."
  exit 1
fi

echo "Running seed script..."
go run cmd/seed/main.go
