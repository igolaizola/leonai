package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/igolaizola/leonai"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// Build flags
var version = ""
var commit = ""
var date = ""

func main() {
	// Create signal based context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Launch command
	cmd := newCommand()
	if err := cmd.ParseAndRun(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func newCommand() *ffcli.Command {
	fs := flag.NewFlagSet("leonai", flag.ExitOnError)

	return &ffcli.Command{
		ShortUsage: "leonai [flags] <subcommand>",
		FlagSet:    fs,
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newVersionCommand(),
			newVideoCommand(),
		},
	}
}

func newVersionCommand() *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "leonai version",
		ShortHelp:  "print version",
		Exec: func(ctx context.Context, args []string) error {
			v := version
			if v == "" {
				if buildInfo, ok := debug.ReadBuildInfo(); ok {
					v = buildInfo.Main.Version
				}
			}
			if v == "" {
				v = "dev"
			}
			versionFields := []string{v}
			if commit != "" {
				versionFields = append(versionFields, commit)
			}
			if date != "" {
				versionFields = append(versionFields, date)
			}
			fmt.Println(strings.Join(versionFields, " "))
			return nil
		},
	}
}

func newVideoCommand() *ffcli.Command {
	cmd := "video"
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	_ = fs.String("config", "", "config file (optional)")

	cfg := &leonai.Config{}
	fs.StringVar(&cfg.Cookie, "cookie", "", "cookie file")
	fs.StringVar(&cfg.Proxy, "proxy", "", "proxy")
	fs.DurationVar(&cfg.Wait, "wait", 0, "wait time")
	fs.BoolVar(&cfg.Debug, "debug", false, "debug mode")

	var image string
	fs.StringVar(&image, "image", "", "image to use")
	var motionStrength int
	fs.IntVar(&motionStrength, "motion-strength", 5, "motion strength")
	var output string
	fs.StringVar(&output, "output", "", "output file")

	return &ffcli.Command{
		Name:       cmd,
		ShortUsage: fmt.Sprintf("leonai %s [flags] <key> <value data...>", cmd),
		Options: []ff.Option{
			ff.WithConfigFileFlag("config"),
			ff.WithConfigFileParser(ff.PlainParser),
			ff.WithEnvVarPrefix("LEONAI"),
		},
		ShortHelp: fmt.Sprintf("leonai %s command", cmd),
		FlagSet:   fs,
		Exec: func(ctx context.Context, args []string) error {
			return leonai.GenerateVideo(ctx, cfg, image, motionStrength, output)
		},
	}
}
