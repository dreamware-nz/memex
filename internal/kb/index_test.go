package kb

import (
	"path/filepath"
	"testing"
)

func TestIndexAndDelete(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"TestIndex_InsertsChunks", testIndexInsertsChunks},
		{"TestIndex_Upsert", testIndexUpsert},
		{"TestIndex_FTSSync", testIndexFTSSync},
		{"TestIndex_FTSTriggerSmoke", testFTSTriggerSmoke},
		{"TestIndex_EmptyContentErrors", testIndexEmptyContent},
		{"TestDelete_CascadesSection", testDeleteCascades},
		{"TestDelete_UnknownLabelNoOp", testDeleteUnknown},
		{"TestIndexPlainText_RoundTrip", testIndexPlainTextRoundTrip},
		{"TestIndexPlainText_Upsert", testIndexPlainTextUpsert},
		{"TestIndexPlainText_TotalChunksMatchesRows", testIndexPlainTextTotalChunks},
		{"TestIndexPlainText_EmptyContent", testIndexPlainTextEmpty},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testIndexPlainTextRoundTrip(t *testing.T) {
	s := newStore(t)

	res, err := s.IndexPlainText("alpha beta gamma\n\ndelta epsilon", "execute:shell")
	if err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	if res.SourceID <= 0 {
		t.Fatalf("SourceID = %d, want > 0", res.SourceID)
	}
	if res.Label != "execute:shell" {
		t.Fatalf("Label = %q, want execute:shell", res.Label)
	}
	if res.TotalChunks <= 0 {
		t.Fatalf("TotalChunks = %d, want > 0", res.TotalChunks)
	}

	var dbID int64
	if err := s.db.QueryRow(`SELECT id FROM sources WHERE label = ?`, "execute:shell").Scan(&dbID); err != nil {
		t.Fatalf("source row: %v", err)
	}
	if dbID != res.SourceID {
		t.Fatalf("SourceID = %d, db id = %d", res.SourceID, dbID)
	}

	var kind string
	if err := s.db.QueryRow(`SELECT kind FROM sources WHERE id = ?`, res.SourceID).Scan(&kind); err != nil {
		t.Fatalf("source kind: %v", err)
	}
	if kind != "plaintext" {
		t.Fatalf("kind = %q, want plaintext", kind)
	}
}

func testIndexPlainTextUpsert(t *testing.T) {
	s := newStore(t)

	r1, err := s.IndexPlainText("first content\n\nmore", "execute:shell")
	if err != nil {
		t.Fatalf("first IndexPlainText: %v", err)
	}
	r2, err := s.IndexPlainText("totally different stuff", "execute:shell")
	if err != nil {
		t.Fatalf("second IndexPlainText: %v", err)
	}

	// SourceID can change because we DELETE and re-INSERT, but label is single
	// and only the most recent content is searchable.
	_ = r1
	if r2.Label != "execute:shell" {
		t.Fatalf("Label = %q", r2.Label)
	}

	var sources int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources WHERE label = ?`, "execute:shell").Scan(&sources); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if sources != 1 {
		t.Fatalf("sources = %d, want 1", sources)
	}

	var firstHits int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sections_fts WHERE sections_fts MATCH ?`, "first").Scan(&firstHits); err != nil {
		t.Fatalf("fts first: %v", err)
	}
	if firstHits != 0 {
		t.Fatalf("first-content fts hits = %d, want 0 after upsert", firstHits)
	}
}

