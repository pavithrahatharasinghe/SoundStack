package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"soundstack/internal/app"
	"soundstack/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		handleInit()
	case "doctor":
		withApp(func(ctx context.Context, a *app.App) error {
			return a.Doctor(ctx)
		})
	case "sync":
		handleSync()
	case "run":
		handleRun()
	case "status":
		withApp(func(ctx context.Context, a *app.App) error {
			counts, err := a.JobCounts(ctx)
			if err != nil {
				return err
			}
			for k, v := range counts {
				fmt.Printf("%s: %d\n", k, v)
			}
			return nil
		})
	case "retry":
		withApp(func(ctx context.Context, a *app.App) error {
			n, err := a.RetryFailed(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("Requeued %d jobs\n", n)
			return nil
		})
	case "scan":
		withApp(func(ctx context.Context, a *app.App) error {
			return a.Run(ctx, true)
		})
	default:
		usage()
	}
}

func usage() {
	fmt.Println("soundstack commands:")
	fmt.Println("  init")
	fmt.Println("  doctor")
	fmt.Println("  sync playlist --id <playlist_id> [--watch]")
	fmt.Println("  run [--once]")
	fmt.Println("  status")
	fmt.Println("  retry --failed")
	fmt.Println("  scan")
}

func configPath() string {
	if v := os.Getenv("SOUNDSTACK_CONFIG"); v != "" {
		return v
	}
	return "./configs/config.yaml"
}

func loadConfigOrExit() *config.Config {
	cfg, err := config.Load(configPath())
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	return cfg
}

func handleInit() {
	cfgPath := configPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := config.WriteExample(cfgPath); err != nil {
			log.Fatalf("write example config: %v", err)
		}
		fmt.Printf("Wrote example config to %s\n", cfgPath)
	}
	cfg := loadConfigOrExit()
	app, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer app.Close()
	if err := app.InitFilesystem(); err != nil {
		log.Fatalf("init filesystem: %v", err)
	}
	fmt.Println("Initialization complete.")
}

func handleSync() {
	if len(os.Args) < 3 || os.Args[2] != "playlist" {
		usage()
		return
	}
	fs := flag.NewFlagSet("playlist", flag.ExitOnError)
	id := fs.String("id", "", "spotify playlist id")
	watch := fs.Bool("watch", false, "watch for changes")
	_ = fs.Parse(os.Args[3:])
	if *id == "" {
		log.Fatal("--id is required")
	}
	withApp(func(ctx context.Context, a *app.App) error {
		return a.SyncPlaylist(ctx, *id, *watch)
	})
}

func handleRun() {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	once := fs.Bool("once", false, "exit when queue is empty")
	_ = fs.Parse(os.Args[2:])
	withApp(func(ctx context.Context, a *app.App) error {
		return a.Run(ctx, *once)
	})
}

func withApp(fn func(ctx context.Context, a *app.App) error) {
	cfg := loadConfigOrExit()
	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}
	defer a.Close()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := fn(ctx, a); err != nil {
		log.Fatal(err)
	}
}
