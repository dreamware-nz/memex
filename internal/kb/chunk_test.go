package kb

import (
	"strings"
	"testing"
)

func TestChunkMarkdown(t *testing.T) {
	cases := []struct {
		name    string
		content string
		max     int
		assert  func(t *testing.T, secs []Section)
	}{
		{
			name:    "no headings yields single section",
			content: "alpha bravo charlie\n\ndelta echo foxtrot",
			max:     2048,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) != 1 {
					t.Fatalf("len = %d, want 1", len(secs))
				}
				if secs[0].Heading != "" {
					t.Fatalf("heading = %q, want empty", secs[0].Heading)
				}
				if !strings.Contains(secs[0].Body, "alpha") || !strings.Contains(secs[0].Body, "foxtrot") {
					t.Fatalf("body missing content: %q", secs[0].Body)
				}
			},
		},
		{
			name: "multi-level headings build path",
			content: "# Top\n\nintro\n\n## Sub A\n\nbody A\n\n## Sub B\n\nbody B\n\n### Deep\n\ndeep body\n",
			max:  2048,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) != 4 {
					t.Fatalf("len = %d, want 4 (intro, Sub A, Sub B, Deep): %#v", len(secs), secs)
				}
				wantHeadings := []string{
					"Top",
					"Top > Sub A",
					"Top > Sub B",
					"Top > Sub B > Deep",
				}
				for i, w := range wantHeadings {
					if secs[i].Heading != w {
						t.Fatalf("section[%d].Heading = %q, want %q", i, secs[i].Heading, w)
					}
				}
				if !strings.Contains(secs[3].Body, "deep body") {
					t.Fatalf("deep body missing: %q", secs[3].Body)
				}
			},
		},
		{
			name:    "oversized paragraph splits with part suffix",
			content: "# H\n\n" + strings.Repeat("x", 5000),
			max:     1024,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) < 2 {
					t.Fatalf("expected >=2 parts, got %d", len(secs))
				}
				for i, s := range secs {
					if len(s.Body) > 1024 {
						t.Fatalf("section[%d] body=%d > 1024", i, len(s.Body))
					}
					if !strings.Contains(s.Heading, "part ") {
						t.Fatalf("section[%d] heading missing part suffix: %q", i, s.Heading)
					}
				}
			},
		},
		{
			name:    "code fence passthrough ignores inner #",
			content: "# Outer\n\nbefore\n\n```\n# not-a-heading\n## also-not\n```\n\nafter\n",
			max:     2048,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) != 1 {
					t.Fatalf("len = %d, want 1: %#v", len(secs), secs)
				}
				if secs[0].Heading != "Outer" {
					t.Fatalf("heading = %q, want %q", secs[0].Heading, "Outer")
				}
				if !strings.Contains(secs[0].Body, "# not-a-heading") {
					t.Fatalf("fenced content lost: %q", secs[0].Body)
				}
			},
		},
		{
			name:    "skipped heading levels do not panic",
			content: "### Deep First\n\nbody\n",
			max:     2048,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) != 1 {
					t.Fatalf("len = %d, want 1", len(secs))
				}
				if !strings.Contains(secs[0].Heading, "Deep First") {
					t.Fatalf("heading missing 'Deep First': %q", secs[0].Heading)
				}
			},
		},
		{
			name:    "empty content yields no sections",
			content: "",
			max:     2048,
			assert: func(t *testing.T, secs []Section) {
				if len(secs) != 0 {
					t.Fatalf("len = %d, want 0", len(secs))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, chunkMarkdown(tc.content, tc.max))
		})
	}
}
