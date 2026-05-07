package kb

import (
	"strconv"
	"strings"
)

type Section struct {
	Heading string
	Body    string
}

const defaultChunkBytes = 2048

func chunkMarkdown(content string, maxBytes int) []Section {
	if maxBytes <= 0 {
		maxBytes = defaultChunkBytes
	}

	lines := strings.Split(content, "\n")
	var (
		sections       []Section
		stack          []string
		bodyLines      []string
		currentHeading string
		inFence        bool
	)

	flush := func() {
		if len(bodyLines) == 0 {
			return
		}
		body := strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")
		bodyLines = bodyLines[:0]
		if strings.TrimSpace(body) == "" {
			return
		}
		sections = append(sections, splitOversized(currentHeading, body, maxBytes)...)
	}

	for _, line := range lines {
		if isFenceLine(line) {
			inFence = !inFence
			bodyLines = append(bodyLines, line)
			continue
		}
		if !inFence {
			if level, text, ok := parseHeading(line); ok && level >= 1 && level <= 4 {
				flush()
				if level-1 <= len(stack) {
					stack = stack[:level-1]
				}
				for len(stack) < level-1 {
					stack = append(stack, "")
				}
				stack = append(stack, text)
				currentHeading = strings.Join(stack, " > ")
				continue
			}
		}
		bodyLines = append(bodyLines, line)
	}
	flush()

	return sections
}

func isFenceLine(line string) bool {
	s := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(s, "```") || strings.HasPrefix(s, "~~~")
}

func parseHeading(line string) (int, string, bool) {
	s := strings.TrimLeft(line, " ")
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0, "", false
	}
	if n == len(s) || s[n] != ' ' {
		return 0, "", false
	}
	return n, strings.TrimSpace(s[n+1:]), true
}

// chunkPlainText splits arbitrary content into Sections without heading
// awareness. Used by IndexPlainText for execution output and other non-markdown
// payloads. It groups paragraphs (blank-line separated) and falls through to
// splitOversized for any over-cap chunk.
func chunkPlainText(content string, maxBytes int) []Section {
	if maxBytes <= 0 {
		maxBytes = defaultChunkBytes
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	paras := strings.Split(content, "\n\n")
	var (
		sections []Section
		cur      strings.Builder
	)
	flush := func() {
		body := strings.TrimRight(cur.String(), "\n")
		cur.Reset()
		if strings.TrimSpace(body) == "" {
			return
		}
		sections = append(sections, splitOversized("", body, maxBytes)...)
	}

	for _, p := range paras {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if cur.Len() == 0 {
			cur.WriteString(p)
			continue
		}
		if cur.Len()+2+len(p) > maxBytes {
			flush()
			cur.WriteString(p)
			continue
		}
		cur.WriteString("\n\n")
		cur.WriteString(p)
	}
	flush()

	if len(sections) == 0 {
		sections = splitOversized("", trimmed, maxBytes)
	}
	return sections
}

func splitOversized(heading, body string, maxBytes int) []Section {
	if len(body) <= maxBytes {
		return []Section{{Heading: heading, Body: body}}
	}

	paras := strings.Split(body, "\n\n")
	var chunks []string
	var cur strings.Builder
	for _, p := range paras {
		if cur.Len() == 0 {
			cur.WriteString(p)
			continue
		}
		if cur.Len()+2+len(p) > maxBytes {
			chunks = append(chunks, cur.String())
			cur.Reset()
			cur.WriteString(p)
			continue
		}
		cur.WriteString("\n\n")
		cur.WriteString(p)
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}

	var final []string
	for _, c := range chunks {
		if len(c) <= maxBytes {
			final = append(final, c)
			continue
		}
		for i := 0; i < len(c); i += maxBytes {
			end := i + maxBytes
			if end > len(c) {
				end = len(c)
			}
			final = append(final, c[i:end])
		}
	}

	if len(final) == 1 {
		return []Section{{Heading: heading, Body: final[0]}}
	}

	out := make([]Section, 0, len(final))
	for i, c := range final {
		suffix := " (part " + strconv.Itoa(i+1) + ")"
		h := heading + suffix
		if heading == "" {
			h = strings.TrimSpace(suffix)
		}
		out = append(out, Section{Heading: h, Body: c})
	}
	return out
}
