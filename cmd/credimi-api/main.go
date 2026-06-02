package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"credimi-conformance-assessment/internal/assessment"
	"credimi-conformance-assessment/internal/config"
)

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	addr := flag.String("addr", "", "HTTP listen address override")
	envPath := flag.String("env", ".env", "path to .env config")
	flag.Parse()
	cfg := config.Load(*envPath)
	listenAddr := *addr
	if listenAddr == "" {
		if cfg.APIPort == "" {
			fmt.Fprintln(os.Stderr, "credimi-api: API_PORT must be set in .env or --addr must be provided")
			os.Exit(1)
		}
		listenAddr = ":" + cfg.APIPort
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /assessments", func(w http.ResponseWriter, r *http.Request) { handleAssessments(w, r, cfg) })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	log.Printf("credimi-api listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "credimi-api:", err)
		os.Exit(1)
	}
}
func handleAssessments(w http.ResponseWriter, r *http.Request, cfg config.Config) {
	var req assessment.Request
	if r.Body != nil {
		defer r.Body.Close()
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON request body"})
		return
	}
	opts := assessment.ApplyRequest(assessment.Options{SourceDir: cfg.SourceDir, TemporalData: cfg.TemporalData, OutDir: cfg.OutDir}, req)
	res, err := assessment.Generate(opts)
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
