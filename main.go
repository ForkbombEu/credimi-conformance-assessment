package main

import (
	"fmt"
	"os"

	"credimi-conformance-assessment/internal/cli"
	"credimi-conformance-assessment/internal/commands/api"
	"credimi-conformance-assessment/internal/commands/assess"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		printUsage()
		return fmt.Errorf("no command provided")
	}

	switch args[0] {
	case "assess":
		return assess.Run(args[1:])
	case "api":
		return api.Run(args[1:])
	case "version", "--version":
		fmt.Println(version)
		return nil
	case "help", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, cli.ASCIIArt)
	fmt.Fprintln(os.Stderr, "Usage: credimi-conformance-assessment <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  assess   Generate conformance assessment reports")
	fmt.Fprintln(os.Stderr, "  api      Serve the assessment HTTP API")
	fmt.Fprintln(os.Stderr, "  version  Print version")
}
