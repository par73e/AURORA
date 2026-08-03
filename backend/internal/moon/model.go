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
