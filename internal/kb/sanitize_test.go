package kb

import "testing"

func TestSanitizeFTS5Query(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single space", " ", " "},
		{"only whitespace", "   \t\n", "   \t\n"},
		{"single dash", "-", "-"},
		{"double dash prefix", "--foo", "--foo"},
		{"plain word", "foo", "foo"},
		{"plain phrase", "hello world", "hello world"},
		{"hyphenated id", "foo-bar", `"foo-bar"`},
		{"hyphenated triple", "polyglot-sandbox-77", `"polyglot-sandbox-77"`},
		{"hyphenated then plain", "foo-bar baz", `"foo-bar" baz`},
		{"plain then hyphenated", "baz foo-bar", `baz "foo-bar"`},
		{"mixed two hyphenated", "a-b c-d", `"a-b" "c-d"`},
		{"intentional negation", "foo -bar", "foo NOT bar"},
		{"negation alongside hyphenated", "foo-bar -baz", `"foo-bar" NOT baz`},
		{"already quoted phrase", `"already-quoted phrase"`, `"already-quoted phrase"`},
		{"already quoted with bare token", `"already-quoted" loose-token`, `"already-quoted" "loose-token"`},
		{"OR operator preserved", "foo-bar OR baz", `"foo-bar" OR baz`},
		{"trailing hyphen", "foo-", `"foo-"`},
		{"multiple internal hyphens", "a-b-c-d-e", `"a-b-c-d-e"`},
		{"interior tabs", "a-b\tc-d", "\"a-b\"\t\"c-d\""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFTS5Query(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeFTS5Query(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
