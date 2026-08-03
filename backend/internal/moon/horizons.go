package moon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// JPL Horizons 客户端：拉取月球飞行器（LRO）月心状态矢量，转成瞬时轨道根数
// API 文档: https://ssd-api.jpl.nasa.gov/doc/horizons.html
// LRO 的 NASA NAIF ID 为 -850；CENTER=500@301 为月球（月心坐标）

const (
	horizonsEndpoint = "https://ssd.jpl.nasa.gov/api/horizons.api"
	moonNAIFCenter   = "500@301"
	lroNAIFID        = "-85" // Horizons 中 LRO 的 ID 是 -85（-850 会报 No such record）
)

// HorizonsResult 一次同步的结果
type HorizonsResult struct {
	Epoch    time.Time
	Elements Elements
	Raw      string
}

// FetchMoonSpacecraftElements 从 Horizons 拉取指定飞行器当前状态矢量并转换。
// 目前仅支持 LRO（NAIF -850）；失败返回 error，由调用方降级（保留旧数据）。
func FetchMoonSpacecraftElements(ctx context.Context, client *http.Client, spacecraftID string) (HorizonsResult, error) {
	if spacecraftID != "lro" {
		return HorizonsResult{}, fmt.Errorf("spacecraft %q 暂无 Horizons 支持", spacecraftID)
	}
	now := time.Now().UTC()
	query := url.Values{}
	query.Set("format", "text")
	query.Set("COMMAND", lroNAIFID)
	query.Set("EPHEM_TYPE", "VECTORS")
	query.Set("CENTER", moonNAIFCenter)
	query.Set("REF_PLANE", "ECLIPTIC")
	query.Set("OUT_UNITS", "KM-S")
	query.Set("VEC_TABLE", "2") // 2 = 位置 + 速度（rv → 轨道根数需要速度）
	query.Set("CSV_FORMAT", "YES")
	// Horizons API 的时间值不接受空格：ISO 8601（T 分隔）；STEP_SIZE 单位紧凑写法（1min）
	query.Set("MAKE_EPHEM", "YES")
	query.Set("START_TIME", now.Add(-10*time.Minute).Format("2006-01-02T15:04:05"))
	query.Set("STOP_TIME", now.Add(10*time.Minute).Format("2006-01-02T15:04:05"))
	query.Set("STEP_SIZE", "1min")

	endpoint := horizonsEndpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return HorizonsResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return HorizonsResult{}, fmt.Errorf("horizons request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return HorizonsResult{}, fmt.Errorf("horizons read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return HorizonsResult{}, fmt.Errorf("horizons status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	// 文本错误提示（如未知 COMMAND）也会以 200 返回
	lower := strings.ToLower(string(body))
	if strings.Contains(lower, "$$soe") == false {
		return HorizonsResult{}, fmt.Errorf("horizons no ephemeris data: %s", truncate(string(body), 300))
	}

	epoch, r, v, err := parseHorizonsSOE(string(body))
	if err != nil {
		return HorizonsResult{}, err
	}
	return HorizonsResult{
		Epoch:    epoch,
		Elements: RVToElements(r[0], r[1], r[2], v[0], v[1], v[2]),
		Raw:      truncate(string(body), 4000),
	}, nil
}

// parseHorizonsSOE 解析 $$SOE … $$EOE 区段的数据行：
// JD, CAL, x, y, z, vx, vy, vz, …（取第一行）
func parseHorizonsSOE(text string) (time.Time, [3]float64, [3]float64, error) {
	start := strings.Index(text, "$$SOE")
	end := strings.Index(text, "$$EOE")
	if start < 0 || end < 0 || end <= start {
		return time.Time{}, [3]float64{}, [3]float64{}, fmt.Errorf("horizons SOE 区段缺失")
	}
	section := text[start+5 : end]
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 8 { // JD, CAL, x, y, z, vx, vy, vz
			continue
		}
		cal := strings.TrimSpace(fields[1])
		epoch, err := time.Parse("A.D. 2006-Jan-02 15:04:05.0000", cal)
		if err != nil {
			// 尝试不带毫秒
			epoch, err = time.Parse("A.D. 2006-Jan-02 15:04:05", cal)
		}
		if err != nil {
			return time.Time{}, [3]float64{}, [3]float64{}, fmt.Errorf("horizons 历元解析失败: %q", cal)
		}
		var nums [6]float64
		for i := 0; i < 6; i++ {
			if _, err := fmt.Sscanf(strings.TrimSpace(fields[2+i]), "%f", &nums[i]); err != nil {
				return time.Time{}, [3]float64{}, [3]float64{}, fmt.Errorf("horizons 数值解析失败: %q", fields[2+i])
			}
		}
		return epoch.UTC(), [3]float64{nums[0], nums[1], nums[2]}, [3]float64{nums[3], nums[4], nums[5]}, nil
	}
	return time.Time{}, [3]float64{}, [3]float64{}, fmt.Errorf("horizons SOE 数据行为空")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
