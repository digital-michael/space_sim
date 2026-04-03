package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/app"
)

func main() {
	appConfigPath := app.DefaultAppConfigPath
	appConfig, err := app.LoadAppConfig(appConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading app config %s: %v\n", appConfigPath, err)
		os.Exit(1)
	}

	performanceMode := flag.Bool("performance", false, "Run automated performance testing")
	profileFlag := flag.String("profile", "", "Camera profile for performance testing: 'worst' (overview) or 'better' (tracking Jupiter from belt)")
	threadsFlag := flag.Int("threads", 0, "Number of physics worker threads (1-25)")
	noLockingFlag := flag.Bool("no-locking", false, "Disable double-buffer locking (unsafe, for performance testing only)")
	systemConfigFlag := flag.String("system-config", "", "Path to JSON system configuration file")
	debugFlag := flag.Bool("debug", false, "Enable verbose debug logging and smoke debug instrumentation")
	flag.Parse()

	profileProvided := false
	threadsProvided := false
	noLockingProvided := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "profile":
			profileProvided = true
		case "threads":
			threadsProvided = true
		case "no-locking":
			noLockingProvided = true
		}
	})

	if (profileProvided || threadsProvided || noLockingProvided) && !*performanceMode {
		fmt.Println("Error: --profile, --threads, and --no-locking flags can only be used with --performance")
		os.Exit(1)
	}

	cfg := app.Config{
		PerformanceMode: *performanceMode,
		Profile:         *profileFlag,
		Threads:         *threadsFlag,
		NoLocking:       *noLockingFlag,
		SystemConfig:    *systemConfigFlag,
		Debug:           *debugFlag,
		AppConfigPath:   appConfigPath,
		AppConfig:       appConfig,
	}

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
