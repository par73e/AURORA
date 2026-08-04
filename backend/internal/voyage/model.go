package voyage

/** 深空探测器（VOYAGE 太阳系标注）：静态目录 + JPL Horizons 位置采样 */

// Probe 深空探测器目录与位置采样（位置 = 日心黄道坐标 km，JPL Horizons 黄道 J2000）
type Probe struct {
	ID             string          `json:"id"`
	NameZH         string          `json:"nameZh"`
	NameEN         string          `json:"nameEn"`
	OperatorName   string          `json:"operatorName"`
	LaunchDate     string          `json:"launchDate"`
	MissionType    string          `json:"missionType"`
	Target         string          `json:"target"`
	Description    string          `json:"description"`
	PrecisionGrade string          `json:"precisionGrade"`
	Color          string          `json:"color"`
	SortOrder      int             `json:"sortOrder"`
	// 轨道绘制方式：ellipse = 拟合椭圆（太阳在焦点）；track = 真实采样折线
	OrbitKind string `json:"orbitKind"`
	// 最近一次同步时间（所有采样中最新 synced_at）；无采样时为空
	SyncedAt  string          `json:"syncedAt,omitempty"`
	Positions []PositionPoint `json:"positions"`
}

// PositionPoint 一个位置采样（日心黄道坐标，km）
type PositionPoint struct {
	Epoch string  `json:"epoch"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
}

// CatalogItem 同步任务用的最小目录项（仅 id + NAIF ID）
type CatalogItem struct {
	ID     string
	NaifID string
}
