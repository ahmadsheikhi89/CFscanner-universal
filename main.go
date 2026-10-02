package main

import (
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Program Info
var (
	version  = "v" + time.Now().Format("2006.01.02")
	build    = "Custom"
	codename = "CFScanner , CloudFlare Scanner."
)

func Version() string {
	return version
}

// VersionStatement returns a list of strings representing the full version info.
func VersionStatement() string {
	return strings.Join([]string{
		"CFScanner ", Version(), " (", codename, ") ", build, " (", runtime.Version(), " ", runtime.GOOS, "/", runtime.GOARCH, ")",
	}, "")
}

func main() {
	rootCmd := run()

	RegisterCommands(rootCmd)

	// Disable go's internal http2 error spam by redirecting standard log output
	// if we are entering the TUI or general scanning phase.
	log.SetOutput(io.Discard)

	if len(os.Args) <= 1 {
		err := rootCmd.Help()
		if err != nil {
			return
		}
		os.Exit(1)
	}

	err := rootCmd.Execute()

	cobra.CheckErr(err)
}
