#!/usr/bin/env bash
set -euo pipefail

# Runs Playwright tests headlessly, one test at a time, fail-fast, with terminal text output only.
# No HTML report browser window will be opened.
#
# Arguments:
#   $@ - Optional additional arguments or filters passed directly to Playwright CLI.
#
# Examples:
#   ./dev-test-playwright.sh
#   ./dev-test-playwright.sh tests/order-flow.spec.ts
#   ./dev-test-playwright.sh --project=chromium

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR="${SCRIPT_DIR}/playwright-tests"

if [ ! -d "${TEST_DIR}" ]; then
  echo "Error: Test directory '${TEST_DIR}' not found." >&2
  exit 1
fi

export PW_TEST_HTML_REPORT_OPEN="never"

cd "${TEST_DIR}"
exec pnpm exec playwright test --workers=1 -x --reporter=list "$@"
