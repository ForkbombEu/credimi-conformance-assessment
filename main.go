package main

import (
	"fmt"
	"os"
)

const asciiArt = `
  ____ ____  _____ ____ ___ __  __ ___
 / ___|  _ \| ____|  _ \_ _|  \/  |_ _|
| |   | |_) |  _| | | | | || |\/| || |
| |___|  _ <| |___| |_| | || |  | || |
 \____|_| \_\_____|____/___|_|  |_|___|

  CONFORMANCE ASSESSMENT
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(os.Stderr, asciiArt)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/credimi-assess [flags]")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/credimi-api [flags]")
		return nil
	}
	return fmt.Errorf("unknown command: %s", args[0])
}
