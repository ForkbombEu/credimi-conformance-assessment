package sot

import "testing"

func TestParseFlatList(t *testing.T) {
	rows, err := ParseFlatList("../../source-of-truth/credimi-flat-conformance-test-list-v1.1.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 150 {
		t.Fatalf("expected at least 150 rows, got %d", len(rows))
	}
	if rows[0].Number != 1 || rows[0].Actor == "" || rows[0].SourceReferences == "" {
		t.Fatalf("unexpected first row: %#v", rows[0])
	}
}
