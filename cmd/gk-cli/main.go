package main

// GophKeeper CLI — small interactive client for auth and vault operations.

import (
	"log"
	"os"

	"github.com/antonminaichev/gophkeeper/internal/client"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli := client.NewCLI(version, commit, date)
	rootCmd := cli.CreateCommands()

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
