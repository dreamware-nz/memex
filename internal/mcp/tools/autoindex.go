package tools

import "strings"

// routeOutput dispatches captured output through the three-branch policy
// matching context-mode/src/server.ts:
//
//  1. intent set + > IntentSearchThreshold bytes → intent-driven indexed
//     search response (matched section previews + Searchable terms hint).
//  2. > LargeOutputThreshold bytes (no intent) → pointer-style auto-index
//     summary referencing the source label.
//  3. otherwise → raw output returned inline.
//
// indexed reports whether the KB was written to so callers can decide
// whether to suppress non-zero-exit signalling.
func routeOutput(output, intent, sourceTag string, store IntentIndexer) (text string, indexed bool, err error) {
	trimmedIntent := strings.TrimSpace(intent)
	size := len(output)

	if trimmedIntent != "" && size > IntentSearchThreshold {
		text, err = intentSearch(output, trimmedIntent, sourceTag, store, intentSearchMaxResults)
		if err != nil {
			return "", false, err
		}
		return text, true, nil
	}

	if size > LargeOutputThreshold {
		text, err = indexStdoutPointer(output, sourceTag, store)
		if err != nil {
			return "", false, err
		}
		return text, true, nil
	}

	return output, false, nil
}
