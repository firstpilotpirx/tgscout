#!/usr/bin/env bash
# The project checks of tgscout (AGENTS.md § How to verify a change).
#   scripts/qa.sh quick   vet, lint, architecture, tidy, unit tests with coverage
#   scripts/qa.sh [full]  quick + vulnerabilities, complexity ratchet, spelling,
#                         shell scripts, duplication, integration tests, E2E, gitleaks
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
service="$root/tgscout"
level="${1:-full}"
# Tools installed with `go install` live in GOPATH/bin.
PATH="$PATH:$(go env GOPATH)/bin"

case "$level" in
  quick | full) ;;
  -h | --help)
    sed -n '2,6p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "✗ unknown level: $level (quick or full)" >&2
    exit 2
    ;;
esac

step() { printf '\n→ %s\n' "$*"; }

# require_tool <name> <expected version text> <install command> <version command...>
# Fails when the tool is missing or at another version (R-139); installs nothing.
require_tool() {
  local name="$1" want="$2" install="$3"
  shift 3
  if ! command -v "$name" > /dev/null 2>&1; then
    echo "✗ $name is missing; install it with: $install" >&2
    exit 1
  fi
  local got
  got="$("$@" 2>&1 | head -n 3)"
  if [[ "$got" != *"$want"* ]]; then
    echo "✗ $name is not at $want (found: ${got%%$'\n'*}); install it with: $install" >&2
    exit 1
  fi
}

require_quick_tools() {
  require_tool golangci-lint "version 2.15.0" \
    "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0" golangci-lint --version
  require_tool go-arch-lint "v1.19.1" \
    "go install github.com/fe3dback/go-arch-lint@v1.19.1" go-arch-lint version --output-color=false
}

require_full_tools() {
  require_tool govulncheck "govulncheck@v1.9.0" \
    "go install golang.org/x/vuln/cmd/govulncheck@v1.9.0" govulncheck -version
  require_tool lizard "1.24.2" "uv tool install lizard==1.24.2" lizard --version
  require_tool typos "typos-cli 1.51.1" "uv tool install typos==1.51.1" typos --version
  require_tool shellcheck "version: 0.11.0" "uv tool install shellcheck-py==0.11.0.1" shellcheck --version
  require_tool jscpd "5.4.2" "npm install --global jscpd@5.4.2" jscpd --version
  require_tool gitleaks "8.30.1" "brew install gitleaks (8.30.1)" gitleaks version
}

quick() {
  require_quick_tools
  cd "$service"
  step "go vet"
  go vet ./...
  step "golangci-lint"
  golangci-lint run --config "$root/.golangci.yml" ./...
  step "go-arch-lint"
  go-arch-lint check --project-path "$service" --arch-file "$root/.go-arch-lint.yml" --output-color=false
  step "go mod tidy -diff"
  go mod tidy -diff
  step "unit tests (race, coverage)"
  go test -race -coverprofile="${TMPDIR:-/tmp}/tgscout-cover.out" ./...
}

e2e() {
  local bin_dir
  bin_dir="$(mktemp -d)"
  step "build tgscout"
  (cd "$service" && go build -tags telegramfake -o "$bin_dir/tgscout" ./apps/cli)
  step "E2E (godog)"
  (cd "$root/e2e" && go vet ./... && golangci-lint run --config "$root/.golangci.yml" ./... &&
    go mod tidy -diff && SUT_BIN="$bin_dir/tgscout" go test -race -count=1 ./...)
  rm -rf "$bin_dir"
}

full() {
  require_full_tools
  cd "$service"
  step "govulncheck"
  govulncheck ./...
  step "complexity ratchet"
  lizard --csv "$service" "$root/e2e" | "$root/scripts/check-complexity-ratchet.sh"
  step "typos"
  (cd "$root" && typos)
  step "shellcheck"
  shellcheck "$root"/scripts/*.sh "$root"/.githooks/*
  step "jscpd"
  (cd "$root" && jscpd --config .jscpd.json --exit-code 1 tgscout e2e)
  step "integration tests"
  go test -race -tags=integration ./...
  e2e
  step "gitleaks"
  (cd "$root" && gitleaks git --redact)
}

quick
if [[ "$level" == full ]]; then
  full
fi
if [[ -z "$(git -C "$root" config core.hooksPath)" ]]; then
  echo "⚠ the gitleaks pre-commit hook is off; run: git config core.hooksPath .githooks" >&2
fi
printf '\n✓ qa %s passed\n' "$level"
