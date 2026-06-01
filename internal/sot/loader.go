package sot

import (
	"os"
	"path/filepath"

	"credimi-conformance-assessment/internal/rules"
)

type Source struct {
	FlatTests []FlatTest
	Taxonomy  rules.Taxonomy
}

func Load(dir string) (*Source, error) {
	flat, err := ParseFlatList(filepath.Join(dir, "credimi-flat-conformance-test-list-v1.1.md"))
	if err != nil {
		return nil, err
	}
	taxBytes, err := os.ReadFile(filepath.Join(dir, "credimi-conformance-aggregation-taxonomy-v1.1.yaml"))
	if err != nil {
		return nil, err
	}
	tax := rules.ParseTaxonomy(taxBytes)
	return &Source{FlatTests: flat, Taxonomy: tax}, nil
}
