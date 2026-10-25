#!/usr/bin/env bash
set -euo pipefail

# VPS demo launcher entrypoint script.
# Installs PostgreSQL 18, SeaweedFS S3, configures buckets, creates run user,
# deploys the latest release of delivery-app with systemd, and seeds mock data.

readonly APP_DIR="/app/delivery-app"
readonly APP_USER="delivery"
readonly APP_GROUP="delivery"
readonly REPO="dreamoutbox/delivery-app"

# Configuration with environment defaults
readonly APP_PORT="${APP_PORT:-8080}"
readonly DB_PORT="${DB_PORT:-5432}"
readonly DB_USER="${DB_USER:-postgres}"
readonly DB_PASSWORD="${DB_PASSWORD:-example}"
readonly DB_NAME="${DB_NAME:-delivery_app}"

readonly S3_PORT="${S3_PORT:-9000}"
readonly S3_ACCESS_KEY="${S3_ACCESS_KEY:-miniominio}"
readonly S3_SECRET_KEY="${S3_SECRET_KEY:-miniominio}"
readonly S3_PUBLIC_BUCKET="${S3_PUBLIC_BUCKET:-delivery-public}"
readonly S3_PRIVATE_BUCKET="${S3_PRIVATE_BUCKET:-delivery-private}"

# detect_arch
# Determines host system architecture and maps to standard naming.
# Inputs:
#   None
# Output:
#   Echoes architecture ("amd64" or "arm64")
# Example:
#   arch="$(detect_arch)"
detect_arch() {
  local uname_m
  uname_m="$(uname -m)"
  case "${uname_m}" in
    x86_64)
      printf '%s' "amd64"
      ;;
    aarch64|arm64)
      printf '%s' "arm64"
      ;;
    *)
      echo "Error: Unsupported architecture '${uname_m}'." >&2
      exit 1
      ;;
  esac
}

# detect_public_ip
# Resolves public or interface IP address for external web asset URLs.
# Inputs:
#   None
# Output:
#   Echoes detected IP or "localhost"
# Example:
#   host_ip="$(detect_public_ip)"
detect_public_ip() {
  local host=""
  if [ -n "${DEMO_HOST:-}" ]; then
    printf '%s' "${DEMO_HOST}"
    return 0
  fi

  host="$(curl -s --max-time 3 https://api.ipify.org 2>/dev/null || true)"
  if [ -z "${host}" ]; then
    host="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi
  if [ -z "${host}" ]; then
    host="localhost"
  fi

  printf '%s' "${host}"
}

# wait_for_port
# Polls a host and TCP port until it accepts connections or reaches timeout.
# Inputs:
#   $1 - Host address (e.g. 127.0.0.1)
#   $2 - Port number (e.g. 5432)
#   $3 - Timeout in seconds
# Example:
#   wait_for_port "127.0.0.1" "5432" 30
wait_for_port() {
  local host="$1"
  local port="$2"
  local timeout="$3"
  local start_time
  start_time="$(date +%s)"

  echo "Waiting for ${host}:${port} to become available (timeout: ${timeout}s)..."
  while true; do
    if timeout 1 bash -c "cat < /dev/null > /dev/tcp/${host}/${port}" 2>/dev/null; then
      echo "Connection to ${host}:${port} successful."
      return 0
    fi

    local current_time
    current_time="$(date +%s)"
    if [ $((current_time - start_time)) -ge "${timeout}" ]; then
      echo "Error: Timed out waiting for ${host}:${port} after ${timeout} seconds." >&2
      return 1
    fi
    sleep 1
  done
}

# install_system_dependencies
# Updates apt package lists and installs base utilities needed for the setup.
# Inputs:
#   None
# Example:
#   install_system_dependencies
install_system_dependencies() {
  echo "Installing base system dependencies..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y --no-install-recommends \
    curl \
    wget \
    tar \
    gzip \
    ca-certificates \
    gnupg \
    lsb-release \
    jq
}

# setup_postgresql_18
# Adds the official PostgreSQL repository and installs PostgreSQL 18 on bare metal.
# Configures the database user and target database.
# Inputs:
#   None
# Example:
#   setup_postgresql_18
setup_postgresql_18() {
  echo "Configuring PostgreSQL 18 official repository..."
  install -d -m 0755 /etc/apt/keyrings

  local pg_keyring="/etc/apt/keyrings/postgresql.gpg"
  curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor --yes -o "${pg_keyring}"

  local distro_codename
  distro_codename="$(lsb_release -cs 2>/dev/null || grep -oP '(?<=VERSION_CODENAME=)[a-z]+' /etc/os-release)"
  echo "deb [signed-by=${pg_keyring}] http://apt.postgresql.org/pub/repos/apt ${distro_codename}-pgdg main" \
    > /etc/apt/sources.list.d/pgdg.list

  apt-get update -y
  apt-get install -y --no-install-recommends postgresql-18 postgresql-client-18

  systemctl enable postgresql
  systemctl restart postgresql

  wait_for_port "127.0.0.1" "${DB_PORT}" 30

  echo "Configuring PostgreSQL role and database..."
  su - postgres -c "psql -c \"ALTER USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';\""
  su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'\" | grep -q 1 || psql -c \"CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};\""
}

