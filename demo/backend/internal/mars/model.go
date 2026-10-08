package mars

// Spacecraft 火星绕行器目录条目（镜像 moon.Spacecraft）
type Spacecraft struct {
	ID                 string        `json:"id"`
	NameZH             string        `json:"nameZh"`
	NameEN             string        `json:"nameEn"`
	Type               string        `json:"type"`
	OperatorName       string        `json:"operatorName"`
	Description        string        `json:"description"`
	LaunchDate         string        `json:"launchDate"`
	LaunchSite         string        `json:"launchSite"`
	LaunchVehicle      string        `json:"launchVehicle"`
	SourceName         string        `json:"sourceName"`
	DisplayInclination string        `json:"displayInclination"`
	DisplayEccentricity string       `json:"displayEccentricity"`
	DisplayPeriod      string        `json:"displayPeriod"`
	Kind               string        `json:"kind"`           // orbital / stationary / surface / catalog
	CatalogGroup       string        `json:"catalogGroup"`    // surface=地表探测器 / orbit=飞行器（航天器区块标签页）
	OrbitA             float64       `json:"orbitA"`
	OrbitE             float64       `json:"orbitE"`
	InclinationDeg     float64       `json:"inclinationDeg"`
	RaanDeg            float64       `json:"raanDeg"`
	ArgPeriapsisDeg    float64       `json:"argPeriapsisDeg"`
	PeriodSeconds      float64       `json:"periodSeconds"`
	StationaryOffset   [3]float64    `json:"stationaryOffset"`
	SortOrder          int           `json:"sortOrder"`
	Snapshot           *OrbitSnapshot `json:"snapshot,omitempty"`
}

// OrbitSnapshot 火星绕行器瞬时轨道根数快照（JPL Horizons，镜像 moon.OrbitSnapshot）
type OrbitSnapshot struct {
	Epoch           string  `json:"epoch"`
	AKm             float64 `json:"aKm"`
	Eccentricity    float64 `json:"eccentricity"`
	InclinationDeg  float64 `json:"inclinationDeg"`
	RaanDeg         float64 `json:"raanDeg"`
	ArgPeriapsisDeg float64 `json:"argPeriapsisDeg"`
	MeanAnomalyDeg  float64 `json:"meanAnomalyDeg"`
	PeriodSeconds   float64 `json:"periodSeconds"`
}

// LandingSite 火星着陆点（真实经纬度；镜像 moon.LandingSite）
type LandingSite struct {
	ID            string   `json:"id"`
	NameZH        string   `json:"nameZh"`
	NameEN        string   `json:"nameEn"`
	Program       string   `json:"program"`
	OperatorName  string   `json:"operatorName"`
	LandingDate   string   `json:"landingDate"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	Region        string   `json:"region"`
	Description   string   `json:"description"`
	SortOrder     int      `json:"sortOrder"`
	SiteName      string   `json:"siteName,omitempty"`
	OfficialName  string   `json:"officialName,omitempty"`
	MissionName   string   `json:"missionName,omitempty"`
	Hardware      []string `json:"hardware,omitempty"`
	Side          string   `json:"side,omitempty"`
	Category      string   `json:"category,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Track         [][]float64 `json:"track,omitempty"`
}
