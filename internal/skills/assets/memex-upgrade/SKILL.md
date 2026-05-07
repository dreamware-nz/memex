---
name: memex-upgrade
description: |
  Upgrade memex to the latest source via `go install`.
  Returns the install command for the agent to execute.
  Trigger: /memex:memex-upgrade
user-invocable: true
---

# memex upgrade

Pull the latest memex from source and reinstall it.

## Instructions

1. Call the `memex_upgrade` MCP tool (no arguments). It returns the canonical `go install` shell command.
2. Run the returned command using your shell tool (Bash). Example:
   ```
   go install github.com/dreamware-nz/memex/cmd/memex@latest
   ```
3. Re-run `memex setup install --platform <host>` so the new binary's hooks and skills land in place.
4. Tell the user to **restart their session** to pick up the new version.
5. **Fallback** (only if the MCP tool is unavailable): run `memex upgrade` via Bash to print the same install command. The Go toolchain is the only runtime required — there is no Node.js or TypeScript step.
