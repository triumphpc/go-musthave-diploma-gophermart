package withdrawal

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"time"
)

// Run withdrawal handler
func Run(ctx context.Context, lgr *zap.Logger, stg storage.Storage) error {
	lgr.Info("Run withdrawal handler")
	defer lgr.Info("Out withdrawal handler")

	for {
		select {
		// How ofter check withdrawals
		case <-time.After(time.Second):
			wds, err := stg.ActiveWithdrawals(ctx)
			if err != nil {
				lgr.Error("Get withdrawals error", zap.Error(err))
				continue
			}

			if len(wds) == 0 {
				continue
			}

			for _, wd := range wds {
				// Some abstract implement for confirm withdraw
				// It's a sample only

				ord := models.Order{
					UserID: wd.UserID,
					ID:     wd.OrderID,
				}
				lgr.Info("Withdraw process", zap.Reflect("order", ord), zap.Reflect("sum", wd.Sum))
				if err := stg.Withdraw(ctx, ord, wd.Sum); err != nil {
					lgr.Error("Error withdraw", zap.Error(err))
					return ctx.Err()
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
