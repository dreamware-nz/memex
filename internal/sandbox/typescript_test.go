package sandbox

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunTypeScript_HelloWorld(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	res, err := RunTypeScript(context.Background(), `const msg: string = 'hello'; console.log(msg)`, RunOptions{})
	if err != nil {
		t.Fatalf("RunTypeScript returned error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (stderr=%q)", res.ExitCode, res.Stderr)
	}
}

func TestRunTypeScript_RuntimeError(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	res, err := RunTypeScript(context.Background(), `throw new Error('boom')`, RunOptions{})
	if err != nil {
		t.Fatalf("RunTypeScript returned error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero")
	}
	if res.Stderr == "" {
		t.Errorf("Stderr is empty, want non-empty")
	}
}

func TestRunTypeScript_Timeout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available on PATH")
	}
	start := time.Now()
	res, err := RunTypeScript(context.Background(), `while (true) {}`, RunOptions{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunTypeScript returned error: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("RunTypeScript took %v, want < 1s", elapsed)
	}
}
