package hooks

import (
	"strings"
	"testing"
)

func TestRoutingBlockNamesMemexTools(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		tokens []string
	}{
		{"SubagentRoutingBlock", SubagentRoutingBlock, []string{
			"memex_fetch_and_index",
			"memex_search",
			"memex_execute",
			"memex_execute_file",
		}},
		{"BashGuidance", BashGuidance, []string{
			"memex_execute",
			"memex_batch_execute",
		}},
		{"ReadGuidance", ReadGuidance, []string{
			"memex_execute_file",
		}},
		{"GrepGuidance", GrepGuidance, []string{
			"memex_execute",
		}},
		{"CurlBlockEcho", CurlBlockEcho, []string{
			"memex_execute",
			"memex_fetch_and_index",
		}},
		{"InlineHTTPBlockEcho", InlineHTTPBlockEcho, []string{
			"memex_execute",
		}},
		{"BuildToolBlockEcho", BuildToolBlockEcho, []string{
			"memex_execute",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, tok := range tc.tokens {
				if !strings.Contains(tc.body, tok) {
					t.Errorf("%s missing token %q\nbody: %s", tc.name, tok, tc.body)
				}
			}
		})
	}
}

func TestWebFetchDenyReasonMentionsMemexAndURL(t *testing.T) {
	t.Run("with URL", func(t *testing.T) {
		r := WebFetchDenyReason("https://example.com/foo")
		for _, tok := range []string{"memex_fetch_and_index", "memex_search", "https://example.com/foo"} {
			if !strings.Contains(r, tok) {
				t.Errorf("missing %q in %q", tok, r)
			}
		}
	})
	t.Run("empty URL", func(t *testing.T) {
		r := WebFetchDenyReason("")
		for _, tok := range []string{"memex_fetch_and_index", "memex_search"} {
			if !strings.Contains(r, tok) {
				t.Errorf("missing %q in %q", tok, r)
			}
		}
	})
}
