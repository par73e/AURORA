package orbit

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var starlinkCountPattern = regexp.MustCompile(`(?i)A batch of (\d+) satellites`)

// LocalizeLaunchEvent keeps the source fields untouched and builds a separate,
// reviewable Chinese presentation layer for the ORBIT interface.
func LocalizeLaunchEvent(event *LaunchEvent) {
	if event == nil {
		return
	}
	event.NameZH = translateLaunchName(event.Name)
	event.StatusNameZH = translateStatus(event.StatusName)
	event.PadNameZH = translatePad(event.PadName)
	event.LocationNameZH = translateLocation(event.LocationName)
	event.MissionNameZH = translateMissionName(event.MissionName)
	event.MissionTypeZH = translateMissionType(event.MissionType)
	event.MissionDescriptionZH = translateMissionDescription(*event)
	event.HasOriginal = needsOriginal(*event)
}

func translateLaunchName(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	parts := strings.SplitN(value, " | ", 2)
	if len(parts) == 1 {
		return translateMissionName(value)
	}
	return translateVehicle(parts[0]) + "｜" + translateMissionName(parts[1])
}

func translateVehicle(value string) string {
	known := map[string]string{
		"Long March 8A":    "长征八号甲",
		"Long March 7A":    "长征七号甲",
		"Long March 12":    "长征十二号",
		"Long March 5":     "长征五号",
		"Smart Dragon 3":   "捷龙三号",
		"Falcon 9 Block 5": "猎鹰9号 Block 5",
		"Falcon Heavy":     "猎鹰重型",
		"Spectrum":         "Spectrum 火箭",
		"Electron":         "电子号",
		"Ariane 62":        "阿丽亚娜62",
		"Zhuque-3":         "朱雀三号",
		"Mir":              "Mir 火箭",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	return value
}

func translateMissionName(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	known := map[string]string{
		"Unknown Payload":                   "未知载荷",
		"BlueBird 11-13 (Block 2 #6-8)":     "蓝鸟 11–13（Block 2 第 6–8 颗）",
		"Onward and Upward":                 "奋勇向上",
		"Michibiki 7 (QZS-7)":               "引路七号（QZS-7）",
		"Chang'e 7":                         "嫦娥七号",
		"MTG-I2":                            "MTG-I2 气象卫星",
		"Nancy Grace Roman Space Telescope": "南希·格蕾丝·罗曼空间望远镜",
		"LOXSAT 1":                          "LOXSAT 1 号",
		"StriX Launch 11":                   "StriX 第 11 次发射",
		"Flight 2":                          "第二次试飞",
		"Globalstar 2-R Mission 1 (x 9)":    "全球星 2-R 第一次任务（9 颗）",
		"Demo Flight":                       "演示飞行",
		"The Grain Goddess Provides (iQPS Launch 7)": "谷物女神的馈赠（iQPS 第 7 次发射）",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	if strings.HasPrefix(value, "Starlink Group ") {
		return "星链第 " + strings.TrimPrefix(value, "Starlink Group ") + " 组"
	}
	return value
}

func translateStatus(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	known := map[string]string{
		"Go for Launch":     "准备发射",
		"To Be Determined":  "时间待定",
		"To Be Confirmed":   "等待确认",
		"Launch Successful": "发射成功",
		"Launch Failure":    "发射失败",
		"In Flight":         "飞行中",
		"Hold":              "暂缓发射",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	return "状态待确认"
}

func translateMissionType(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	known := map[string]string{
		"Unknown":           "任务类型待定",
		"Communications":    "通信任务",
		"Test Flight":       "试飞任务",
		"Navigation":        "导航任务",
		"Planetary Science": "行星科学任务",
		"Earth Science":     "地球科学任务",
		"Astrophysics":      "天体物理任务",
		"Technology":        "技术验证任务",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	return value
}

func translatePad(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	known := map[string]string{
		"Commercial LC-1":                  "商业一号发射工位",
		"Commercial LC-2":                  "商业二号发射工位",
		"Space Launch Complex 4E":          "4E 号航天发射复合体",
		"Space Launch Complex 40":          "40 号航天发射复合体",
		"Haiyang offshore launch location": "海阳海上发射点",
		"Orbital Launch Pad":               "轨道发射台",
		"Yoshinobu Launch Complex LP-2":    "吉信发射区第二发射台",
		"Ariane Launch Area 4":             "阿丽亚娜第四发射区",
		"Launch Complex 39A":               "39A 发射复合体",
		"Unknown Pad":                      "发射台待定",
		"Launch Area 96B":                  "96B 发射区",
		"ADD Offshore launch platform":     "ADD 海上发射平台",
		"Rocket Lab Launch Complex 1A":     "火箭实验室第一发射复合体 A 工位",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	return value
}

func translateLocation(value string) string {
	if value == "" || containsHan(value) {
		return value
	}
	known := map[string]string{
		"Wenchang Space Launch Site, People's Republic of China":      "中国文昌航天发射场",
		"Vandenberg SFB, CA, USA":                                     "美国加利福尼亚州范登堡太空军基地",
		"Haiyang Oriental Spaceport":                                  "中国海阳东方航天港",
		"Cape Canaveral SFS, FL, USA":                                 "美国佛罗里达州卡纳维拉尔角太空军基地",
		"Andøya Spaceport":                                            "挪威安岛航天港",
		"Tanegashima Space Center, Japan":                             "日本种子岛宇宙中心",
		"Guiana Space Centre, French Guiana":                          "法属圭亚那圭亚那航天中心",
		"Kennedy Space Center, FL, USA":                               "美国佛罗里达州肯尼迪航天中心",
		"Rocket Lab Launch Complex 1, Mahia Peninsula, New Zealand":   "新西兰马希亚半岛火箭实验室第一发射复合体",
		"Jiuquan Satellite Launch Center, People's Republic of China": "中国酒泉卫星发射中心",
		"Sea Launch": "海上发射平台",
	}
	if translated, ok := known[value]; ok {
		return translated
	}
	return value
}

func translateMissionDescription(event LaunchEvent) string {
	value := strings.TrimSpace(event.MissionDescription)
	if value == "" {
		return "任务详情尚未公布。"
	}
	if containsHan(value) {
		return value
	}
	name := event.MissionName
	if name == "Unknown Payload" || strings.EqualFold(value, "Details TBD.") {
		return "任务载荷和任务细节尚未公布。"
	}
	if strings.HasPrefix(name, "Starlink Group ") {
		if matches := starlinkCountPattern.FindStringSubmatch(value); len(matches) == 2 {
			return fmt.Sprintf("本次任务计划发射 %s 颗星链卫星，用于扩充 SpaceX 的全球卫星互联网星座。", matches[1])
		}
		return "本次任务计划发射一批星链卫星，用于扩充 SpaceX 的全球卫星互联网星座。"
	}
	known := map[string]string{
		"BlueBird 11-13 (Block 2 #6-8)":     "本次任务将发射 3 颗 BlueBird Block 2 通信卫星。该系列面向手机直连卫星宽带服务，单星配备大型通信阵列，目标峰值传输速度最高可达 120 Mbps。",
		"Onward and Upward":                 "这是 Isar Aerospace Spectrum 火箭的第二次试飞，将为欧洲航天局“Boost!”计划搭载 5 颗立方星和 1 项不可分离实验。",
		"Michibiki 7 (QZS-7)":               "日本准天顶卫星系统第七颗卫星。该系统通过倾斜椭圆地球同步轨道改善城市峡谷和山区的高仰角导航覆盖，并播发兼容 GPS 的导航与增强信号。",
		"Chang'e 7":                         "嫦娥七号计划于 2026 年发射，由轨道器、着陆器、飞跃器和月球车组成，目标是在月球南极开展环境探测，并重点寻找月壤中的水冰。",
		"MTG-I2":                            "欧洲气象卫星应用组织第三代气象卫星系列的第三颗卫星。",
		"Nancy Grace Roman Space Telescope": "这是 NASA 的红外空间望远镜，配备 2.4 米主镜、广域成像仪和日冕仪，主要用于系外行星、暗能量、宇宙膨胀与大尺度结构研究。",
		"LOXSAT 1":                          "LOXSAT 1 是由 Eta Space 研制、NASA Tipping Point 计划支持的在轨低温液氧管理技术验证卫星，将开展约 9 个月的低温流体储存与转移实验。",
		"StriX Launch 11":                   "为日本地球观测公司 Synspective 发射一颗合成孔径雷达卫星。",
		"Flight 2":                          "蓝箭航天朱雀三号火箭第二次试验发射，载荷信息尚待公布。",
		"Globalstar 2-R Mission 1 (x 9)":    "本次任务计划发射 9 颗第二代 Globalstar 补网卫星，用于更新其近地轨道全球移动通信星座。",
		"Demo Flight":                       "韩国军用小型卫星运载火箭的首次完整构型轨道试飞，火箭名称目前仍为暂定名称，更多任务细节尚待公布。",
		"The Grain Goddess Provides (iQPS Launch 7)": "为日本地球观测公司 iQPS 发射一颗合成孔径雷达地球观测卫星。",
	}
	if translated, ok := known[name]; ok {
		return translated
	}
	if event.MissionTypeZH != "" {
		return "这是一项" + event.MissionTypeZH + "，中文任务资料正在整理。"
	}
	return "任务详情的中文资料正在整理。"
}

func needsOriginal(event LaunchEvent) bool {
	pairs := [][2]string{
		{event.Name, event.NameZH},
		{event.StatusName, event.StatusNameZH},
		{event.PadName, event.PadNameZH},
		{event.LocationName, event.LocationNameZH},
		{event.MissionName, event.MissionNameZH},
		{event.MissionType, event.MissionTypeZH},
		{event.MissionDescription, event.MissionDescriptionZH},
	}
	for _, pair := range pairs {
		if pair[0] != "" && pair[0] != pair[1] && containsLatin(pair[0]) && !containsHan(pair[0]) {
			return true
		}
	}
	return false
}

func containsHan(value string) bool {
	for _, character := range value {
		if unicode.Is(unicode.Han, character) {
			return true
		}
	}
	return false
}

func containsLatin(value string) bool {
	for _, character := range value {
		if unicode.Is(unicode.Latin, character) {
			return true
		}
	}
	return false
}
