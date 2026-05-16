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
	const appName = "space-sim"
	appConfigPath := app.DefaultAppConfigPathFor(appName)
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
	noTexturesFlag := flag.Bool("no-textures", false, "disable diffuse texture rendering; use solid colors (enabled by default)")
	noLightingFlag := flag.Bool("no-lighting", false, "disable Phong star lighting shader; use flat diffuse rendering")
	noMSAAFlag := flag.Bool("no-msaa", false, "disable 4× MSAA anti-aliasing (enabled by default)")
	simSpeedFlag := flag.Float64("sim-speed", 3600, "simulated seconds per real second (1=real-time, 3600=1 sim-hour/sec, 86400=1 sim-day/sec, 604800=1 sim-week/sec)")
	keybindingsFlag := flag.String("keybindings", "", "path to a custom keybindings JSON file (default: configs/keybindings.json)")
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
		NoTextures:      *noTexturesFlag,
		NoLighting:      *noLightingFlag,
		NoMSAA:          *noMSAAFlag,
		SimTimeScale:    *simSpeedFlag,
		AppConfigPath:   appConfigPath,
		AppConfig:       appConfig,
		KeybindingsPath: *keybindingsFlag,
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
