package kb

import "strings"

// sanitizeFTS5Query rewrites a user-supplied query so it parses cleanly
// against the FTS5 grammar. Two rewrites: (1) hyphenated identifiers like
// "polyglot-sandbox-77" become quoted phrases so FTS5 doesn't mis-parse the
// internal `-` as a column filter; (2) tokens beginning with a single `-`
// are rewritten to `NOT <rest>` so the natural `foo -bar` syntax actually
// excludes bar — FTS5's standard grammar uses the binary `NOT` keyword,
// not unary `-`. Already-quoted phrases pass through unchanged. The default
// tokenizer still splits on `-` at index time, so phrase-quoting a
// hyphenated identifier means "the indexed tokens must appear adjacently".
func sanitizeFTS5Query(q string) string {
	var (
		out strings.Builder
		cur strings.Builder
	)
	out.Grow(len(q) + 8)

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		token := cur.String()
		cur.Reset()
		if isNegationToken(token) {
			rest := token[1:]
			out.WriteString("NOT ")
			if needsPhraseWrap(rest) {
				out.WriteByte('"')
				out.WriteString(strings.ReplaceAll(rest, `"`, `""`))
				out.WriteByte('"')
			} else {
				out.WriteString(rest)
			}
			return
		}
		if needsPhraseWrap(token) {
			out.WriteByte('"')
			out.WriteString(strings.ReplaceAll(token, `"`, `""`))
			out.WriteByte('"')
			return
		}
		out.WriteString(token)
	}

	for i := 0; i < len(q); i++ {
		c := q[i]
		switch {
		case c == '"':
			flush()
			out.WriteByte(c)
			for i++; i < len(q); i++ {
				out.WriteByte(q[i])
				if q[i] == '"' {
					break
				}
			}
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
			out.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out.String()
}

// isNegationToken reports whether an unquoted token looks like
// user-typed negation: a single leading `-` followed by at least one
// non-`-` character. `--foo` and bare `-` are left alone (broken
// syntax that the sanitiser doesn't try to rescue).
func isNegationToken(tok string) bool {
	return len(tok) >= 2 && tok[0] == '-' && tok[1] != '-'
}

// needsPhraseWrap reports whether an unquoted token has an internal `-`
// that would otherwise be parsed as FTS5 negation. Leading `-` runs are
// real negation operators and are left alone; tokens shorter than two
// bytes cannot have an internal hyphen.
func needsPhraseWrap(tok string) bool {
	if len(tok) < 2 {
		return false
	}
	start := 0
	for start < len(tok) && tok[start] == '-' {
		start++
	}
	if start >= len(tok) {
		return false
	}
	for i := start; i < len(tok); i++ {
		if tok[i] == '-' {
			return true
		}
	}
	return false
}
