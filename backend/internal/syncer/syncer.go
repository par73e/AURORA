package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// client.Timeout=30s 必须 ≤ 最小调度预算（45s）：否则请求可在调度 ctx 截止后继续，
// 出现"僵尸同步"（曾实测 1h38m >> 45s）。
func NewWithMoonVoyageMars(repository *orbit.Repository, moonRepo *moon.Repository, marsRepo *mars.Repository, voyageRepo *voyage.Repository) *Syncer {
	return &Syncer{repository: repository, moonRepo: moonRepo, marsRepo: marsRepo, voyageRepo: voyageRepo, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *Syncer) SyncCelesTrak(ctx context.Context) error {
	return s.syncTargets(ctx, "celestrak", func() ([]targetTask, error) {
		catalog, err := s.repository.SpacecraftCatalog(ctx)
		if err != nil {
			return nil, err
		}
		tasks := make([]targetTask, 0, len(catalog))
		for spacecraftID, catalogID := range catalog {
			tasks = append(tasks, targetTask{id: spacecraftID, supported: true, run: func(ctx context.Context) (int, error) {
				endpoint := fmt.Sprintf("https://celestrak.org/NORAD/elements/gp.php?CATNR=%d&FORMAT=JSON", catalogID)
				body, err := s.getWithRetry(ctx, endpoint)
				if err != nil {
					return 0, err
				}
				var payload []struct {
					Epoch string `json:"EPOCH"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					return 0, err
				}
				if len(payload) == 0 {
					return 0, errors.New("empty CelesTrak payload")
				}
				epoch, err := time.Parse("2006-01-02T15:04:05.999999", payload[0].Epoch)
				if err != nil {
					epoch, err = time.Parse(time.RFC3339Nano, payload[0].Epoch+"Z")
				}
				if err != nil {
					return 0, err
				}
				var raw []json.RawMessage
				if err := json.Unmarshal(body, &raw); err != nil {
					return 0, err
				}
				if err := s.repository.SaveOrbitSnapshot(ctx, spacecraftID, epoch.UTC(), raw[0]); err != nil {
					return 0, err
				}
				return 1, nil
			}})
		}
		return tasks, nil
	})
}

type launchLibraryResponse struct {
	Results []json.RawMessage `json:"results"`
	Next    string            `json:"next"`
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

	now := time.Now().UTC()
	cutoff := now.Add(30 * 24 * time.Hour)
	results, err := s.fetchUpcomingLaunches(ctx, "https://ll.thespacedevs.com/2.3.0/launches/upcoming/", now, cutoff)
	if err != nil {
		syncErr = err
		return err
	}
	seen := make(map[string]bool)
	for _, raw := range results {
		var source launchLibraryEvent
		if err := json.Unmarshal(raw, &source); err != nil {
			syncErr = fmt.Errorf("decode launch event: %w", err)
			return syncErr
		}
		if source.ID == "" || source.Net.IsZero() {
			syncErr = errors.New("launch event is missing ID or NET")
			return syncErr
		}
		if seen[source.ID] || source.Net.Before(now) || source.Net.After(cutoff) {
			continue
		}
		seen[source.ID] = true
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

func (s *Syncer) fetchUpcomingLaunches(ctx context.Context, endpoint string, start, cutoff time.Time) ([]json.RawMessage, error) {
	base, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := base.Query()
	query.Set("limit", "100")
	query.Set("mode", "normal")
	query.Set("ordering", "net")
	query.Set("net__gte", start.Format(time.RFC3339))
	query.Set("net__lte", cutoff.Format(time.RFC3339))
	base.RawQuery = query.Encode()
	next := base.String()
	seenPages := make(map[string]bool)
	var results []json.RawMessage
	for next != "" {
		if seenPages[next] {
			return nil, errors.New("launch pagination returned a repeated page")
		}
		seenPages[next] = true
		body, err := s.get(ctx, next)
		if err != nil {
			return nil, err
		}
		var page launchLibraryResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode launch page: %w", err)
		}
		results = append(results, page.Results...)
		if page.Next == "" {
			break
		}
		parsed, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		resolved := base.ResolveReference(parsed)
		if resolved.Scheme != base.Scheme || resolved.Host != base.Host || resolved.Path != base.Path {
			return nil, errors.New("launch pagination redirected outside its endpoint")
		}
		next = resolved.String()
	}
	return results, nil
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
		return errors.New("voyage repository is not configured")
	}
	return s.syncTargets(ctx, "jpl_horizons_voyage", func() ([]targetTask, error) {
		catalog, err := s.voyageRepo.ListCatalog(ctx)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		tasks := make([]targetTask, 0, len(catalog))
		for _, item := range catalog {
			tasks = append(tasks, targetTask{id: item.ID, supported: item.NaifID != "", run: func(ctx context.Context) (int, error) {
				samples, err := voyage.FetchProbeSamples(ctx, s.client, item.NaifID, now, probeSampleWindowDays(item.ID))
				if err != nil {
					return 0, err
				}
				if len(samples) == 0 {
					return 0, errors.New("empty probe position samples")
				}
				if err := s.voyageRepo.ReplaceProbeSamples(ctx, item.ID, samples); err != nil {
					return 0, err
				}
				return len(samples), nil
			}})
		}
		return tasks, nil
	})
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

// getWithRetry 只为短暂网络抖动补一次请求；重试等待也受当前调度 ctx 约束，不能留下后台僵尸任务。
func (s *Syncer) getWithRetry(ctx context.Context, endpoint string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		body, err := s.get(ctx, endpoint)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if attempt == 1 || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(600 * time.Millisecond):
		}
	}
	return nil, lastErr
}

// SyncMoonSpacecraft 从 JPL Horizons 拉取月球飞行器（LRO/CAPSTONE 等已支持）实时轨道根数；
// 未支持的飞行器跳过（前端回退静态参数）。
func (s *Syncer) SyncMoonSpacecraft(ctx context.Context) error {
	if s.moonRepo == nil {
		return errors.New("moon repository is not configured")
	}
	return s.syncTargets(ctx, "jpl_horizons_moon", func() ([]targetTask, error) {
		catalog, err := s.moonRepo.ListSpacecraft(ctx)
		if err != nil {
			return nil, err
		}
		tasks := make([]targetTask, 0, len(catalog))
		for _, craft := range catalog {
			tasks = append(tasks, targetTask{id: craft.ID, supported: craft.Kind == "orbital" && moon.SupportsHorizons(craft.ID), run: func(ctx context.Context) (int, error) {
				result, err := moon.FetchMoonSpacecraftElements(ctx, s.client, craft.ID)
				if err != nil {
					return 0, err
				}
				if err := s.moonRepo.SaveMoonSnapshot(ctx, craft.ID, result.Epoch.UTC().Format(time.RFC3339), result.Elements, result.Raw); err != nil {
					return 0, err
				}
				return 1, nil
			}})
		}
		return tasks, nil
	})
}

// SyncMarsSpacecraft 火星绕行器实时轨道根数同步（镜像 SyncMoonSpacecraft；CENTER=500@499 火心）
func (s *Syncer) SyncMarsSpacecraft(ctx context.Context) error {
	if s.marsRepo == nil {
		return errors.New("mars repository is not configured")
	}
	return s.syncTargets(ctx, "jpl_horizons_mars", func() ([]targetTask, error) {
		catalog, err := s.marsRepo.ListSpacecraft(ctx)
		if err != nil {
			return nil, err
		}
		tasks := make([]targetTask, 0, len(catalog))
		for _, craft := range catalog {
			tasks = append(tasks, targetTask{id: craft.ID, supported: craft.Kind == "orbital" && mars.SupportsHorizons(craft.ID), run: func(ctx context.Context) (int, error) {
				result, err := mars.FetchMarsSpacecraftElements(ctx, s.client, craft.ID)
				if err != nil {
					return 0, err
				}
				if err := s.marsRepo.SaveMarsSnapshot(ctx, craft.ID, result.Epoch.UTC().Format(time.RFC3339), result.Elements, result.Raw); err != nil {
					return 0, err
				}
				return 1, nil
			}})
		}
		return tasks, nil
	})
}
