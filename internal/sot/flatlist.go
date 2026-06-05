package sot

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type FlatTest struct {
	Number               int
	ID                   string
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
	return ParseFlatListReader(path, f)
}

func ParseFlatListReader(name string, r io.Reader) ([]FlatTest, error) {
	var rows []FlatTest
	seen := map[int]bool{}
	s := bufio.NewScanner(r)
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
		rows = append(rows, FlatTest{Number: n, ID: flatTestID(n, cells[1], cells[2]), Actor: cells[1], Test: cells[2], EvidenceStrength: cells[3], RecommendedExecution: cells[4], SourceReferences: cells[5], Notes: cells[6]})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no flat conformance rows parsed from %s", name)
	}
	return rows, nil
}

func flatTestID(number int, actor string, test string) string {
	return fmt.Sprintf("%s-%03d", flatTestPrefix(actor, test), number)
}

func flatTestPrefix(actor string, test string) string {
	switch strings.ToLower(strings.TrimSpace(actor)) {
	case "wallet":
		return "CR-W"
	case "issuer":
		return "CR-I"
	case "verifier/rp", "verifier/reader":
		return "CR-V"
	case "trust infrastructure":
		return trustInfrastructurePrefix(test)
	case "external/conformance":
		return "CR-CON"
	default:
		return "CR-MIS"
	}
}

func trustInfrastructurePrefix(test string) string {
	normalized := strings.ToLower(test)
	for _, marker := range []string{"lotl", "trusted list", "trust anchor", "certificate", "revocation", "status", "openid federation"} {
		if strings.Contains(normalized, marker) {
			return "CR-TRU"
		}
	}
	return "CR-INF"
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
