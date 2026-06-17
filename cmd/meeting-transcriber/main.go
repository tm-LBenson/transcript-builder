package main

import (
	"os"

	"github.com/tm-LBenson/transcript-builder/internal/app"
)

var (
	version = "0.1.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	os.Exit(app.Main(os.Args, app.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}))
}
