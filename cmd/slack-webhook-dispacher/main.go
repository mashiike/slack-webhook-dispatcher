package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/fujiwara/ridge"
	slackwebhookdispatcher "github.com/mashiike/slack-webhook-dispatcher"
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	var (
		debug      bool
		configPath string
		port       int
	)
	flag.StringVar(&configPath, "config", "config.jsonnet", "config file path")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.IntVar(&port, "port", 8282, "port number")
	flag.VisitAll(func(f *flag.Flag) {
		envName := strings.ToUpper(strings.ReplaceAll(f.Name, "_", "-"))
		if v, ok := os.LookupEnv(envName); ok {
			f.Value.Set(v)
		}
	})
	flag.Parse()
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(
			os.Stderr,
			&slog.HandlerOptions{
				Level: level,
			},
		)),
	)
	config, err := slackwebhookdispatcher.LoadConfig(ctx, configPath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to load config", "details", err.Error())
		return 1
	}
	h := slackwebhookdispatcher.New(config)
	ridge.RunWithContext(ctx, fmt.Sprintf(":%d", port), "/", h)
	return 0
}
