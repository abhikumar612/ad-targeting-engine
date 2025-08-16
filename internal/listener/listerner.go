package listener

import (
	"context"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"

	"ad-targeting-engine/internal/engine"
	"ad-targeting-engine/internal/storage"
)

func ListenAndRefresh(ctx context.Context, st *storage.Store, eng *engine.DeliveryEngine, channel string, baseBackoff time.Duration) {
	conn, err := st.PgxPool().Acquire(ctx)
	if err != nil {
		log.Error().Err(err).Msg("acquire conn for listen")
		return
	}
	defer conn.Release()

	if channel == "" {
		channel = st.ListenChannel()
	}
	if _, err = conn.Exec(ctx, "LISTEN "+channel); err != nil {
		log.Error().Err(err).Str("channel", channel).Msg("listen")
		return
	}
	log.Info().Str("channel", channel).Msg("listening for DB changes")

	var lastRefresh time.Time
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("listener stopped")
			return
		default:
			ntf, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				backoff := jitter(baseBackoff)
				log.Error().Err(err).Dur("retry_in", backoff).Msg("notify wait error")
				time.Sleep(backoff)
				continue
			}
			if time.Since(lastRefresh) < 200*time.Millisecond {
				continue // debounce burst of notifications
			}
			lastRefresh = time.Now()
			log.Info().Str("channel", ntf.Channel).Msg("db change; refreshing snapshot")
			if err := eng.BuildSnapshot(ctx, st); err != nil {
				log.Error().Err(err).Msg("refresh snapshot error")
			}
		}
	}
}

func jitter(base time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	factor := 0.5 + rand.Float64() // 0.5x–1.5x
	return time.Duration(float64(base) * factor)
}
