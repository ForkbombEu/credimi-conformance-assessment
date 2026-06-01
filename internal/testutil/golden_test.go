package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestGeneratedAssessmentsMatchGoldenSemantics(t *testing.T) {
	repo := filepath.Clean("../..")
	out := t.TempDir()
	cmd := exec.Command("go", "run", "./cmd/credimi-assess", "--source-dir", "./source-of-truth", "--fixtures-dir", "./fixtures", "--extracted-dir", "./out", "--out-dir", out)
	cmd.Dir = repo
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("generator failed: %v\n%s", err, stderr.String())
	}
	want := map[string]int{
		"conformance-assessment-age-verification.md": 1,
		"conformance-assessment-eudi-iss-ver.md":     30,
		"conformance-assessment-eudi-iss2.md":        19,
		"conformance-assessment-eudiw-checks-5x.md":  23,
		"conformance-assessment-multipaz.md":         18,
		"conformance-assessment-talao-iss-cred13.md": 13,
	}
	for name, count := range want {
		goldenBytes, err := os.ReadFile(filepath.Join(repo, "golden-assessments", name))
		if err != nil {
			t.Fatal(err)
		}
		gotBytes, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		golden := parseResults(string(goldenBytes))
		got := parseResults(string(gotBytes))
		if len(got) != count {
			t.Fatalf("%s passed count: got %d want %d", name, len(got), count)
		}
		if !sameIDs(golden, got) {
			t.Fatalf("%s passed IDs got %v want %v", name, keys(got), keys(golden))
		}
		for id, wantText := range golden {
			if got[id] != wantText {
				t.Fatalf("%s test %d text got %q want %q", name, id, got[id], wantText)
			}
		}
		if !strings.Contains(string(gotBytes), "## Passed tests digest") || !strings.Contains(string(gotBytes), "Blank **Test result** cells mean") {
			t.Fatalf("%s missing required report sections", name)
		}
	}
}

func parseResults(md string) map[int]string {
	res := map[int]string{}
	re := regexp.MustCompile(`^\|\s*(\d+)\s*\|`)
	for _, line := range strings.Split(md, "\n") {
		if !re.MatchString(line) {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 4 {
			continue
		}
		id, _ := strconv.Atoi(strings.TrimSpace(cells[0]))
		text := strings.TrimSpace(cells[3])
		if text != "" {
			res[id] = text
		}
	}
	return res
}
func keys(m map[int]string) []int {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}
func sameIDs(a, b map[int]string) bool {
	ak, bk := keys(a), keys(b)
	if len(ak) != len(bk) {
		return false
	}
	for i := range ak {
		if ak[i] != bk[i] {
			return false
		}
	}
	return true
}
