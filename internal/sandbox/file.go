package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// langByExt maps a lowercase file extension (including the leading dot) to a
// language key understood by RunFile.
var langByExt = map[string]string{
	".py": "python",
	".js": "node",
	".ts": "typescript",
	".sh": "shell",
}

// detectLanguage returns the language key for the given file path's
// extension. The lookup is case-insensitive; an unrecognised extension
// produces an error whose message contains the offending extension.
func detectLanguage(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := langByExt[ext]; ok {
		return lang, nil
	}
	return "", fmt.Errorf("sandbox: unsupported file extension %q", ext)
}

// RunFile reads the file at path, detects the runtime from its extension,
// and dispatches to the matching sandbox runner. opts is forwarded
// unchanged. Unknown extensions return an error mentioning the extension.
func RunFile(ctx context.Context, path string, opts RunOptions) (Result, error) {
	lang, err := detectLanguage(path)
	if err != nil {
		return Result{}, err
	}
	code, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("sandbox: read file: %w", err)
	}
	switch lang {
	case "python":
		return RunPython(ctx, string(code), opts)
	case "node":
		return RunNode(ctx, string(code), opts)
	case "typescript":
		return RunTypeScript(ctx, string(code), opts)
	case "shell":
		opts.Command = []string{"sh", "-c", string(code)}
		return Run(ctx, opts)
	}
	return Result{}, fmt.Errorf("sandbox: unsupported language %q", lang)
}
