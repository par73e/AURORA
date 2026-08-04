package voyage

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aurora/backend/internal/config"
	"aurora/backend/internal/database"
)

// 深空探测器目录/同步数据集成冒烟测试（需本地 aurora 库与已同步数据；库不可用时自动跳过，
// 不影响常规构建）。用于"比对验证"：确保新增探测器（先驱者10/11、尤利西斯、露西、
// STEREO-A、隼鸟2）与原有 5 颗一样：字段齐全、采样窗口覆盖当前时刻、距离符合公开数值。

const auKM = 149597870.7

var expectedIDs = []string{
	"parker", "solar-orbiter", "new-horizons", "voyager-1", "voyager-2",
	"pioneer-10", "pioneer-11", "ulysses", "lucy", "stereo-a", "hayabusa-2",
}

func openTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	pool, err := database.Open(ctx, config.Load().DatabaseURL)
	if err != nil {
		t.Skipf("数据库不可用，跳过集成测试: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

// parseEpoch 解析后端序列化的 RFC3339 历元（如 2026-08-04T00:00:00Z）；失败返回零值
func parseEpoch(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// probeAtNow 在采样窗口内线性插值"当前时刻"位置；窗口未覆盖 now 时返回 ok=false
func probeAtNow(p *Probe, now time.Time) (x, y, z float64, ok bool) {
	pts := p.Positions
	if len(pts) < 2 {
		return 0, 0, 0, false
	}
	if now.Before(parseEpoch(pts[0].Epoch)) || now.After(parseEpoch(pts[len(pts)-1].Epoch)) {
		return 0, 0, 0, false
	}
	for i := 0; i < len(pts)-1; i++ {
		t0, t1 := parseEpoch(pts[i].Epoch), parseEpoch(pts[i+1].Epoch)
		if !t1.After(t0) {
			continue // 重复/乱序历元，跳过该段
		}
		if !now.Before(t0) && !now.After(t1) {
			f := float64(now.Sub(t0)) / float64(t1.Sub(t0))
			lerp := func(a, b float64) float64 { return a + (b-a)*f }
			return lerp(pts[i].X, pts[i+1].X), lerp(pts[i].Y, pts[i+1].Y), lerp(pts[i].Z, pts[i+1].Z), true
		}
	}
	return 0, 0, 0, false
}

// TestProbeCatalogSanity 目录完整性：11 颗、字段齐全、采样窗口覆盖当前时刻
func TestProbeCatalogSanity(t *testing.T) {
	ctx, pool := openTestDB(t)
	repo := NewRepository(pool)

	probes, err := repo.ListProbes(ctx)
	if err != nil {
		t.Fatalf("ListProbes: %v", err)
	}
	if len(probes) != len(expectedIDs) {
		t.Errorf("探测器目录应含 %d 个，实际 %d", len(expectedIDs), len(probes))
	}

	catalog, err := repo.ListCatalog(ctx)
	if err != nil {
		t.Fatalf("ListCatalog: %v", err)
	}
	naifByID := map[string]string{}
	for _, c := range catalog {
		naifByID[c.ID] = c.NaifID
	}

	now := time.Now().UTC()
	seen := map[string]bool{}
	for i := range probes {
		p := &probes[i]
		seen[p.ID] = true
		if naifByID[p.ID] == "" {
			t.Errorf("%s: naif_id 缺失", p.ID)
		}
		if p.OrbitKind == "" || (p.OrbitKind != "track" && p.OrbitKind != "ellipse") {
			t.Errorf("%s: orbit_kind 非法 %q", p.ID, p.OrbitKind)
		}
		if p.LaunchSite == "" || p.LaunchVehicle == "" {
			t.Errorf("%s: 发射地点/火箭缺失", p.ID)
		}
		if p.Color == "" {
			t.Errorf("%s: 颜色缺失", p.ID)
		}
		// 采样数：日采样应约等于窗口跨度天数（正常 ±90 天 ≈ 181）；STEREO-A/隼鸟2
		// 星历截止时窗口被钳制、跨度随之变短——下限相对窗口跨度计算，避免 2026 年
		// 9–10 月后窗口收缩导致误报（允许 ±3 条松弛）
		if len(p.Positions) >= 2 {
			spanDays := int(parseEpoch(p.Positions[len(p.Positions)-1].Epoch).Sub(parseEpoch(p.Positions[0].Epoch)).Hours()/24) + 1
			if len(p.Positions) < spanDays-3 {
				t.Errorf("%s: 采样数 %d 少于窗口跨度 %d 天（日采样）", p.ID, len(p.Positions), spanDays)
			}
		}
		if _, _, _, ok := probeAtNow(p, now); !ok {
			t.Errorf("%s: 采样窗口未覆盖当前时刻（前端标记将处于外推状态）", p.ID)
		}
	}
	for _, id := range expectedIDs {
		if !seen[id] {
			t.Errorf("缺少探测器 %s", id)
		}
	}
}

// TestProbeCurrentDistances 当前日心距离与公开数值比对（宽松范围防数据/换算错误；输出实际值）
func TestProbeCurrentDistances(t *testing.T) {
	ctx, pool := openTestDB(t)
	repo := NewRepository(pool)

	probes, err := repo.ListProbes(ctx)
	if err != nil {
		t.Fatalf("ListProbes: %v", err)
	}

	// 公开数值（2026 年中概略）：先驱者10 ~130 AU、先驱者11 ~106 AU、新视野 ~65 AU、
	// 旅行者1 ~171 AU、旅行者2 ~144 AU；其余为绕日轨道内插
	ranges := map[string][2]float64{
		"parker":        {0.1, 0.9},
		"solar-orbiter": {0.2, 1.2},
		"new-horizons":  {55, 80},
		"voyager-1":     {150, 195},
		"voyager-2":     {125, 170},
		"pioneer-10":    {110, 155},
		"pioneer-11":    {90, 135},
		"ulysses":       {1.0, 6.5},
		"lucy":          {0.5, 7.0},
		"stereo-a":      {0.8, 1.5},
		"hayabusa-2":    {0.5, 3.5},
	}
	now := time.Now().UTC()
	for i := range probes {
		p := &probes[i]
		want, ok := ranges[p.ID]
		if !ok {
			continue
		}
		x, y, z, found := probeAtNow(p, now)
		if !found {
			t.Errorf("%s: 无法插值当前位置", p.ID)
			continue
		}
		au := math.Sqrt(x*x+y*y+z*z) / auKM
		t.Logf("%s 当前距日 %.2f AU（范围 [%.1f, %.1f]）", p.ID, au, want[0], want[1])
		if au < want[0] || au > want[1] {
			t.Errorf("%s 当前距日 %.2f AU 超出合理范围 [%.1f, %.1f]", p.ID, au, want[0], want[1])
		}
	}
}
