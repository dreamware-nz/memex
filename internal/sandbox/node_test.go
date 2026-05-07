package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunNode_HelloWorld(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	res, err := RunNode(context.Background(), `console.log('hello')`, RunOptions{})
	if err != nil {
		t.Fatalf("RunNode returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunNode_RuntimeError(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	res, err := RunNode(context.Background(), `throw new Error('boom')`, RunOptions{})
	if err != nil {
		t.Fatalf("RunNode returned error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero")
	}
	if res.Stderr == "" {
		t.Errorf("Stderr is empty, want non-empty")
	}
}

func TestRunNode_Timeout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	start := time.Now()
	res, err := RunNode(context.Background(), `while (true) {}`, RunOptions{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunNode returned error: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("RunNode took %v, want < 1s", elapsed)
	}
}

func TestRunNode_MissingRuntime(t *testing.T) {
	if _, err := exec.LookPath("node"); err == nil {
		t.Skip("node is available; cannot test missing-runtime path")
	}
	_, err := RunNode(context.Background(), `console.log('hi')`, RunOptions{})
	if err == nil {
		t.Fatal("RunNode returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "node") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "node")
	}
}
