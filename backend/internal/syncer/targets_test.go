package syncer

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestTargetStatusClassification(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"success", nil, "success"},
		{"unpublished", fmt.Errorf("wrap: %w", errTargetUnpublished), "unpublished"},
		{"canceled", context.Canceled, "canceled"},
		{"deadline", context.DeadlineExceeded, "canceled"},
		{"failed", errors.New("transient upstream error"), "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := targetStatus(tc.err); got != tc.want {
				t.Fatalf("targetStatus(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestSourceResultToleratesUnpublishedTargets(t *testing.T) {
	// 单个目标被上游目录下架：源级同步仍算成功（其余目标数据新鲜）。
	if err := sourceResult("celestrak", 26, 0, 1, nil); err != nil {
		t.Fatalf("one unpublished target failed the whole source: %v", err)
	}
	// 真实同步故障仍然判源失败。
	if err := sourceResult("celestrak", 26, 1, 1, nil); err == nil {
		t.Fatal("a real target failure must fail the source")
	}
	// 全部目标下架说明目录本身不可用，不能记为成功。
	if err := sourceResult("celestrak", 3, 0, 3, nil); err == nil {
		t.Fatal("a fully unpublished catalog must fail the source")
	}
	if err := sourceResult("celestrak", 0, 0, 0, nil); err == nil {
		t.Fatal("a source without supported targets must fail")
	}
}

func TestClassifyCatalogErrorMaps404ToUnpublished(t *testing.T) {
	err := classifyCatalogError(&httpStatusError{endpoint: "https://celestrak.org/NORAD/elements/gp.php", status: 404}, 28485)
	if !errors.Is(err, errTargetUnpublished) {
		t.Fatalf("404 was not classified as unpublished: %v", err)
	}
	plain := errors.New("transient")
	if got := classifyCatalogError(plain, 25544); got != plain {
		t.Fatalf("non-status error was rewritten: %v", got)
	}
	if got := classifyCatalogError(&httpStatusError{status: 500}, 25544); errors.Is(got, errTargetUnpublished) {
		t.Fatalf("500 misclassified as unpublished: %v", got)
	}
}
