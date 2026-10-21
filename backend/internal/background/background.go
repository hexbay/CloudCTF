package background

import (
	"context"
	"log/slog"
	"time"

	"cloudctf/backend/internal/services"
)

func Start(ctx context.Context, logger *slog.Logger, containerGroups *services.ContainerGroupService) {
	if containerGroups != nil {
		startExpiredContainerCleanupLoop(ctx, logger, containerGroups)
	}
}

func startExpiredContainerCleanupLoop(ctx context.Context, logger *slog.Logger, containerGroups *services.ContainerGroupService) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				containerGroups.CleanupExpiredContainerGroups()
				if logger != nil {
					logger.Debug("expired container cleanup tick completed")
				}
			}
		}
	}()
}
