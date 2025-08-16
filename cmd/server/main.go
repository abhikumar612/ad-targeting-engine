package main

import (
	app "ad-targeting-engine/internal/app/server"
	"ad-targeting-engine/internal/config"
)

func main() {
	cfg := config.Load()
	config.SetupLogging(cfg.Server.LogLevel)
	app.Run(cfg)
}
