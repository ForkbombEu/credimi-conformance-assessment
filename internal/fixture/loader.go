package fixture

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Fixture struct{ Name, Slug, Dir, ExtractedDir string }

func Slug(name string) string {
	name = strings.ReplaceAll(name, "_", "-")
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(name[i-1])
			if prev >= 'a' && prev <= 'z' {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func List(fixturesDir, extractedDir, selected string) ([]Fixture, error) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return nil, err
	}
	var out []Fixture
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if selected != "" && selected != name && selected != Slug(name) {
			continue
		}
		slug := Slug(name)
		ex := filepath.Join(extractedDir, slug)
		if _, err := os.Stat(ex); err != nil {
			if _, err2 := os.Stat(filepath.Join(extractedDir, name)); err2 == nil {
				ex = filepath.Join(extractedDir, name)
			}
		}
		out = append(out, Fixture{Name: name, Slug: slug, Dir: filepath.Join(fixturesDir, name), ExtractedDir: ex})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	if selected != "" && len(out) == 0 {
		return nil, fmt.Errorf("fixture %q not found", selected)
	}
	return out, nil
}