func testIndexPlainTextTotalChunks(t *testing.T) {
	s := newStore(t)

	const content = "para one\n\npara two\n\npara three"
	res, err := s.IndexPlainText(content, "tc")
	if err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	var sectionRows int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections WHERE source_id = ?`, res.SourceID,
	).Scan(&sectionRows); err != nil {
		t.Fatalf("count sections: %v", err)
	}
	if sectionRows != res.TotalChunks {
		t.Fatalf("TotalChunks = %d, sections rows = %d", res.TotalChunks, sectionRows)
	}
}

func testIndexPlainTextEmpty(t *testing.T) {
	s := newStore(t)

	if _, err := s.IndexPlainText("", "label"); err == nil {
		t.Fatalf("IndexPlainText empty: want error, got nil")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources`).Scan(&n); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if n != 0 {
		t.Fatalf("sources = %d, want 0", n)
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "kb.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func testIndexInsertsChunks(t *testing.T) {
	s := newStore(t)

	const doc = "# Top\n\nintro body\n\n## Sub A\n\nbody A text\n\n## Sub B\n\nbody B text\n"
	if err := s.Index("doc-1", doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	var sources int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources WHERE label = ?`, "doc-1").Scan(&sources); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if sources != 1 {
		t.Fatalf("sources = %d, want 1", sources)
	}

	var sections int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections WHERE source_id = (SELECT id FROM sources WHERE label = ?)`,
		"doc-1",
	).Scan(&sections); err != nil {
		t.Fatalf("count sections: %v", err)
	}
	if sections != 3 {
		t.Fatalf("sections = %d, want 3", sections)
	}

	rows, err := s.db.Query(
		`SELECT heading FROM sections WHERE source_id = (SELECT id FROM sources WHERE label = ?) ORDER BY id`,
		"doc-1",
	)
	if err != nil {
		t.Fatalf("query headings: %v", err)
	}
	defer rows.Close()
	want := []string{"Top", "Top > Sub A", "Top > Sub B"}
	got := []string{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, h)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("heading[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func testIndexUpsert(t *testing.T) {
	s := newStore(t)

	first := "# A\n\nfirst pass body\n\n## A2\n\nmore\n"
	second := "# B\n\nonly one section now\n"
	if err := s.Index("label-x", first); err != nil {
		t.Fatalf("first Index: %v", err)
	}
	if err := s.Index("label-x", second); err != nil {
		t.Fatalf("second Index: %v", err)
	}

	var sections int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections WHERE source_id = (SELECT id FROM sources WHERE label = ?)`,
		"label-x",
	).Scan(&sections); err != nil {
		t.Fatalf("count sections: %v", err)
	}
	if sections != 1 {
		t.Fatalf("sections after re-index = %d, want 1", sections)
	}

	var sources int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources WHERE label = ?`, "label-x").Scan(&sources); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if sources != 1 {
		t.Fatalf("sources = %d, want 1", sources)
	}
}

func testIndexFTSSync(t *testing.T) {
	s := newStore(t)

	if err := s.Index("doc-fts", "# Heading\n\nzqxlepton appears here once\n"); err != nil {
		t.Fatalf("Index: %v", err)
	}

	var hits int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections_fts WHERE sections_fts MATCH ?`,
		"zqxlepton",
	).Scan(&hits); err != nil {
		t.Fatalf("FTS query: %v", err)
	}
	if hits < 1 {
		t.Fatalf("fts hits for unique token = %d, want >=1", hits)
	}

	if err := s.Index("doc-fts", "# Heading\n\nreplacement token: yzwflarp\n"); err != nil {
		t.Fatalf("re-Index: %v", err)
	}

	var oldHits int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections_fts WHERE sections_fts MATCH ?`,
		"zqxlepton",
	).Scan(&oldHits); err != nil {
		t.Fatalf("FTS query old: %v", err)
	}
	if oldHits != 0 {
		t.Fatalf("old token hits = %d, want 0", oldHits)
	}

	var newHits int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections_fts WHERE sections_fts MATCH ?`,
		"yzwflarp",
	).Scan(&newHits); err != nil {
		t.Fatalf("FTS query new: %v", err)
	}
	if newHits < 1 {
		t.Fatalf("new token hits = %d, want >=1", newHits)
	}
}

func testFTSTriggerSmoke(t *testing.T) {
	s := newStore(t)

	res, err := s.db.Exec(
		`INSERT INTO sources (label, kind, created_at) VALUES (?, 'markdown', 0)`,
		"trigger-smoke",
	)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	srcID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO sections (source_id, heading, body, byte_size, created_at) VALUES (?, ?, ?, ?, 0)`,
		srcID, "h", "qzxctrigger payload", len("qzxctrigger payload"),
	); err != nil {
		t.Fatalf("insert section: %v", err)
	}

	var hits int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sections_fts WHERE sections_fts MATCH ?`,
		"qzxctrigger",
	).Scan(&hits); err != nil {
		t.Fatalf("FTS smoke query: %v", err)
	}
	if hits != 1 {
		t.Fatalf("trigger-driven fts hits = %d, want 1", hits)
	}
}

func testIndexEmptyContent(t *testing.T) {
	s := newStore(t)
	if err := s.Index("empty", ""); err == nil {
		t.Fatalf("Index empty: want error, got nil")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources`).Scan(&n); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if n != 0 {
		t.Fatalf("sources = %d, want 0", n)
	}
}

func testDeleteCascades(t *testing.T) {
	s := newStore(t)
	if err := s.Index("gone", "# H\n\nbody\n"); err != nil {
		t.Fatalf("Index: %v", err)
	}
	if err := s.Delete("gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var srcs int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources WHERE label = ?`, "gone").Scan(&srcs); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if srcs != 0 {
		t.Fatalf("sources = %d, want 0", srcs)
	}
	var secs int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sections`).Scan(&secs); err != nil {
		t.Fatalf("count sections: %v", err)
	}
	if secs != 0 {
		t.Fatalf("sections = %d, want 0", secs)
	}
}

func testDeleteUnknown(t *testing.T) {
	s := newStore(t)
	if err := s.Delete("never-indexed"); err != nil {
		t.Fatalf("Delete unknown: %v", err)
	}
}
