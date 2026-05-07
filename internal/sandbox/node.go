package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// resolveNode returns the absolute path to the node binary, or a descriptive
// error if it is not found on PATH.
func resolveNode() (string, error) {
	path, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("sandbox: node not found on PATH: %w", err)
	}
	return path, nil
}

// RunNode writes code to a script file in a fresh tempdir and executes it
// with node via Run. opts.Command is ignored; all other RunOptions fields
// (Timeout, MaxBytes) pass through to Run.
func RunNode(ctx context.Context, code string, opts RunOptions) (Result, error) {
	nodePath, err := resolveNode()
	if err != nil {
		return Result{}, err
	}

	dir, cleanup, err := newTempDir()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	scriptPath := filepath.Join(dir, "script.js")
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return Result{}, fmt.Errorf("sandbox: write script: %w", err)
	}

	opts.Command = []string{nodePath, scriptPath}
	return Run(ctx, opts)
}
