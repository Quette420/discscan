// Command discscan finds what is eating your disk space: the largest files,
// large files that appeared recently and the largest folders.
package main

import (
	"os"

	"github.com/gsaranov/discscan/internal/cli"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	os.Exit(cli.Run(version, os.Args[1:]))
}
