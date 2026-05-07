#!/usr/bin/env sh
# memex installer. Run via:
#   curl -fsSL https://raw.githubusercontent.com/dreamware-nz/memex/main/install.sh | sh
#
# Requires Go >= 1.25 on PATH (older toolchains are auto-fetched if Go >= 1.21).
# Installs the binary into $(go env GOBIN || go env GOPATH/bin), then runs
# `memex setup install` to wire it into the detected host agent.

set -eu

err() { printf 'error: %s\n' "$1" >&2; exit 1; }
log() { printf '==> %s\n' "$1"; }

command -v go >/dev/null 2>&1 \
  || err "go is required (>= 1.25). Install from https://go.dev/dl, then re-run."

GOBIN="$(go env GOBIN 2>/dev/null || true)"
[ -z "$GOBIN" ] && GOBIN="$(go env GOPATH)/bin"

log "go install github.com/dreamware-nz/memex/cmd/memex@latest"
go install github.com/dreamware-nz/memex/cmd/memex@latest

if [ ! -x "$GOBIN/memex" ]; then
  err "go install completed but $GOBIN/memex is missing — check 'go env GOBIN'"
fi

case ":$PATH:" in
  *":$GOBIN:"*) ;;
  *) printf '\nNOTE: %s is not on PATH. Add to your shell rc:\n  export PATH="%s:$PATH"\n\n' "$GOBIN" "$GOBIN" ;;
esac

log "memex setup install"
"$GOBIN/memex" setup install

log "memex setup validate"
"$GOBIN/memex" setup validate || true

cat <<EOF

memex installed at $GOBIN/memex.

If you use Claude Code, fully quit and reopen it now. MCP servers
declared in plugin .mcp.json files are loaded only at startup, so the
mcp__plugin_memex_memex__* tools will appear after the restart.
EOF
