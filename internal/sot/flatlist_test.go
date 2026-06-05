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
	if rows[0].Number != 1 || rows[0].ID != "CR-W-001" || rows[0].Actor == "" || rows[0].SourceReferences == "" {
		t.Fatalf("unexpected first row: %#v", rows[0])
	}
	wants := map[int]string{
		1:   "CR-W-001",
		65:  "CR-I-065",
		104: "CR-V-104",
		137: "CR-TRU-137",
		150: "CR-INF-150",
		159: "CR-CON-159",
	}
	for _, row := range rows {
		if want, ok := wants[row.Number]; ok {
			if row.ID != want {
				t.Fatalf("row %d id got %q want %q", row.Number, row.ID, want)
			}
			delete(wants, row.Number)
		}
	}
	if len(wants) > 0 {
		t.Fatalf("missing expected rows: %#v", wants)
	}
}
