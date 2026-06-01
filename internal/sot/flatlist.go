package sot

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FlatTest struct {
	Number               int
	Actor                string
	Test                 string
	EvidenceStrength     string
	RecommendedExecution string
	SourceReferences     string
	Notes                string
}

func ParseFlatList(path string) ([]FlatTest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []FlatTest
	seen := map[int]bool{}
	s := bufio.NewScanner(f)
	// allow long source refs
	s.Buffer(make([]byte, 1024), 1024*1024*4)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) < 7 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(cells[0]))
		if err != nil {
			continue
		}
		if seen[n] {
			return nil, fmt.Errorf("duplicate flat test number %d", n)
		}
		seen[n] = true
		rows = append(rows, FlatTest{Number: n, Actor: cells[1], Test: cells[2], EvidenceStrength: cells[3], RecommendedExecution: cells[4], SourceReferences: cells[5], Notes: cells[6]})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no flat conformance rows parsed from %s", path)
	}
	return rows, nil
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}
