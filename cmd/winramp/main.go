package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/winramp/winramp/internal/config"
	"github.com/winramp/winramp/internal/infrastructure/db"
	"github.com/winramp/winramp/internal/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		logLevel   = flag.String("log-level", "", "Log level (debug, info, warn, error)")
		version    = flag.Bool("version", false, "Show version information")
		migrate    = flag.String("migrate", "", "Run database migrations (up/down)")
		backup     = flag.String("backup", "", "Backup database to specified path")
		restore    = flag.String("restore", "", "Restore database from specified path")
	)
	flag.Parse()

	// Show version and exit
	if *version {
		fmt.Printf("WinRamp %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Initialize configuration
	cfg := config.Get()
	if *configPath != "" {
		// Load custom config file
		fmt.Printf("Loading configuration from: %s\n", *configPath)
	}

	// Initialize logger
	logConfig := logger.DefaultConfig()
	if *logLevel != "" {
		logConfig.Level = *logLevel
	} else if cfg.Advanced.LogLevel != "" {
		logConfig.Level = cfg.Advanced.LogLevel
	}
	logConfig.FilePath = cfg.App.LogDir + "/winramp.log"
	logger.Initialize(logConfig)

	// Log startup
	logger.Info("WinRamp starting",
		logger.String("version", Version),
		logger.String("build_time", BuildTime),
	)

	// Initialize database
	dbConfig := db.DefaultConfig()
	dbConfig.Path = cfg.Library.DatabasePath
	if err := db.Initialize(dbConfig); err != nil {
		logger.Fatal("Failed to initialize database", logger.Error(err))
	}
	defer db.Get().Close()

	// Handle database operations
	if *migrate != "" {
		handleMigration(*migrate)
		os.Exit(0)
	}

	if *backup != "" {
		if err := db.Get().Backup(*backup); err != nil {
			logger.Fatal("Failed to backup database", logger.Error(err))
		}
		logger.Info("Database backed up successfully", logger.String("path", *backup))
		os.Exit(0)
	}

	if *restore != "" {
		if err := db.Get().Restore(*restore); err != nil {
			logger.Fatal("Failed to restore database", logger.Error(err))
		}
		logger.Info("Database restored successfully", logger.String("path", *restore))
		os.Exit(0)
	}

	// Create application instance
	app := NewApp()

	// Create Wails application with options
	err := wails.Run(&options.App{
		Title:     "WinRamp",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		logger.Fatal("Failed to run application", logger.Error(err))
	}
}

func handleMigration(direction string) {
	switch direction {
	case "up":
		if err := db.Get().Migrate(); err != nil {
			logger.Fatal("Failed to run migrations", logger.Error(err))
		}
		logger.Info("Migrations completed successfully")
	case "down":
		logger.Error("Migration rollback not yet implemented")
	default:
		logger.Fatal("Invalid migration direction. Use 'up' or 'down'")
	}
}

