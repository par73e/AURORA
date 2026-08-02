package orbit

import (
	"encoding/json"
	"time"
)

type Spacecraft struct {
	ID             string          `json:"id"`
	NameZH         string          `json:"nameZh"`
	NameEN         string          `json:"nameEn"`
	NORADCatalogID int64           `json:"noradCatalogId"`
	Category       string          `json:"category"`
	OperatorName   string          `json:"operatorName"`
	Description    string          `json:"description"`
	SourceName     string          `json:"sourceName"`
	SourceURL      string          `json:"sourceUrl"`
	OrbitEpoch     *time.Time      `json:"orbitEpoch,omitempty"`
	OrbitSyncedAt  *time.Time      `json:"orbitSyncedAt,omitempty"`
	OMM            json.RawMessage `json:"omm,omitempty"`
}

type LaunchSite struct {
	ID            string  `json:"id"`
	NameZH        string  `json:"nameZh"`
	NameEN        string  `json:"nameEn"`
	CountryCode   string  `json:"countryCode"`
	CountryNameZH string  `json:"countryNameZh"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Description   string  `json:"description"`
	SourceURL     string  `json:"sourceUrl"`
}

type LaunchEvent struct {
	ExternalID         string     `json:"externalId"`
	Name               string     `json:"name"`
	StatusName         string     `json:"statusName"`
	StatusAbbrev       string     `json:"statusAbbrev"`
	Net                time.Time  `json:"net"`
	WindowStart        *time.Time `json:"windowStart,omitempty"`
	WindowEnd          *time.Time `json:"windowEnd,omitempty"`
	PadName            string     `json:"padName,omitempty"`
	LocationName       string     `json:"locationName,omitempty"`
	Latitude           *float64   `json:"latitude,omitempty"`
	Longitude          *float64   `json:"longitude,omitempty"`
	MissionName        string     `json:"missionName,omitempty"`
	MissionType        string     `json:"missionType,omitempty"`
	MissionDescription string     `json:"missionDescription,omitempty"`
	ProviderName       string     `json:"providerName,omitempty"`
	SourceURL          string     `json:"sourceUrl"`
	SyncedAt           time.Time  `json:"syncedAt"`
}

type DataFreshness struct {
	SourceCode     string     `json:"sourceCode"`
	SourceName     string     `json:"sourceName"`
	LastFinishedAt *time.Time `json:"lastFinishedAt,omitempty"`
	Success        *bool      `json:"success,omitempty"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
}

type Overview struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Spacecraft  []Spacecraft    `json:"spacecraft"`
	LaunchSites []LaunchSite    `json:"launchSites"`
	Events      []LaunchEvent   `json:"events"`
	Freshness   []DataFreshness `json:"freshness"`
}
