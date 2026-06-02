package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	SourceDir    string
	TemporalData string
	OutDir       string
	APIPort      string
}

func Load(path string) Config {
	cfg := Config{SourceDir: "./source-of-truth"}
	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), "\"'")
		switch key {
		case "SOURCE_DIR":
			if val != "" {
				cfg.SourceDir = val
			}
		case "TEMPORAL_DATA":
			cfg.TemporalData = val
		case "OUT_DIR":
			cfg.OutDir = val
		case "API_PORT":
			cfg.APIPort = val
		}
	}
	return cfg
}
