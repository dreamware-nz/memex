package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// RunTypeScript writes code to a .ts file in a fresh tempdir and executes it
// with node using the --experimental-strip-types flag (Node ≥22.6). Hosts on
// older Node will see a flag-rejection error; hosts that prefer tsx/ts-node
// can swap the flag externally — the dispatcher just routes the language key.
func RunTypeScript(ctx context.Context, code string, opts RunOptions) (Result, error) {
	nodePath, err := resolveNode()
	if err != nil {
		return Result{}, err
	}

	dir, cleanup, err := newTempDir()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	scriptPath := filepath.Join(dir, "script.ts")
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return Result{}, fmt.Errorf("sandbox: write script: %w", err)
	}

	opts.Command = []string{nodePath, "--experimental-strip-types", "--no-warnings", scriptPath}
	return Run(ctx, opts)
}
