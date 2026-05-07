package sandbox

import (
	"context"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"echo", "hello"},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.TimedOut {
		t.Errorf("TimedOut = true, want false")
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"sh", "-c", "exit 42"},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", res.ExitCode)
	}
}

func TestRun_Timeout(t *testing.T) {
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"sleep", "60"},
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("TimedOut = false, want true")
	}
}

func TestRun_ByteCap(t *testing.T) {
	maxBytes := 1024
	res, err := Run(context.Background(), RunOptions{
		// Emit ~1 MiB of zeros.
		Command:  []string{"sh", "-c", "head -c 1048576 /dev/zero"},
		MaxBytes: maxBytes,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(res.Stdout) > maxBytes {
		t.Errorf("len(Stdout) = %d, want <= %d", len(res.Stdout), maxBytes)
	}
}
