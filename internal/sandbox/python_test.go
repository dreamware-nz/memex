package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunPython_HelloWorld(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available on PATH")
	}
	res, err := RunPython(context.Background(), `print("hello")`, RunOptions{})
	if err != nil {
		t.Fatalf("RunPython returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunPython_SyntaxError(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available on PATH")
	}
	res, err := RunPython(context.Background(), `def (:`, RunOptions{})
	if err != nil {
		t.Fatalf("RunPython returned error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero")
	}
	if res.Stderr == "" {
		t.Errorf("Stderr is empty, want non-empty")
	}
}

func TestRunPython_Timeout(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available on PATH")
	}
	start := time.Now()
	res, err := RunPython(context.Background(), `import time; time.sleep(60)`, RunOptions{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunPython returned error: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("RunPython took %v, want < 1s", elapsed)
	}
}

func TestRunPython_MissingRuntime(t *testing.T) {
	if _, err := exec.LookPath("python3"); err == nil {
		t.Skip("python3 is available; cannot test missing-runtime path")
	}
	_, err := RunPython(context.Background(), `print("hi")`, RunOptions{})
	if err == nil {
		t.Fatal("RunPython returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "python3") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "python3")
	}
}
