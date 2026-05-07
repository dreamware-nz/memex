package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// resolvePython3 returns the absolute path to the python3 binary, or a
// descriptive error if it is not found on PATH.
func resolvePython3() (string, error) {
	path, err := exec.LookPath("python3")
	if err != nil {
		return "", fmt.Errorf("sandbox: python3 not found on PATH: %w", err)
	}
	return path, nil
}

// RunPython writes code to a script file in a fresh tempdir and executes it
// with python3 via Run. opts.Command is ignored; all other RunOptions fields
// (Timeout, MaxBytes) pass through to Run.
func RunPython(ctx context.Context, code string, opts RunOptions) (Result, error) {
	pythonPath, err := resolvePython3()
	if err != nil {
		return Result{}, err
	}

	dir, cleanup, err := newTempDir()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	scriptPath := filepath.Join(dir, "script.py")
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return Result{}, fmt.Errorf("sandbox: write script: %w", err)
	}

	opts.Command = []string{pythonPath, scriptPath}
	return Run(ctx, opts)
}