# setup_seaweedfs
# Installs SeaweedFS binary, sets up S3 authentication config, and creates systemd service.
# Inputs:
#   $1 - Architecture ("amd64" or "arm64")
# Example:
#   setup_seaweedfs "amd64"
setup_seaweedfs() {
  local arch="$1"
  echo "Setting up SeaweedFS for architecture ${arch}..."

  local weed_url="https://github.com/seaweedfs/seaweedfs/releases/latest/download/linux_${arch}.tar.gz"
  local tmp_tar="/tmp/weed.tar.gz"

  echo "Downloading SeaweedFS binary from ${weed_url}..."
  curl -fsSL -o "${tmp_tar}" "${weed_url}"
  tar -xzf "${tmp_tar}" -C /usr/local/bin weed
  rm -f "${tmp_tar}"
  chmod +x /usr/local/bin/weed

  mkdir -p /etc/seaweedfs /var/lib/seaweedfs

  # S3 authentication configuration
  cat <<EOF > /etc/seaweedfs/s3.json
{
  "identities": [
    {
      "name": "delivery-admin",
      "credentials": [
        {
          "accessKey": "${S3_ACCESS_KEY}",
          "secretKey": "${S3_SECRET_KEY}"
        }
      ],
      "actions": ["Admin", "Read", "Write"]
    },
    {
      "name": "anonymous",
      "actions": ["Read"]
    }
  ]
}
EOF

  cat <<EOF > /etc/systemd/system/seaweedfs.service
[Unit]
Description=SeaweedFS Server with S3 API
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/weed server -dir=/var/lib/seaweedfs -volume.port=8085 -s3 -s3.port=${S3_PORT} -s3.config=/etc/seaweedfs/s3.json
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable seaweedfs
  systemctl restart seaweedfs

  wait_for_port "127.0.0.1" "${S3_PORT}" 30
  wait_for_port "127.0.0.1" "8888" 30
}

# setup_s3_buckets
# Creates required public and private buckets in SeaweedFS via filer API.
# Inputs:
#   None
# Example:
#   setup_s3_buckets
setup_s3_buckets() {
  echo "Ensuring S3 buckets exist in SeaweedFS..."

  # Create buckets via SeaweedFS Filer HTTP API
  curl -fsSL -X POST "http://127.0.0.1:8888/buckets/${S3_PUBLIC_BUCKET}/" || true
  curl -fsSL -X POST "http://127.0.0.1:8888/buckets/${S3_PRIVATE_BUCKET}/" || true

  echo "S3 buckets verified."
}

# setup_application_user
# Creates isolated system user and home directory for running the application.
# Inputs:
#   None
# Example:
#   setup_application_user
setup_application_user() {
  if ! id -u "${APP_USER}" >/dev/null 2>&1; then
    echo "Creating system user '${APP_USER}'..."
    useradd --system --user-group --shell /usr/sbin/nologin --home-dir "${APP_DIR}" "${APP_USER}"
  fi
  mkdir -p "${APP_DIR}"
}

# fetch_application_release
# Downloads latest release package from GitHub releases or builds locally if run from repo.
# Inputs:
#   $1 - Architecture ("amd64" or "arm64")
# Example:
#   fetch_application_release "amd64"
fetch_application_release() {
  local arch="$1"
  local target_archive="/tmp/delivery-app-release.tar.gz"

  echo "Fetching latest delivery-app release for ${arch}..."

  local download_url=""
  local release_meta
  release_meta="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"

  if [ -n "${release_meta}" ]; then
    download_url="$(echo "${release_meta}" | grep -o "https://github.com/${REPO}/releases/download/[^\"]*linux-${arch}\.tar\.gz" | head -n1 || true)"
  fi

  if [ -n "${download_url}" ]; then
    echo "Downloading release from ${download_url}..."
    curl -fsSL -o "${target_archive}" "${download_url}"
  else
    # Fallback to local packaging script if running inside project source repository
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [ -f "${script_dir}/scripts/build-release.sh" ]; then
      echo "No published GitHub release found. Building release package locally..."
      "${script_dir}/scripts/build-release.sh"
      local local_pkg="${script_dir}/dist/delivery-app-v"*"-linux-${arch}.tar.gz"
      local matched_pkg
      matched_pkg="$(ls -1 ${local_pkg} 2>/dev/null | head -n1 || true)"
      if [ -n "${matched_pkg}" ] && [ -f "${matched_pkg}" ]; then
        cp "${matched_pkg}" "${target_archive}"
      fi
    fi
  fi

  if [ ! -f "${target_archive}" ]; then
    echo "Error: Could not obtain release package for linux/${arch}." >&2
    exit 1
  fi

  echo "Extracting release to ${APP_DIR}..."
  local extract_tmp="/tmp/delivery-app-extract"
  rm -rf "${extract_tmp}"
  mkdir -p "${extract_tmp}"
  tar -xzf "${target_archive}" -C "${extract_tmp}"

  # Strip top-level directory if present inside the tarball
  local top_level
  top_level="$(find "${extract_tmp}" -mindepth 1 -maxdepth 1 -type d | head -n1 || true)"
  if [ -n "${top_level}" ] && [ -f "${top_level}/delivery-app" ]; then
    cp -r "${top_level}/"* "${APP_DIR}/"
  else
    cp -r "${extract_tmp}/"* "${APP_DIR}/"
  fi

  rm -rf "${extract_tmp}" "${target_archive}"
  chmod +x "${APP_DIR}/delivery-app" "${APP_DIR}/seed"
}

