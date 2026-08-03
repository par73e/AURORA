package moon

/** 月球飞行器（静态 curation 目录，镜像地球 spacecraft 表） */

type Spacecraft struct {
	ID                  string  `json:"id"`
	NameZH              string  `json:"nameZh"`
	NameEN              string  `json:"nameEn"`
	Type                string  `json:"type"`
	OperatorName        string  `json:"operatorName"`
	Description         string  `json:"description"`
	LaunchDate          string  `json:"launchDate"`
	LaunchSite          string  `json:"launchSite"`
	LaunchVehicle       string  `json:"launchVehicle"`
	SourceName          string  `json:"sourceName"`
	DisplayInclination  string  `json:"displayInclination"`
	DisplayEccentricity string  `json:"displayEccentricity"`
	DisplayPeriod       string  `json:"displayPeriod"`
	Kind                string  `json:"kind"`
	OrbitA              float64 `json:"orbitA"`
	OrbitE              float64 `json:"orbitE"`
	InclinationDeg      float64 `json:"inclinationDeg"`
	RaanDeg             float64 `json:"raanDeg"`
	ArgPeriapsisDeg     float64 `json:"argPeriapsisDeg"`
	PeriodSeconds       float64 `json:"periodSeconds"`
	StationaryOffset    [3]float64 `json:"stationaryOffset"`
	SortOrder           int     `json:"sortOrder"`
	// 最新同步快照（JPL Horizons；无同步时为 null，前端回退静态参数）
	Snapshot            *OrbitSnapshot `json:"snapshot"`
}

// LandingSite 月球着陆点（真实经纬度，历史测量值）
type LandingSite struct {
	ID           string      `json:"id"`
	NameZH       string      `json:"nameZh"`
	NameEN       string      `json:"nameEn"`
	Program      string      `json:"program"`
	OperatorName string      `json:"operatorName"`
	LandingDate  string      `json:"landingDate"`
	Latitude     float64     `json:"latitude"`
	Longitude    float64     `json:"longitude"`
	Region       string      `json:"region"`
	Description  string      `json:"description"`
	SortOrder    int         `json:"sortOrder"`
	SiteName     string      `json:"siteName"`
	OfficialName string      `json:"officialName"`
	MissionName  string      `json:"missionName"`
	Hardware     []string    `json:"hardware"`
	Side         string      `json:"side"`
	Category     string      `json:"category"`
	Icon         string      `json:"icon"`
	Track        [][]float64 `json:"track"`
}

// OrbitSnapshot 月球飞行器瞬时轨道根数快照
type OrbitSnapshot struct {
	Epoch            string  `json:"epoch"`
	AKm              float64 `json:"aKm"`
	Eccentricity     float64 `json:"eccentricity"`
	InclinationDeg   float64 `json:"inclinationDeg"`
	RaanDeg          float64 `json:"raanDeg"`
	ArgPeriapsisDeg  float64 `json:"argPeriapsisDeg"`
	MeanAnomalyDeg   float64 `json:"meanAnomalyDeg"`
	PeriodSeconds    float64 `json:"periodSeconds"`
}
