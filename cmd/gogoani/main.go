package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/neyfua/gogoani/internal/config"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/ui"
)

func main() {
	var dubFlag bool
	flag.BoolVar(&dubFlag, "d", false, "play dubbed version")
	flag.BoolVar(&dubFlag, "dub", false, "play dubbed version")
	debugFlag := flag.Bool("debug", false, "enable debug logging")

	flag.Usage = func () {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gogoani [option]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "       gogoani [option] \"anime name\"\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -d, --dub       play dubbed version\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --debug         enable debug logging\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -h, --help      print out help commands\n")
	}

	flag.Parse()

	if err := logger.Init(*debugFlag); err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize logger:", err)
		os.Exit(1)
	}

	query := strings.Join(flag.Args(), " ")
	cfg := config.Load()

	logger.Log.Debug("starting gogoani", "query", query, "player", cfg.Player)

	mode := "sub"
	if dubFlag {
		mode = "dub"
	}

	if err := ui.Run(cfg, query, mode); err != nil {
		logger.Log.Error("application error", "error", err)
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
