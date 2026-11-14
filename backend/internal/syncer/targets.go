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

// errTargetUnpublished 表示上游目录已不再提供该目标（如 CelesTrak 对已离目录的
// NORAD 编号返回 404）。这是永久状态而非同步故障：单目标下架不应把整个数据源判失败。
var errTargetUnpublished = errors.New("target no longer published by upstream catalog")

// targetStatus 把单个目标的同步结果归入状态。上游目录下架（errTargetUnpublished）
// 是永久状态而非同步故障，不计入 failed，避免单目标下架把整个数据源判失败。
func targetStatus(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, errTargetUnpublished):
		return "unpublished"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "canceled"
	default:
		return "failed"
	}
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
	eligible, failed, unpublished := 0, 0, 0
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
			status = targetStatus(targetErr)
			switch status {
			case "failed", "canceled":
				failed++
			case "unpublished":
				unpublished++
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
	if err := sourceResult(source, eligible, failed, unpublished, ctx.Err()); err != nil {
		return err
	}
	return nil
}

// sourceResult 归并单源各目标的同步结果。单个目标被上游目录下架不算源失败；
// 但全部目标都已下架说明目录本身不可用，仍判失败，避免把空同步记为成功。
func sourceResult(source string, eligible, failed, unpublished int, ctxErr error) error {
	if eligible == 0 {
		return errors.New(source + " sync has no supported targets")
	}
	if failed > 0 {
		return errors.Join(ctxErr, fmt.Errorf("%s sync incomplete: %d/%d supported targets failed", source, failed, eligible))
	}
	if unpublished == eligible {
		return errors.New(source + " sync has no published targets")
	}
	return nil
}
