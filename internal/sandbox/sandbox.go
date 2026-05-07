package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dreamware-nz/memex/internal/kb"
)

const (
	DefaultTimeout        = 30 * time.Second
	DefaultMaxBytes       = 512 * 1024
	DefaultMaxLines       = 200
	DefaultMaxOutputBytes = 32 * 1024
)

// RunOptions configures a sandboxed command invocation.
type RunOptions struct {
	Command        []string
	Timeout        time.Duration
	MaxBytes       int
	MaxLines       int
	MaxOutputBytes int
	Store          *kb.Store
}

// Result holds the captured outcome of a sandboxed command.
type Result struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	TimedOut  bool
	Truncated bool
	IndexedAs string
}

// newTempDir creates a fresh temp directory for a sandbox invocation and
// returns a cleanup closure that removes it.
func newTempDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "ctx-sandbox-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	return dir, cleanup, nil
}

// Run executes opts.Command in an isolated tempdir with a bounded timeout
// and byte-capped stdout/stderr capture.
func Run(ctx context.Context, opts RunOptions) (Result, error) {
	if len(opts.Command) == 0 {
		return Result{}, errors.New("sandbox: empty command")
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.MaxLines == 0 {
		opts.MaxLines = DefaultMaxLines
	}
	if opts.MaxOutputBytes == 0 {
		opts.MaxOutputBytes = DefaultMaxOutputBytes
	}

	dir, cleanup, err := newTempDir()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, opts.Command[0], opts.Command[1:]...)
	cmd.Dir = dir
	cmd.Env = minimalEnv()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, err
	}

	if err := cmd.Start(); err != nil {
		return Result{}, err
	}

	// Drain both pipes concurrently so a full pipe on one stream cannot
	// deadlock the child while we are blocked reading the other.
	var stdoutBytes, stderrBytes []byte
	done := make(chan struct{}, 2)
	go func() {
		stdoutBytes, _ = io.ReadAll(&io.LimitedReader{R: stdoutPipe, N: int64(opts.MaxBytes)})
		_, _ = io.Copy(io.Discard, stdoutPipe)
		done <- struct{}{}
	}()
	go func() {
		stderrBytes, _ = io.ReadAll(&io.LimitedReader{R: stderrPipe, N: int64(opts.MaxBytes)})
		_, _ = io.Copy(io.Discard, stderrPipe)
		done <- struct{}{}
	}()
	<-done
	<-done

	waitErr := cmd.Wait()

	res := Result{
		Stdout: string(stdoutBytes),
		Stderr: string(stderrBytes),
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
	}

	if waitErr != nil && !res.TimedOut {
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			return res, waitErr
		}
	}

	if shouldTruncate(res.Stdout, res.Stderr, opts) {
		truncateResult(&res, opts)
	}

	return res, nil
}

// shouldTruncate reports whether combined stdout+stderr exceeds either the
// line or byte threshold from opts.
func shouldTruncate(stdout, stderr string, opts RunOptions) bool {
	totalBytes := len(stdout) + len(stderr)
	if totalBytes > opts.MaxOutputBytes {
		return true
	}
	lines := strings.Count(stdout, "\n") + strings.Count(stderr, "\n")
	return lines > opts.MaxLines
}

// truncateResult routes the full output into opts.Store (if non-nil) under a
// unique label and replaces r.Stdout with a one-line summary.
func truncateResult(r *Result, opts RunOptions) {
	label := "sandbox/" + time.Now().UTC().Format(time.RFC3339)
	totalLines := strings.Count(r.Stdout, "\n") + strings.Count(r.Stderr, "\n")
	totalBytes := len(r.Stdout) + len(r.Stderr)

	combined := r.Stdout + "\n---\n" + r.Stderr
	indexed := ""
	if opts.Store != nil {
		if err := opts.Store.Index(label, combined); err == nil {
			indexed = label
		}
	}

	var summary string
	if indexed != "" {
		summary = fmt.Sprintf(
			"[output truncated: %d lines, %d bytes — indexed as %s]",
			totalLines, totalBytes, indexed,
		)
	} else {
		summary = fmt.Sprintf(
			"[output truncated: %d lines, %d bytes]",
			totalLines, totalBytes,
		)
	}

	r.Stdout = summary
	r.Truncated = true
	r.IndexedAs = indexed
}

// minimalEnv returns a stripped environment containing only PATH, HOME, and
// TMPDIR from the parent process.
func minimalEnv() []string {
	keys := []string{"PATH", "HOME", "TMPDIR"}
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}
