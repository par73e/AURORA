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

	"aurora/backend/internal/orbit"
)

type Syncer struct {
	repository *orbit.Repository
	client     *http.Client
}

func New(repository *orbit.Repository) *Syncer {
	return &Syncer{repository: repository, client: &http.Client{Timeout: 20 * time.Second}}
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
		if err := s.repository.SaveLaunchEvent(ctx, event, raw); err != nil {
			syncErr = err
			return err
		}
		records++
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