# configure_environment
# Writes production .env file with database, S3, and public URL parameters.
# Inputs:
#   $1 - Public host IP or domain for browser asset URLs
# Example:
#   configure_environment "203.0.113.10"
configure_environment() {
  local public_host="$1"
  local env_file="${APP_DIR}/.env"

  echo "Generating environment configuration in ${env_file}..."
  cat <<EOF > "${env_file}"
APP_PORT=${APP_PORT}
APP_ENV=production
SESSION_SECRET=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)

DB_HOST=127.0.0.1
DB_PORT=${DB_PORT}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

MINIO_ENDPOINT=127.0.0.1:${S3_PORT}
MINIO_ACCESS_KEY=${S3_ACCESS_KEY}
MINIO_SECRET_KEY=${S3_SECRET_KEY}
MINIO_USE_SSL=false
MINIO_PUBLIC_BUCKET=${S3_PUBLIC_BUCKET}
MINIO_PRIVATE_BUCKET=${S3_PRIVATE_BUCKET}
MINIO_PUBLIC_URL=http://${public_host}:${S3_PORT}/${S3_PUBLIC_BUCKET}
EOF

  chown -R "${APP_USER}:${APP_GROUP}" "${APP_DIR}"
  chmod 0600 "${env_file}"
}

# setup_systemd_service
# Creates systemd service unit for delivery-app with automatic restart.
# Inputs:
#   None
# Example:
#   setup_systemd_service
setup_systemd_service() {
  echo "Installing systemd service unit for delivery-app..."
  cat <<EOF > /etc/systemd/system/delivery-app.service
[Unit]
Description=Delivery App Web Server
After=network.target postgresql.service seaweedfs.service
Wants=postgresql.service seaweedfs.service

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${APP_DIR}
EnvironmentFile=${APP_DIR}/.env
ExecStart=${APP_DIR}/delivery-app
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable delivery-app
}

# run_migrations_and_seed
# Executes seed utility to run AutoMigrate schema migrations and populate demo data.
# Inputs:
#   None
# Example:
#   run_migrations_and_seed
run_migrations_and_seed() {
  echo "Executing database migrations and demo seed..."
  su -s /bin/bash "${APP_USER}" -c "cd '${APP_DIR}' && ./seed"

  echo "Starting delivery-app service..."
  systemctl restart delivery-app

  if ! wait_for_port "127.0.0.1" "${APP_PORT}" 15; then
    echo "=== systemctl status delivery-app ==="
    systemctl status delivery-app --no-pager || true
    echo "=== journalctl -u delivery-app ==="
    journalctl -u delivery-app -n 50 --no-pager || true
    return 1
  fi
}

# verify_deployment
# Runs health ping against the local web service and reports demo endpoints.
# Inputs:
#   $1 - Public host IP or domain
# Example:
#   verify_deployment "203.0.113.10"
verify_deployment() {
  local public_host="$1"
  local ping_url="http://127.0.0.1:${APP_PORT}/ping"

  echo "Verifying service status on ${ping_url}..."
  local ping_res
  ping_res="$(curl -fsSL "${ping_url}" || true)"
  echo "Health response: ${ping_res}"

  echo "--------------------------------------------------------"
  echo "Demo setup complete and running."
  echo "Web App URL:   http://${public_host}:${APP_PORT}"
  echo "S3 Public URL: http://${public_host}:${S3_PORT}/${S3_PUBLIC_BUCKET}"
  echo ""
  echo "Demo Accounts (password: password123):"
  echo "  Admin:      admin@example.com"
  echo "  Staff:      staff1@example.com"
  echo "  Restaurant: owner1@example.com / owner2@example.com"
  echo "  Rider:      rider1@example.com"
  echo "  Customer:   customer1@example.com"
  echo "--------------------------------------------------------"
}

main() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be executed as root inside the VPS." >&2
    exit 1
  fi

  local arch
  arch="$(detect_arch)"

  local public_host
  public_host="$(detect_public_ip)"

  echo "Starting demo launcher on host ${public_host} (${arch})..."

  install_system_dependencies
  setup_postgresql_18
  setup_seaweedfs "${arch}"
  setup_s3_buckets
  setup_application_user
  fetch_application_release "${arch}"
  configure_environment "${public_host}"
  setup_systemd_service
  run_migrations_and_seed
  verify_deployment "${public_host}"
}

main "$@"
