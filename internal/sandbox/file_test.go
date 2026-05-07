package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestRunFile_Python(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available on PATH")
	}
	path := writeTempFile(t, "hello.py", `print("hello-py")`)
	res, err := RunFile(context.Background(), path, RunOptions{})
	if err != nil {
		t.Fatalf("RunFile returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello-py") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello-py")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunFile_Node(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	path := writeTempFile(t, "hello.js", `console.log("hello-js")`)
	res, err := RunFile(context.Background(), path, RunOptions{})
	if err != nil {
		t.Fatalf("RunFile returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello-js") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello-js")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunFile_Shell(t *testing.T) {
	path := writeTempFile(t, "hello.sh", "#!/bin/sh\necho hi\n")
	res, err := RunFile(context.Background(), path, RunOptions{})
	if err != nil {
		t.Fatalf("RunFile returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hi") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hi")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunFile_UnknownExtension(t *testing.T) {
	path := writeTempFile(t, "script.rb", `puts "hi"`)
	_, err := RunFile(context.Background(), path, RunOptions{})
	if err == nil {
		t.Fatal("RunFile returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), ".rb") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), ".rb")
	}
}
