package maxpoll

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"section-near-home/backend/internal/clients/maxapi"
	"section-near-home/backend/internal/dialog"
	"section-near-home/backend/internal/logging"
)

const pollErrorBackoff = 2 * time.Second

const pollPanicBackoff = 5 * time.Second

const maxConsecutivePollPanics = 3

var errPollPanic = errors.New("panic in poll request")

type Poller struct {
	client  *maxapi.Client
	machine *dialog.Machine
	logger  *slog.Logger
}

func NewPoller(
	client *maxapi.Client,
	machine *dialog.Machine,
	logger *slog.Logger,
) *Poller {
	return &Poller{
		client:  client,
		machine: machine,
		logger:  logging.WithComponent(logger, "maxpoll"),
	}
}

func (poller *Poller) Run(ctx context.Context) error {
	poller.logger.InfoContext(ctx, "poller started",
		"poll_error_backoff", pollErrorBackoff.String(),
		"poll_panic_backoff", pollPanicBackoff.String(),
		"max_consecutive_panics", maxConsecutivePollPanics)
	defer poller.logger.Info("poller stopped")

	var marker int64
	var consecutivePanics int

	for {
		select {
		case <-ctx.Done():
			poller.logger.Info("context cancelled, stopping poller")
			return ctx.Err()
		default:
		}

		updates, newMarker, err := poller.pollOnce(ctx, marker)
		if err != nil {

			if errors.Is(err, errPollPanic) {
				consecutivePanics++
				poller.logger.Error("poll panic",
					"consecutive", consecutivePanics,
					"limit", maxConsecutivePollPanics,
					"error", err)
				if consecutivePanics >= maxConsecutivePollPanics {
					return fmt.Errorf("poller stopped after %d consecutive panics: %w",
						consecutivePanics, err)
				}
				time.Sleep(pollPanicBackoff)
				continue
			}

			if maxapi.IsTimeoutError(err) {
				continue
			}

			consecutivePanics = 0
			logging.ErrorWithCause(poller.logger, "get updates failed", err)
			time.Sleep(pollErrorBackoff)
			continue
		}

		consecutivePanics = 0

		if len(updates) == 0 {
			poller.logger.DebugContext(ctx, "no updates received")
		} else {
			poller.logger.DebugContext(ctx, "updates received", "count", len(updates))
		}

		for _, update := range updates {
			poller.handleUpdate(ctx, update)
		}

		marker = newMarker
	}
}

func (poller *Poller) pollOnce(ctx context.Context, marker int64) (
	updates []maxapi.Update, newMarker int64, err error,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			poller.logger.Error("panic recovered in GetUpdates",
				"panic", recovered,
				"stack", string(debug.Stack()))
			err = fmt.Errorf("%w: %v", errPollPanic, recovered)
		}
	}()
	return poller.client.GetUpdates(ctx, marker)
}

func (poller *Poller) handleUpdate(ctx context.Context, update maxapi.Update) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			poller.logger.Error("panic recovered in update handler",
				"panic", recovered,
				"update_type", string(update.Type),
				"stack", string(debug.Stack()))
			err = fmt.Errorf("panic in update handler: %v", recovered)
		}
	}()

	if err = poller.machine.Handle(ctx, update); err != nil {
		logging.ErrorWithCause(poller.logger, "handle update failed", err,
			"update_type", string(update.Type))
	}
	return err
}
