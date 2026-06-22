package jwt_repository_postgres

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type CleanupLogger interface {
	Debug(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// Запуск фоновой очистки просроченных refresh токенов
func (r *JwtRepository) StartCleanup(ctx context.Context, interval time.Duration, log CleanupLogger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Debug("jwt cleanup: started", zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			log.Debug("jwt cleanup: stopped")
			return
		case <-ticker.C:
			if err := r.deleteExpiredTokens(ctx); err != nil {
				log.Error("jwt cleanup: failed to delete expired tokens", zap.Error(err))
			}
		}
	}
}

func (r *JwtRepository) deleteExpiredTokens(ctx context.Context) error {
	cleanupCtx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `DELETE FROM gochat.refresh_tokens WHERE expires_at < NOW();`
	tag, err := r.pool.Exec(cleanupCtx, query)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}

	if tag.RowsAffected() > 0 {
		//Лог только если что-то удалили
		_ = tag.RowsAffected()
	}
	return nil
}
