package main

import (
	"context"

	"ad-targeting-engine/internal/app/server"
	"ad-targeting-engine/internal/config"
	"ad-targeting-engine/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	store, err := storage.New(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	cache := storage.NewCache()

	// warmup cache
	campaigns, _ := store.LoadActiveCampaigns(ctx)
	cache.UpdateCampaigns(campaigns)

	srv := server.New(store, cache)
	srv.StartCacheRefresher(ctx)
	srv.Start(cfg.Server.Addr)
}
