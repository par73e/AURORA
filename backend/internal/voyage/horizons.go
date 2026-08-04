package voyage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// JPL Horizons 客户端：拉取深空探测器日心黄道状态矢量采样（±90 天、日采样）。
// API 文档: https://ssd-api.jpl.nasa.gov/doc/horizons.html
// CENTER=500@10 为日心（太阳）；REF_PLANE=ECLIPTIC 黄道坐标；OUT_UNITS=KM-S；
// VEC_TABLE=1 仅返回位置（轨迹渲染不需要速度，省带宽）。

const (
	horizonsEndpoint = "https://ssd.jpl.nasa.gov/api/horizons.api"
	sunNAIFCenter    = "500@10"
	// 采样窗口：当前时刻 ±90 天（前端轨迹线用）；日同步保证窗口始终覆盖"现在"
	windowDays = 90
	stepSize   = "1d"
)

// PositionSample 一个位置采样
type PositionSample struct {
	Epoch time.Time
	X     float64
	Y     float64
	Z     float64
}

// FetchProbeSamples 拉取指定探测器从 now-windowDays 到 now+windowDays 的日心黄道位置采样。
// 返回按时间升序的采样点（约 181 条）；失败返回 error（由调用方降级保留旧数据）。
func FetchProbeSamples(ctx context.Context, client *http.Client, naifID string, now time.Time) ([]PositionSample, error) {
	query := url.Values{}
	query.Set("format", "text")
	query.Set("COMMAND", naifID)
	query.Set("EPHEM_TYPE", "VECTORS")
	query.Set("CENTER", sunNAIFCenter)
	query.Set("REF_PLANE", "ECLIPTIC")
	query.Set("OUT_UNITS", "KM-S")
	query.Set("VEC_TABLE", "1") // 1 = 仅位置
	query.Set("CSV_FORMAT", "YES")
	query.Set("MAKE_EPHEM", "YES")
	// Horizons API 时间值不接受空格：ISO 8601（T 分隔）
	query.Set("START_TIME", now.AddDate(0, 0, -windowDays).Format("2006-01-02T15:04:05"))
	query.Set("STOP_TIME", now.AddDate(0, 0, windowDays).Format("2006-01-02T15:04:05"))
	query.Set("STEP_SIZE", stepSize)

	endpoint := horizonsEndpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("horizons request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("horizons read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("horizons status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	// 文本错误提示（未知 COMMAND 等）也会以 200 返回
	if !strings.Contains(strings.ToLower(string(body)), "$$soe") {
		return nil, fmt.Errorf("horizons no ephemeris data: %s", truncate(string(body), 300))
	}
	samples, err := parseSOESamples(string(body))
	if err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("horizons SOE 数据行为空")
	}
	return samples, nil
}

// parseSOESamples 解析 $$SOE … $$EOE 区段全部数据行：
// JD, CAL, x, y, z（VEC_TABLE=1 每行 5 列）
func parseSOESamples(text string) ([]PositionSample, error) {
	start := strings.Index(text, "$$SOE")
	end := strings.Index(text, "$$EOE")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("horizons SOE 区段缺失")
	}
	section := text[start+5 : end]
	samples := []PositionSample{}
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 { // JD, CAL, x, y, z
			continue
		}
		cal := strings.TrimSpace(fields[1])
		epoch, err := time.Parse("A.D. 2006-Jan-02 15:04:05.0000", cal)
		if err != nil {
			epoch, err = time.Parse("A.D. 2006-Jan-02 15:04:05", cal)
		}
		if err != nil {
			return nil, fmt.Errorf("horizons 历元解析失败: %q", cal)
		}
		var nums [3]float64
		for i := 0; i < 3; i++ {
			if _, err := fmt.Sscanf(strings.TrimSpace(fields[2+i]), "%f", &nums[i]); err != nil {
				return nil, fmt.Errorf("horizons 数值解析失败: %q", fields[2+i])
			}
		}
		samples = append(samples, PositionSample{
			Epoch: epoch.UTC(),
			X:     nums[0],
			Y:     nums[1],
			Z:     nums[2],
		})
	}
	return samples, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
