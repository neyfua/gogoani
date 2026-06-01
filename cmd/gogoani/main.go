package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/neyfua/gogoani/internal/anilist"
	"github.com/neyfua/gogoani/internal/config"
	"github.com/neyfua/gogoani/internal/logger"
	"github.com/neyfua/gogoani/internal/ui"
)

func main() {
	var dubFlag bool
	flag.BoolVar(&dubFlag, "d", false, "play dubbed version")
	flag.BoolVar(&dubFlag, "dub", false, "play dubbed version")
	debugFlag := flag.Bool("debug", false, "enable debug logging")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gogoani [option]\n")
		fmt.Fprintf(flag.CommandLine.Output(), "       gogoani [option] \"anime name\"\n")
		fmt.Fprintf(flag.CommandLine.Output(), "       gogoani anilist [--auth|--sync|--status [watching|completed|paused|dropped]]\n\n")
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

	args := flag.Args()
	if len(args) > 0 && args[0] == "anilist" {
		if err := runAniList(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	query := strings.Join(args, " ")
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

func runAniList(args []string) error {
	// Manually handle --status so it works with or without a filter value
	var statusFilter string
	var statusSeen bool
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" {
			statusSeen = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				statusFilter = args[i+1]
				i++ // skip next arg (the filter value)
			}
			continue
		}
		if strings.HasPrefix(args[i], "--status=") {
			statusSeen = true
			statusFilter = args[i][len("--status="):]
			continue
		}
		remaining = append(remaining, args[i])
	}

	fs := flag.NewFlagSet("anilist", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	authFlag := fs.Bool("auth", false, "authenticate with AniList token")
	syncFlag := fs.Bool("sync", false, "sync AniList anime list to cache")

	if err := fs.Parse(remaining); err != nil {
		fmt.Fprintln(os.Stderr, "Usage: gogoani anilist [--auth|--sync|--status <status>]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(os.Stderr, "  --%-14s %s\n", f.Name, f.Usage)
		})
		fmt.Fprintln(os.Stderr, "  --status        filter by status: watching|completed|paused|dropped")
		return nil
	}

	if *authFlag {
		viewer, err := anilist.PromptAuth()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Authenticated as %s\n", viewer.Name)
		return nil
	}

	if *syncFlag {
		token, err := anilist.LoadToken()
		if err != nil {
			return err
		}
		client := anilist.NewClient(token)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		entries, err := client.SyncList(ctx)
		if err != nil {
			return err
		}
		path, err := anilist.ListPath()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Synced %d anime to %s\n", len(entries), path)
		return nil
	}

	if statusSeen {
		entries, err := anilist.LoadList()
		if err != nil {
			return err
		}
		cfg := config.Load()
		return anilist.ShowStatusList(entries, statusFilter, func(title string) error {
			return ui.PlayAnimeByTitle(cfg, title, "sub")
		})
	}

	return fmt.Errorf("usage: gogoani anilist [--auth|--sync|--status [watching|completed|paused|dropped]]")
}
