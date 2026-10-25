#!/usr/bin/env bash
set -euo pipefail

# Release automation script: bumps version, commits changes, and tags git release.
# Does not push to remote unless --push flag is explicitly provided.

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly VERSION_FILE="${SCRIPT_DIR}/VERSION"

# Allowed bump types (closed variant set)
readonly BUMP_TYPES=("major" "minor" "patch")

# print_usage
# Displays command-line usage information.
# Inputs:
#   None
# Example:
#   print_usage
print_usage() {
  cat <<'EOF'
Usage: ./release.sh <major|minor|patch> [--push]

Arguments:
  <major|minor|patch>   Version component to bump (required)
  --push                Push commit and tag to origin remote (optional)
  -h, --help            Show this help message

Examples:
  ./release.sh patch
  ./release.sh minor --push
  ./release.sh major
EOF
}

# is_valid_bump_type
# Checks if the provided bump argument is one of the allowed bump types.
# Inputs:
#   $1 - candidate bump type string
# Returns:
#   0 if valid, 1 otherwise
# Example:
#   is_valid_bump_type "minor"
is_valid_bump_type() {
  local candidate="$1"
  for allowed in "${BUMP_TYPES[@]}"; do
    if [ "${candidate}" = "${allowed}" ]; then
      return 0
    fi
  done
  return 1
}

# parse_args
# Parses CLI arguments for bump type and push flag.
# Inputs:
#   $@ - All arguments passed to script
# Outputs variables:
#   BUMP_TYPE - Selected bump type (major/minor/patch)
#   DO_PUSH   - "true" or "false"
# Example:
#   parse_args "$@"
parse_args() {
  BUMP_TYPE=""
  DO_PUSH="false"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --push)
        DO_PUSH="true"
        shift
        ;;
      -h|--help)
        print_usage
        exit 0
        ;;
      major|minor|patch)
        if [ -n "${BUMP_TYPE}" ]; then
          echo "Error: Multiple bump types specified ('${BUMP_TYPE}' and '$1')." >&2
          print_usage >&2
          exit 1
        fi
        BUMP_TYPE="$1"
        shift
        ;;
      *)
        echo "Error: Unknown argument '$1'." >&2
        print_usage >&2
        exit 1
        ;;
    esac
  done

  if [ -z "${BUMP_TYPE}" ]; then
    echo "Error: Missing required bump type (major, minor, or patch)." >&2
    print_usage >&2
    exit 1
  fi
}

# ensure_clean_working_tree
# Verifies that git tracked files have no uncommitted changes.
# Inputs:
#   None
# Example:
#   ensure_clean_working_tree
ensure_clean_working_tree() {
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: Current directory is not a git repository." >&2
    exit 1
  fi

  if ! git diff-index --quiet HEAD --; then
    echo "Error: Working directory has uncommitted changes. Please commit or stash first." >&2
    exit 1
  fi
}

# get_current_version
# Reads current semantic version from VERSION file or latest git tag.
# Inputs:
#   None
# Output:
#   Echoes version string (X.Y.Z) without leading 'v'
# Example:
#   current_ver="$(get_current_version)"
get_current_version() {
  local ver=""

  if [ -f "${VERSION_FILE}" ]; then
    ver="$(tr -d '[:space:]' < "${VERSION_FILE}")"
  fi

  if [ -z "${ver}" ]; then
    local tag
    tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
    if [ -n "${tag}" ]; then
      ver="${tag#v}"
    fi
  fi

  if [ -z "${ver}" ]; then
    ver="0.0.0"
  fi

  ver="${ver#v}"

  # Validate semver format X.Y.Z
  if [[ ! "${ver}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Invalid version format '${ver}'. Expected X.Y.Z semver." >&2
    exit 1
  fi

  printf '%s' "${ver}"
}

# calculate_bumped_version
# Computes the next semantic version given the current version and bump type.
# Inputs:
#   $1 - Current version string in X.Y.Z format
#   $2 - Bump type: major, minor, or patch
# Output:
#   Echoes incremented semantic version
# Example:
#   new_ver="$(calculate_bumped_version "1.2.3" "minor")"
calculate_bumped_version() {
  local current="$1"
  local bump="$2"

  local major minor patch
  IFS='.' read -r major minor patch <<< "${current}"

  case "${bump}" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
    *)
      echo "Error: Unsupported bump type '${bump}'." >&2
      exit 1
      ;;
  esac

  printf '%d.%d.%d' "${major}" "${minor}" "${patch}"
}

# tag_exists
# Checks if a given git tag already exists locally or in refs.
# Inputs:
#   $1 - Tag name (e.g. v1.0.0)
# Returns:
#   0 if tag exists, 1 if not
# Example:
#   if tag_exists "v1.0.0"; then ...
tag_exists() {
  local tag_name="$1"
  git rev-parse -q --verify "refs/tags/${tag_name}" >/dev/null 2>&1
}

main() {
  parse_args "$@"
  ensure_clean_working_tree

  local current_version
  current_version="$(get_current_version)"

  local new_version
  new_version="$(calculate_bumped_version "${current_version}" "${BUMP_TYPE}")"
  local new_tag="v${new_version}"

  if tag_exists "${new_tag}"; then
    echo "Error: Tag '${new_tag}' already exists." >&2
    exit 1
  fi

  echo "Bumping version: ${current_version} -> ${new_version} (${BUMP_TYPE})"

  # 1. Bump version file
  printf '%s\n' "${new_version}" > "${VERSION_FILE}"

  # 2. Commit version bump
  git add "${VERSION_FILE}"
  git commit -m "chore(release): ${new_tag}"

  # 3. Tag commit
  git tag -a "${new_tag}" -m "Release ${new_tag}"

  echo "Created commit and tag ${new_tag} successfully."

  # 4. Handle push condition
  if [ "${DO_PUSH}" = "true" ]; then
    local current_branch
    current_branch="$(git rev-parse --abbrev-ref HEAD)"
    echo "Pushing branch '${current_branch}' and tag '${new_tag}' to origin..."
    git push origin "${current_branch}"
    git push origin "${new_tag}"
    echo "Pushed release ${new_tag} to origin."
  else
    echo "Notice: No push performed (default). To push now, run:"
    echo "  git push origin HEAD && git push origin ${new_tag}"
  fi
}

main "$@"
