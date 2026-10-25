#!/usr/bin/env bash
set -euo pipefail

# Build release artifacts for multiple OS/ARCH targets and package them.

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly DIST_DIR="${ROOT_DIR}/dist"

# Target platform matrix: (OS ARCH EXTENSION_ARCHIVE)
readonly TARGET_PLATFORMS=(
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
)

# resolve_version
# Determines the release version string from argument, env, VERSION file, or git.
# Inputs:
#   $1 - optional explicit version passed as CLI argument
# Output:
#   Echoes version string without leading 'v'
# Example:
#   ver="$(resolve_version "v1.2.0")"
resolve_version() {
  local explicit_version="${1:-}"

  if [ -n "${explicit_version}" ]; then
    printf '%s' "${explicit_version#v}"
    return 0
  fi

  if [ -n "${GITHUB_REF_NAME:-}" ]; then
    printf '%s' "${GITHUB_REF_NAME#v}"
    return 0
  fi

  if [ -f "${ROOT_DIR}/VERSION" ]; then
    local v
    v="$(tr -d '[:space:]' < "${ROOT_DIR}/VERSION")"
    if [ -n "${v}" ]; then
      printf '%s' "${v#v}"
      return 0
    fi
  fi

  local git_tag
  git_tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  if [ -n "${git_tag}" ]; then
    printf '%s' "${git_tag#v}"
    return 0
  fi

  printf '%s' "0.1.0"
}

# build_target_binaries
# Compiles server and seed binaries for a specific GOOS and GOARCH.
# Inputs:
#   $1 - Target GOOS (e.g. linux, darwin, windows)
#   $2 - Target GOARCH (e.g. amd64, arm64)
#   $3 - Destination directory for compiled binaries
# Example:
#   build_target_binaries "linux" "amd64" "/tmp/stage/bin"
build_target_binaries() {
  local target_os="$1"
  local target_arch="$2"
  local dest_dir="$3"

  local binary_suffix=""
  if [ "${target_os}" = "windows" ]; then
    binary_suffix=".exe"
  fi

  mkdir -p "${dest_dir}"

  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
    go build -trimpath -ldflags="-s -w" \
    -o "${dest_dir}/delivery-app${binary_suffix}" "${ROOT_DIR}/cmd/server"

  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
    go build -trimpath -ldflags="-s -w" \
    -o "${dest_dir}/seed${binary_suffix}" "${ROOT_DIR}/cmd/seed"
}

# package_target
# Bundles binaries, templates, static assets, and documentation into an archive.
# Inputs:
#   $1 - Target GOOS
#   $2 - Target GOARCH
#   $3 - Archive type (tar.gz or zip)
#   $4 - Release version
# Example:
#   package_target "linux" "amd64" "tar.gz" "1.0.0"
package_target() {
  local target_os="$1"
  local target_arch="$2"
  local archive_ext="$3"
  local version="$4"

  local archive_name="delivery-app-v${version}-${target_os}-${target_arch}"
  local stage_dir="${DIST_DIR}/tmp/${archive_name}"

  rm -rf "${stage_dir}"
  mkdir -p "${stage_dir}"

  build_target_binaries "${target_os}" "${target_arch}" "${stage_dir}"

  # Copy runtime dependencies required by the server if present
  if [ -d "${ROOT_DIR}/templates" ]; then
    cp -r "${ROOT_DIR}/templates" "${stage_dir}/templates"
  fi
  if [ -d "${ROOT_DIR}/static" ]; then
    cp -r "${ROOT_DIR}/static" "${stage_dir}/static"
  fi
  if [ -d "${ROOT_DIR}/mock" ]; then
    cp -r "${ROOT_DIR}/mock" "${stage_dir}/mock"
  fi
  if [ -f "${ROOT_DIR}/.env.example" ]; then
    cp "${ROOT_DIR}/.env.example" "${stage_dir}/.env.example"
  fi
  if [ -f "${ROOT_DIR}/README.md" ]; then
    cp "${ROOT_DIR}/README.md" "${stage_dir}/README.md"
  fi

  local archive_file="${DIST_DIR}/${archive_name}.${archive_ext}"

  if [ "${archive_ext}" = "zip" ]; then
    (cd "${DIST_DIR}/tmp" && zip -r -q "${archive_file}" "${archive_name}")
  else
    tar -czf "${archive_file}" -C "${DIST_DIR}/tmp" "${archive_name}"
  fi

  rm -rf "${stage_dir}"
  echo "Created: ${archive_file}"
}

# generate_checksums
# Computes SHA256 hashes of all release packages in the dist directory.
# Inputs:
#   $1 - Directory containing built archives
# Example:
#   generate_checksums "${ROOT_DIR}/dist"
generate_checksums() {
  local dist_path="$1"
  local checksums_file="${dist_path}/checksums.txt"

  (
    cd "${dist_path}"
    sha256sum delivery-app-* > checksums.txt
  )
  echo "Checksums written to: ${checksums_file}"
}

main() {
  local version
  version="$(resolve_version "${1:-}")"
  echo "Building release artifacts for version v${version}..."

  rm -rf "${DIST_DIR}"
  mkdir -p "${DIST_DIR}/tmp"

  for platform in "${TARGET_PLATFORMS[@]}"; do
    read -r target_os target_arch archive_ext <<< "${platform}"
    echo "Compiling and packaging for ${target_os}/${target_arch}..."
    package_target "${target_os}" "${target_arch}" "${archive_ext}" "${version}"
  done

  rm -rf "${DIST_DIR}/tmp"
  generate_checksums "${DIST_DIR}"
  echo "All release artifacts built successfully in ${DIST_DIR}"
}

main "$@"
