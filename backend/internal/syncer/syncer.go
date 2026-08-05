package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/voyage"
)

type Syncer struct {
	repository *orbit.Repository
	moonRepo   *moon.Repository
	marsRepo   *mars.Repository
	voyageRepo *voyage.Repository
	client     *http.Client
}

func New(repository *orbit.Repository) *Syncer {
	return &Syncer{repository: repository, client: &http.Client{Timeout: 20 * time.Second}}
}

// NewWithMoon 附带月球仓库（月球飞行器同步需要）
func NewWithMoon(repository *orbit.Repository, moonRepo *moon.Repository) *Syncer {
	return &Syncer{repository: repository, moonRepo: moonRepo, client: &http.Client{Timeout: 30 * time.Second}}
}

// NewWithMoonVoyageMars 附带月球、火星与深空探测器仓库（同步都需要）
func NewWithMoonVoyageMars(repository *orbit.Repository, moonRepo *moon.Repository, marsRepo *mars.Repository, voyageRepo *voyage.Repository) *Syncer {
	return &Syncer{repository: repository, moonRepo: moonRepo, marsRepo: marsRepo, voyageRepo: voyageRepo, client: &http.Client{Timeout: 60 * time.Second}}
}

func (s *Syncer) SyncCelesTrak(ctx context.Context) error {
	runID, err := s.repository.StartSync(ctx, "celestrak")
	if err != nil {
		return err
	}
	records := 0
	var syncErr error
	defer func() { _ = s.repository.FinishSync(context.Background(), runID, records, syncErr) }()

	catalog, err := s.repository.SpacecraftCatalog(ctx)
	if err != nil {
		syncErr = err
		return err
	}
	for spacecraftID, catalogID := range catalog {
		endpoint := fmt.Sprintf("https://celestrak.org/NORAD/elements/gp.php?CATNR=%d&FORMAT=JSON", catalogID)
		body, err := s.get(ctx, endpoint)
		if err != nil {
			syncErr = err
			return err
		}
		var payload []struct {
			Epoch string `json:"EPOCH"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || len(payload) == 0 {
			if err == nil {
				err = errors.New("empty CelesTrak payload")
			}
			syncErr = err
			return err
		}
		epoch, err := time.Parse("2006-01-02T15:04:05.999999", payload[0].Epoch)
		if err != nil {
			epoch, err = time.Parse(time.RFC3339Nano, payload[0].Epoch+"Z")
		}
		if err != nil {
			syncErr = fmt.Errorf("parse CelesTrak epoch: %w", err)
			return syncErr
		}
		var rawItems []json.RawMessage
		if err := json.Unmarshal(body, &rawItems); err != nil || len(rawItems) == 0 {
			syncErr = fmt.Errorf("decode CelesTrak raw payload: %w", err)
			return syncErr
		}
		if err := s.repository.SaveOrbitSnapshot(ctx, spacecraftID, epoch.UTC(), rawItems[0]); err != nil {
			syncErr = err
			return err
		}
		records++
	}
	return nil
}

type launchLibraryResponse struct {
	Results []json.RawMessage `json:"results"`
}

type launchLibraryEvent struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Net         time.Time  `json:"net"`
	WindowStart *time.Time `json:"window_start"`
	WindowEnd   *time.Time `json:"window_end"`
	Status      struct {
		Name   string `json:"name"`
		Abbrev string `json:"abbrev"`
	} `json:"status"`
	Pad *struct {
		Name      string   `json:"name"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Location  *struct {
			Name string `json:"name"`
		} `json:"location"`
	} `json:"pad"`
	Mission *struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"mission"`
	LSP *struct {
		Name string `json:"name"`
	} `json:"lsp"`
}

func (s *Syncer) SyncLaunches(ctx context.Context) error {
	runID, err := s.repository.StartSync(ctx, "launch_library_2")
	if err != nil {
		return err
	}
	records := 0
	var syncErr error
	defer func() { _ = s.repository.FinishSync(context.Background(), runID, records, syncErr) }()

	query := url.Values{}
	query.Set("limit", "20")
	query.Set("mode", "normal")
	body, err := s.get(ctx, "https://ll.thespacedevs.com/2.3.0/launches/upcoming/?"+query.Encode())
	if err != nil {
		syncErr = err
		return err
	}
	var response launchLibraryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		syncErr = err
		return err
	}
	cutoff := time.Now().UTC().Add(30 * 24 * time.Hour)
	for _, raw := range response.Results {
		var source launchLibraryEvent
		if err := json.Unmarshal(raw, &source); err != nil {
			continue
		}
		if source.Net.After(cutoff) {
			continue
		}
		event := orbit.LaunchEvent{ExternalID: source.ID, Name: source.Name, StatusName: source.Status.Name, StatusAbbrev: source.Status.Abbrev, Net: source.Net, WindowStart: source.WindowStart, WindowEnd: source.WindowEnd, SourceURL: "https://ll.thespacedevs.com/2.3.0/launches/" + source.ID + "/"}
		if source.Pad != nil {
			event.PadName = source.Pad.Name
			event.Latitude = source.Pad.Latitude
			event.Longitude = source.Pad.Longitude
			if source.Pad.Location != nil {
				event.LocationName = source.Pad.Location.Name
			}
		}
		if source.Mission != nil {
			event.MissionName = source.Mission.Name
			event.MissionType = source.Mission.Type
			event.MissionDescription = source.Mission.Description
		}
		if source.LSP != nil {
			event.ProviderName = source.LSP.Name
		}
		orbit.LocalizeLaunchEvent(&event)
		// 坐标单源：能匹配到库内发射场则记录 launch_site_id（查询时坐标以站点为准），否则回退事件自带坐标
		siteID, err := s.repository.MatchLaunchSite(ctx, event.LocationName, event.PadName)
		if err != nil {
			syncErr = err
			return err
		}
		event.LaunchSiteID = siteID
		if err := s.repository.SaveLaunchEvent(ctx, event, raw); err != nil {
			syncErr = err
			return err
		}
		records++
	}
	return nil
}

// probeSampleWindowDays 返回探测器的采样窗口天数（±N 天）。
// 默认 ±90（日同步，前端轨迹插值用）；斯皮策（-79）真实轨道周期约 377 天，
// ±90 仅覆盖约 48% 轨道、椭圆拟合（u=1/r 最小二乘）需要完整一圈——用 ±200 覆盖 400 天（>377，留余量）。
func probeSampleWindowDays(id string) int {
	if id == "spitzer" {
		return 200
	}
	return 90
}

// SyncDeepSpaceProbes 从 JPL Horizons 拉取深空探测器日心位置采样（默认 ±90 天、日采样），
// 整窗替换每个目标的位置采样表（同步失败保留旧数据，前端仍可展示）。
func (s *Syncer) SyncDeepSpaceProbes(ctx context.Context) error {
	if s.voyageRepo == nil {
		return errors.New("voyage repository 未配置")
	}
	runID, err := s.repository.StartSync(ctx, "jpl_horizons")
	if err != nil {
		return err
	}
	records := 0
	var syncErr error
	defer func() { _ = s.repository.FinishSync(context.Background(), runID, records, syncErr) }()

	catalog, err := s.voyageRepo.ListCatalog(ctx)
	if err != nil {
		syncErr = err
		return err
	}
	now := time.Now().UTC()
	for _, item := range catalog {
		samples, err := voyage.FetchProbeSamples(ctx, s.client, item.NaifID, now, probeSampleWindowDays(item.ID))
		if err != nil {
			// 单个目标失败不 abort 整轮：记日志继续，其余目标照常刷新
			slog.Warn("deep space probe sync skipped", "probe", item.ID, "error", err)
			continue
		}
		if err := s.voyageRepo.ReplaceProbeSamples(ctx, item.ID, samples); err != nil {
			slog.Warn("deep space probe samples save failed", "probe", item.ID, "error", err)
			continue
		}
		records += len(samples)
		slog.Info("deep space probe synced", "probe", item.ID, "samples", len(samples),
			"window", now.AddDate(0, 0, -90).Format("2006-01-02")+" → "+now.AddDate(0, 0, 90).Format("2006-01-02"))
	}
	return nil
}

func (s *Syncer) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AURORA/0.1 (+local-development)")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %s", endpoint, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return body, nil
}

// SyncMoonSpacecraft 从 JPL Horizons 拉取月球飞行器（LRO/CAPSTONE 等已支持）实时轨道根数；
// 未支持的飞行器跳过（前端回退静态参数）。
func (s *Syncer) SyncMoonSpacecraft(ctx context.Context) error {
	if s.moonRepo == nil {
		return errors.New("moon repository 未配置")
	}
	runID, err := s.repository.StartSync(ctx, "jpl_horizons")
	if err != nil {
		return err
	}
	records := 0
	var syncErr error
	defer func() { _ = s.repository.FinishSync(context.Background(), runID, records, syncErr) }()

	catalog, err := s.moonRepo.ListSpacecraft(ctx)
	if err != nil {
		syncErr = err
		return err
	}
	for _, craft := range catalog {
		if craft.Kind != "orbital" {
			continue // 定点飞行器（鹊桥二号等）无绕月轨道
		}
		result, err := moon.FetchMoonSpacecraftElements(ctx, s.client, craft.ID)
		if err != nil {
			// 单个飞行器失败（未支持/星历结束/临时错误）不 abort 整轮：
			// 记录日志继续，其余飞行器照常同步，前端回退静态参数
			slog.Warn("moon orbit sync skipped", "craft", craft.ID, "error", err)
			continue
		}
		if err := s.moonRepo.SaveMoonSnapshot(ctx, craft.ID, result.Epoch.UTC().Format(time.RFC3339), result.Elements, result.Raw); err != nil {
			syncErr = err
			return err
		}
		records++
		slog.Info("moon orbit synced", "craft", craft.ID, "epoch", result.Epoch.UTC().Format(time.RFC3339),
			"a_km", fmt.Sprintf("%.1f", result.Elements.A), "period", fmt.Sprintf("%.0fs", result.Elements.PeriodSeconds))
	}
	return nil
}

// SyncMarsSpacecraft 火星绕行器实时轨道根数同步（镜像 SyncMoonSpacecraft；CENTER=500@499 火心）
func (s *Syncer) SyncMarsSpacecraft(ctx context.Context) error {
	if s.marsRepo == nil {
		return errors.New("mars repository 未配置")
	}
	runID, err := s.repository.StartSync(ctx, "jpl_horizons")
	if err != nil {
		return err
	}
	records := 0
	var syncErr error
	defer func() { _ = s.repository.FinishSync(context.Background(), runID, records, syncErr) }()

	catalog, err := s.marsRepo.ListSpacecraft(ctx)
	if err != nil {
		syncErr = err
		return err
	}
	for _, craft := range catalog {
		result, err := mars.FetchMarsSpacecraftElements(ctx, s.client, craft.ID)
		if err != nil {
			// 单个飞行器失败（未支持/星历结束/临时错误）不 abort 整轮：
			// 记录日志继续，其余飞行器照常同步，前端回退静态参数
			slog.Warn("mars orbit sync skipped", "craft", craft.ID, "error", err)
			continue
		}
		if err := s.marsRepo.SaveMarsSnapshot(ctx, craft.ID, result.Epoch.UTC().Format(time.RFC3339), result.Elements, result.Raw); err != nil {
			syncErr = err
			return err
		}
		records++
		slog.Info("mars orbit synced", "craft", craft.ID, "epoch", result.Epoch.UTC().Format(time.RFC3339),
			"a_km", fmt.Sprintf("%.1f", result.Elements.A), "period", fmt.Sprintf("%.0fs", result.Elements.PeriodSeconds))
	}
	return nil
}
