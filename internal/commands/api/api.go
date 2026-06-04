package api

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"credimi-conformance-assessment/internal/cli"
	"credimi-conformance-assessment/internal/config"
	"credimi-conformance-assessment/pkg/conformance"
)

type errorResponse struct {
	Error string `json:"error"`
}

// Run executes the api CLI command.
func Run(args []string) error {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	addr := fs.String("addr", "", "HTTP listen address override")
	envPath := fs.String("env", ".env", "path to .env config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := config.Load(*envPath)
	listenAddr := *addr
	if listenAddr == "" {
		if cfg.APIPort == "" {
			return fmt.Errorf("credimi-api: API_PORT must be set in .env or --addr must be provided")
		}
		listenAddr = ":" + cfg.APIPort
	}

	fmt.Fprint(os.Stderr, cli.ASCIIArt)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /assessments", func(w http.ResponseWriter, r *http.Request) {
		handleAssessments(w, r, cfg)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	log.Printf("credimi-api listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		return fmt.Errorf("credimi-api: %w", err)
	}
	return nil
}

func handleAssessments(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	var req conformance.ReportInput
	if r.Body != nil {
		defer r.Body.Close()
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON request body"})
		return
	}

	res, err := conformance.Generate(req, conformance.ReportOptions{SourceDir: cfg.SourceDir, OutDir: cfg.OutDir})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
