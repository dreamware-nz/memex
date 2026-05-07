package adapters

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectPlatform infers the host agent using a three-tier strategy:
//  1. MEMEX_PLATFORM env var override (high)
//  2. Per-platform env vars in priority order (high)
//  3. Config dir existence under ~/ (medium)
//  4. Fallback to claude-code (low)
//
// Mirrors src/adapters/detect.ts in the TS reference. MCP clientInfo
// detection is intentionally out of scope for this change.
func DetectPlatform() DetectionSignal {
	if override := os.Getenv("MEMEX_PLATFORM"); override != "" {
		if isValidPlatform(override) {
			return DetectionSignal{
				Platform:   override,
				Confidence: "high",
				Reason:     "MEMEX_PLATFORM=" + override + " override",
			}
		}
	}

	for _, p := range platformTable {
		for _, v := range p.EnvVars {
			if os.Getenv(v) != "" {
				return DetectionSignal{
					Platform:   p.ID,
					Confidence: "high",
					Reason:     strings.Join(p.EnvVars, " or ") + " env var set",
				}
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		for _, p := range platformTable {
			if p.SkipConfigDetect || len(p.SessionDir) == 0 {
				continue
			}
			path := filepath.Join(append([]string{home}, p.SessionDir...)...)
			if _, err := os.Stat(path); err == nil {
				return DetectionSignal{
					Platform:   p.ID,
					Confidence: "medium",
					Reason:     "~/" + filepath.Join(p.SessionDir...) + "/ directory exists",
				}
			}
		}
	}

	return DetectionSignal{
		Platform:   "claude-code",
		Confidence: "low",
		Reason:     "fallback",
	}
}
