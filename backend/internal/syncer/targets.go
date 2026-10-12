package syncer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type targetTask struct {
	id        string
	supported bool
	run       func(context.Context) (int, error) // commits data and successful acquisition time together
}

// All spacecraft pipelines use the same failure and freshness rules. Cancellation
// still finalizes metadata with a bounded context, without advancing success time.
func (s *Syncer) syncTargets(ctx context.Context, source string, catalog func() ([]targetTask, error)) (syncErr error) {
	runID, err := s.repository.StartSync(ctx, source)
	if err != nil {
		return err
	}
	records := 0
	var canceledMetadataCtx context.Context
	defer func() {
		finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		syncErr = errors.Join(syncErr, s.repository.FinishSync(finishCtx, runID, records, syncErr))
	}()
	tasks, err := catalog()
	if err != nil {
		return err
	}
	eligible, failed := 0, 0
	for _, task := range tasks {
		attemptedAt := time.Now().UTC()
		status := "unsupported"
		var targetErr error
		if task.supported {
			eligible++
			targetErr = ctx.Err()
			if targetErr == nil {
				var written int
				written, targetErr = task.run(ctx)
				records += written
			}
			status = "success"
			if targetErr != nil {
				failed++
				status = "failed"
				if errors.Is(targetErr, context.Canceled) || errors.Is(targetErr, context.DeadlineExceeded) {
					status = "canceled"
				}
			}
		}
		if status == "success" {
			continue
		}
		if targetErr != nil {
			slog.Warn("spacecraft target sync failed", "source", source, "target", task.id, "error", targetErr)
		}
		saveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if ctx.Err() != nil {
			if canceledMetadataCtx == nil {
				var cancelMetadata context.CancelFunc
				canceledMetadataCtx, cancelMetadata = context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelMetadata()
			}
			saveCtx = canceledMetadataCtx
		}
		err := s.repository.RecordTargetSync(saveCtx, source, task.id, attemptedAt, status, targetErr)
		cancel()
		if err != nil {
			return errors.Join(targetErr, fmt.Errorf("record target sync %s: %w", task.id, err))
		}
	}
	if eligible == 0 {
		return errors.New(source + " sync has no supported targets")
	}
	if failed > 0 {
		return errors.Join(ctx.Err(), fmt.Errorf("%s sync incomplete: %d/%d supported targets failed", source, failed, eligible))
	}
	return nil
}
