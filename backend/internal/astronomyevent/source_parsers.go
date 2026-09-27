package astronomyevent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// parseUSNOMoonPhases 解析 USNO 月相 JSON，返回用于交叉校验的事件。
// USNO 仅作为本地计算的校验源，不覆盖本地计算结果。
func parseUSNOMoonPhases(_ context.Context, body []byte, sourceURL string, year int) (parsedEvents, error) {
	var payload struct {
		Year      int `json:"year"`
		Phasedata []struct {
			Day   int    `json:"day"`
			Month int    `json:"month"`
			Phase string `json:"phase"`
			Time  string `json:"time"`
			Year  int    `json:"year"`
		} `json:"phasedata"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return parsedEvents{}, fmt.Errorf("parse USNO JSON: %w", err)
	}
	payloadJSON := json.RawMessage(body)
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	var events []Event
	for _, item := range payload.Phasedata {
		yearValue := item.Year
		if yearValue == 0 {
			yearValue = payload.Year
		}
		at, err := time.ParseInLocation("2006-1-2 15:04", fmt.Sprintf("%d-%d-%d %s", yearValue, item.Month, item.Day, item.Time), time.UTC)
		if err != nil {
			// USNO date format may differ; try alternate parse
			continue
		}
		kind := usnoPhaseKind(item.Phase)
		if kind == "" {
			continue
		}
		events = append(events, Event{
			ID:         fmt.Sprintf("usno-%s-%s", kind, at.UTC().Format("20060102")),
			Kind:       kind,
			Title:      usnoPhaseTitle(kind),
			TitleEN:    strings.ToUpper(strings.ReplaceAll(kind, "_", " ")),
			StartsAt:   at.UTC(),
			DateLabel:  at.UTC().Format("2006年1月2日"),
			Summary:    "USNO 月相校验数据。",
			Origin:     "external_forecast",
			SourceCode: "usno_astronomy",
			SourceURL:  sourceURL,
			VerifiedAt: time.Now().UTC().Format(time.DateOnly),
			Geometry:   json.RawMessage(fmt.Sprintf(`{"phase":%q,"precision":"usno_cross_check"}`, item.Phase)),
		})
	}
	return parsedEvents{events: events, payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

func usnoPhaseKind(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "new moon":
		return "new_moon"
	case "first quarter":
		return "first_quarter"
	case "full moon":
		return "full_moon"
	case "last quarter":
		return "last_quarter"
	}
	return ""
}

func usnoPhaseTitle(kind string) string {
	switch kind {
	case "new_moon":
		return "新月"
	case "first_quarter":
		return "上弦月"
	case "full_moon":
		return "满月"
	case "last_quarter":
		return "下弦月"
	}
	return kind
}

// parseUSNOSeasons 解析 USNO 年度季节/近日远日点资料。AURORA 的季节节点
// 仍由本地模型生成；这里保存可审计的交叉校验资料，而不制造重复日历事件。
func parseUSNOSeasons(_ context.Context, body []byte, _ string, year int) (parsedEvents, error) {
	var payload struct {
		Year int `json:"year"`
		Data []struct {
			Day    int    `json:"day"`
			Month  int    `json:"month"`
			Phenom string `json:"phenom"`
			Time   string `json:"time"`
			Year   int    `json:"year"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return parsedEvents{}, fmt.Errorf("parse USNO seasons JSON: %w", err)
	}
	coverageYear := payload.Year
	if coverageYear == 0 {
		coverageYear = year
	}
	coverageStart := time.Date(coverageYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	return parsedEvents{payload: body, payloadJSON: json.RawMessage(body), coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

// parseNASAGSFCEclipsePage 解析 NASA/GSFC 食目录页面，提取日食/月食事件。
// NASA/GSFC 页面是 HTML 表格，每行含 <td>YYYY Mon DD</td><td>Total/Annular/Partial/Hybrid</td>
// <td>saros</td><td>食分</td><td>时长</td><td>...path...</td>。
// 本函数用正则提取每行的日期、食类型、食分、saros、路径链接，写 external_forecast 事件。
func parseNASAGSFCEclipsePage(_ context.Context, body []byte, sourceURL string) (parsedEvents, error) {
	text := string(body)
	// NASA 页面是 HTML，不能直接作为 json.RawMessage 传给 JSONB 快照。
	// 留空后由 saveSnapshot 封装为带原文 base64 的可审计 JSON。
	var payloadJSON json.RawMessage
	// 每行 <tr ...> <td>YYYY Mon DD</td> <td>Type</td> <td>...saros...</td> <td>食分</td> <td>时长</td> <td>...path...</td> ... </tr>
	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>.*?</tr>`)
	dateRegex := regexp.MustCompile(`(\d{4})\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})`)
	typeRegex := regexp.MustCompile(`(?s)<td[^>]*>\s*(?:<a[^>]*>)?\s*(Total|Annular|Partial|Hybrid|Penumbral)\s*(?:</a>)?\s*</td>`)
	magnitudeRegex := regexp.MustCompile(`<td[^>]*>\s*(\d\.\d+)\s*</td>`)
	timeRegex := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})`)
	sarosRegex := regexp.MustCompile(`SEsaros/SEsaros(\d+)`)
	pathRegex := regexp.MustCompile(`(SEpath/SEpath\d+/SE\d+[A-Za-z]+\d+[A-Za-z]?path\.html|LEplot/LE\d+[A-Za-z]+\d+\.GIF|LEdecade/LE\d+\.html)`)
	year := time.Now().UTC().Year()
	var events []Event
	for _, row := range rowRegex.FindAllString(text, -1) {
		dateMatch := dateRegex.FindStringSubmatch(row)
		if dateMatch == nil {
			continue
		}
		yearStr, monthStr, dayStr := dateMatch[1], dateMatch[2], dateMatch[3]
		eventYear, _ := strconv.Atoi(yearStr)
		month := monthFromAbbr(monthStr)
		day, _ := strconv.Atoi(dayStr)
		if eventYear < year-1 || eventYear > year+18 || month == 0 || day == 0 {
			continue
		}
		// 食类型
		eclipseType := "Partial"
		if tm := typeRegex.FindStringSubmatch(row); tm != nil {
			eclipseType = tm[1]
		}
		// 食分
		magnitude := ""
		if mm := magnitudeRegex.FindStringSubmatch(row); mm != nil {
			magnitude = mm[1]
		}
		// saros
		saros := ""
		if sm := sarosRegex.FindStringSubmatch(row); sm != nil {
			saros = sm[1]
		}
		// 路径链接
		pathURL := ""
		if pm := pathRegex.FindStringSubmatch(row); pm != nil {
			pathURL = "https://eclipse.gsfc.nasa.gov/" + strings.TrimPrefix(pm[1], "../")
		}

		// 判断日食/月食：SEpath 是日食，LEdecade/LEplot 是月食
		kind := "solar_eclipse"
		titlePrefix := "日食"
		if strings.Contains(sourceURL, "LEdecade") || strings.Contains(sourceURL, "LEplot") {
			kind = "lunar_eclipse"
			titlePrefix = "月食"
		}
		hour, minute, second := 12, 0, 0
		if timeMatch := timeRegex.FindStringSubmatch(row); timeMatch != nil {
			hour, _ = strconv.Atoi(timeMatch[1])
			minute, _ = strconv.Atoi(timeMatch[2])
			second, _ = strconv.Atoi(timeMatch[3])
		}
		at := time.Date(eventYear, month, day, hour, minute, second, 0, time.UTC)
		// 食类型缩写：T=Total, A=Annular, P=Partial, H=Hybrid
		title := fmt.Sprintf("%s·%s", eclipseType, titlePrefix)
		titleEN := strings.ToUpper(strings.ReplaceAll(kind, "_", " ")) + " " + eclipseType
		geometry := map[string]any{
			"precision":   "nasa_gsfc_table",
			"eclipseType": eclipseType,
		}
		if magnitude != "" {
			geometry["magnitude"] = magnitude
		}
		if saros != "" {
			geometry["saros"] = saros
		}
		if pathURL != "" {
			geometry["pathUrl"] = pathURL
		}
		geomRaw, _ := json.Marshal(geometry)
		events = append(events, Event{
			ID:         fmt.Sprintf("nasa-gsfc-%s-%s", kind, at.UTC().Format("20060102")),
			Kind:       kind,
			Title:      title,
			TitleEN:    titleEN,
			StartsAt:   at.UTC(),
			DateLabel:  at.UTC().Format("2006年1月2日"),
			Summary:    fmt.Sprintf("NASA/GSFC 食目录：%s型%s。", eclipseType, titlePrefix),
			Origin:     "external_forecast",
			SourceCode: "nasa_gsfc_eclipse",
			SourceURL:  sourceURL,
			VerifiedAt: time.Now().UTC().Format(time.DateOnly),
			Geometry:   geomRaw,
		})
	}
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	return parsedEvents{events: events, payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

func monthFromAbbr(abbr string) time.Month {
	months := map[string]time.Month{
		"Jan": time.January, "Feb": time.February, "Mar": time.March,
		"Apr": time.April, "May": time.May, "Jun": time.June,
		"Jul": time.July, "Aug": time.August, "Sep": time.September,
		"Oct": time.October, "Nov": time.November, "Dec": time.December,
	}
	return months[abbr]
}

// parseIMOMeteorCalendar 解析 IMO 年度年历 PDF 中的 Working List。RSS 的发布时间
// 不是流星雨极大时刻，故 RSS 只做快照，绝不调用本函数伪造预测事件。
func parseIMOMeteorCalendar(_ context.Context, body []byte, sourceURL string, year int) (parsedEvents, error) {
	reader, err := pdf.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return parsedEvents{payload: body}, fmt.Errorf("open IMO calendar PDF: %w", err)
	}
	plainText, err := reader.GetPlainText()
	if err != nil {
		return parsedEvents{payload: body}, fmt.Errorf("extract IMO calendar PDF text: %w", err)
	}
	text, err := io.ReadAll(plainText)
	if err != nil {
		return parsedEvents{payload: body}, fmt.Errorf("read IMO calendar PDF text: %w", err)
	}
	events := parseIMOCalendarText(string(text), sourceURL, year)
	if len(events) == 0 {
		return parsedEvents{payload: body}, fmt.Errorf("IMO calendar contains no parseable annual shower records")
	}

	// PDF 不是 JSON；快照统一由 saveSnapshot 封装，并包含完整原文与内容哈希。
	var payloadJSON json.RawMessage
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	return parsedEvents{events: events, payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

// parseIMORSS 保存 IMO 动态资讯版本，但不把文章发布时间误当成流星雨峰值。
func parseIMORSS(_ context.Context, body []byte, _ string, year int) (parsedEvents, error) {
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	return parsedEvents{payload: body, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

var imoShowerPattern = regexp.MustCompile(`(?s)([^()]{1,80})\((\d{3}[A-Z]{3})\)Active:([A-Za-z]{3,9}\d{1,2}[–-](?:[A-Za-z]{3,9})?\d{1,2}(?:\(\?\))?);Maximum:([^;]+);ZHR(?:=|≈|variable,usually≈)([^;]+);Radiant:α=([0-9]{1,3})◦,δ=([+−-]?[0-9]{1,3})◦`)
var imoMaximumDatePattern = regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)(?:≈)?(\d{1,2})(?:,(\d{1,2})h(?:(\d{2})m)?)?`)

// parseIMOCalendarText 从 IMO 提取后的纯文本中读取年度 Working List。它显式保留
// 资料给出的活动期、ZHR 和辐射点；不确定的峰值时间只使用该资料明确给出的日期。
func parseIMOCalendarText(text, sourceURL string, year int) []Event {
	var events []Event
	seen := make(map[string]struct{})
	for _, match := range imoShowerPattern.FindAllStringSubmatch(text, -1) {
		code := match[2]
		if _, duplicate := seen[code]; duplicate {
			continue
		}
		peak := imoMaximumDatePattern.FindStringSubmatch(match[4])
		if peak == nil {
			continue
		}
		month := fullMonth(peak[1])
		day, _ := strconv.Atoi(peak[2])
		if month == 0 || day == 0 {
			continue
		}
		hour, minute := 0, 0
		if peak[3] != "" {
			hour, _ = strconv.Atoi(peak[3])
			if peak[4] != "" {
				minute, _ = strconv.Atoi(peak[4])
			}
		}
		start, end, ok := imoActivityRange(match[3], year)
		if !ok {
			continue
		}
		at := time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
		ra, raErr := strconv.ParseFloat(match[6], 64)
		decText := strings.ReplaceAll(match[7], "−", "-")
		dec, decErr := strconv.ParseFloat(decText, 64)
		if raErr != nil || decErr != nil {
			continue
		}
		slug, title := imoShowerName(code, match[1])
		geometry, _ := json.Marshal(map[string]any{
			"precision":     "imo_calendar_pdf",
			"iauCode":       code,
			"slug":          slug,
			"activityStart": start.Format(time.DateOnly),
			"activityEnd":   end.Format(time.DateOnly),
			"zhr":           strings.TrimSpace(match[5]),
			"radiantRA":     ra,
			"radiantDec":    dec,
		})
		endsAt := end.AddDate(0, 0, 1)
		events = append(events, Event{
			ID:         fmt.Sprintf("imo-meteor_shower-%s-%s", strings.ToLower(code), at.Format("20060102")),
			Kind:       "meteor_shower",
			Title:      title + "极大",
			TitleEN:    strings.ToUpper(strings.TrimSpace(match[1])) + " MAXIMUM",
			StartsAt:   at,
			EndsAt:     &endsAt,
			DateLabel:  at.Format("2006年1月2日"),
			Summary:    fmt.Sprintf("IMO %d 年度流星雨年历：活动期 %s 至 %s，ZHR %s。", year, start.Format("1月2日"), end.Format("1月2日"), strings.TrimSpace(match[5])),
			Origin:     "external_forecast",
			SourceCode: "imo_meteor_calendar",
			SourceURL:  sourceURL,
			VerifiedAt: time.Now().UTC().Format(time.DateOnly),
			Geometry:   geometry,
		})
		seen[code] = struct{}{}
	}
	return events
}

func fullMonth(value string) time.Month {
	months := map[string]time.Month{"january": time.January, "february": time.February, "march": time.March, "april": time.April, "may": time.May, "june": time.June, "july": time.July, "august": time.August, "september": time.September, "october": time.October, "november": time.November, "december": time.December}
	return months[strings.ToLower(value)]
}

func imoActivityRange(value string, year int) (time.Time, time.Time, bool) {
	value = strings.TrimSuffix(value, "(?)")
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '–' || r == '-' })
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, false
	}
	activityPart := regexp.MustCompile(`^([A-Za-z]{3,9})?(\d{1,2})$`)
	parse := func(part string, fallbackMonth time.Month) (time.Month, int, bool) {
		match := activityPart.FindStringSubmatch(part)
		if match == nil {
			return 0, 0, false
		}
		month := fallbackMonth
		if match[1] != "" {
			month = fullMonth(match[1])
		}
		if month == 0 && len(match[1]) >= 3 {
			month = monthFromAbbr(match[1][:3])
		}
		day, err := strconv.Atoi(match[2])
		return month, day, month != 0 && err == nil
	}
	startMonth, startDay, startOK := parse(parts[0], 0)
	endMonth, endDay, endOK := parse(parts[1], startMonth)
	if !startOK || !endOK {
		return time.Time{}, time.Time{}, false
	}
	start := time.Date(year, startMonth, startDay, 0, 0, 0, 0, time.UTC)
	endYear := year
	if endMonth < startMonth {
		endYear++
	}
	end := time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, time.UTC)
	return start, end, true
}

func imoShowerName(code, fallback string) (string, string) {
	known := map[string][2]string{
		"007PER": {"perseids", "英仙座流星雨"}, "012KCG": {"kappa-cygnids", "天鹅座κ流星雨"},
		"206AUR": {"aurigids", "御夫座流星雨"}, "008ORI": {"orionids", "猎户座流星雨"},
		"013LEO": {"leonids", "狮子座流星雨"}, "004GEM": {"geminids", "双子座流星雨"},
		"015URS": {"ursids", "小熊座流星雨"}, "010QUA": {"quadrantids", "象限仪座流星雨"},
		"006LYR": {"lyrids", "天琴座流星雨"}, "031ETA": {"eta-aquariids", "宝瓶座η流星雨"},
		"005SDA": {"delta-aquariids", "宝瓶座δ南流星雨"}, "001CAP": {"capricornids", "摩羯座α流星雨"},
	}
	if value, ok := known[code]; ok {
		return value[0], value[1]
	}
	name := strings.TrimSpace(fallback)
	if index := strings.LastIndexAny(name, ".;。\n"); index >= 0 {
		name = strings.TrimSpace(name[index+1:])
	}
	return strings.ToLower(code), name
}

// meteorShowerChineseName 返回流星雨的中文名称（按 IMO slug）。
func meteorShowerChineseName(slug string) string {
	names := map[string]string{
		"perseids":        "英仙座流星雨",
		"kappa-cygnids":   "天鹅座κ流星雨",
		"aurigids":        "御夫座流星雨",
		"orionids":        "猎户座流星雨",
		"leonids":         "狮子座流星雨",
		"geminids":        "双子座流星雨",
		"ursids":          "小熊座流星雨",
		"quadrantids":     "象限仪座流星雨",
		"taurs":           "金牛座流星雨",
		"lyrids":          "天琴座流星雨",
		"eta-aquariids":   "宝瓶座η流星雨",
		"delta-aquariids": "宝瓶座δ流星雨",
		"capricornids":    "摩羯座流星雨",
	}
	if name, ok := names[slug]; ok {
		return name
	}
	return slug
}

// parseNASAEclipsePathPage 解析 NASA/GSFC 路径页（SEpath/SEYYYYMonDD{T|A|P}path.html），
// 提取 Besselian 路径元素：食类型、日期、中央线坐标序列、食分、太阳高度/方位角、路径宽度、持续时间。
// 路径页标题含 "Total/Annular Solar Eclipse of YYYY Mon DD"，
// 正文是 120 秒间隔的路径表，每行含中央线/北限/南限坐标 + 食分/太阳高度/方位角/路径宽度/持续时间。
// 本函数把路径元素解析为 geometry.pathWaypoints 数组，写 external_forecast 事件。
func parseNASAEclipsePathPage(_ context.Context, body []byte, sourceURL string) (parsedEvents, error) {
	text := string(body)
	// NASA 路径页是 HTML，不是 JSON；快照统一由 saveSnapshot 封装。
	var payloadJSON json.RawMessage
	// 从标题提取食类型与日期
	titleRegex := regexp.MustCompile(`(?s)(Total|Annular|Partial|Hybrid)\s+Solar\s+Eclipse\s+of\s+(\d{4})\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})`)
	titleMatch := titleRegex.FindStringSubmatch(text)
	if titleMatch == nil {
		// 不是路径页；只保存快照
		year := time.Now().UTC().Year()
		coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		coverageEnd := coverageStart.AddDate(1, 0, 0)
		return parsedEvents{payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
	}
	eclipseType := titleMatch[1]
	eventYear, _ := strconv.Atoi(titleMatch[2])
	month := monthFromAbbr(titleMatch[3])
	day, _ := strconv.Atoi(titleMatch[4])
	at := time.Date(eventYear, month, day, 12, 0, 0, 0, time.UTC)

	// 提取路径表行：每行以 UTC 时刻开头，后跟坐标
	// 格式：HH:MM   lat lat lon lon lat lat lon lon ratio alt azm width dur
	rowRegex := regexp.MustCompile(`(?m)^\s*(\d{2}):(\d{2})\s+(-?\d+\s+\d+\.\d+[NS])\s+(\d+\s+\d+\.\d+[EW])\s+(-?\d+\s+\d+\.\d+[NS])\s+(\d+\s+\d+\.\d+[EW])\s+(-?\d+\s+\d+\.\d+[NS])\s+(\d+\s+\d+\.\d+[EW])\s+(\d\.\d+)\s+(-?\d+)\s+(-?\d+)\s+(\d+)\s+(\d+m\d+\.\d+s)`)
	waypoints := make([]map[string]any, 0, 64)
	for _, m := range rowRegex.FindAllStringSubmatch(text, -1) {
		hour, _ := strconv.Atoi(m[1])
		minute, _ := strconv.Atoi(m[2])
		waypointTime := time.Date(eventYear, month, day, hour, minute, 0, 0, time.UTC)
		ratio, _ := strconv.ParseFloat(m[9], 64)
		altitude, _ := strconv.Atoi(m[10])
		azimuth, _ := strconv.Atoi(m[11])
		widthKm, _ := strconv.Atoi(m[12])
		waypoints = append(waypoints, map[string]any{
			"time":        waypointTime.UTC().Format("2006-01-02T15:04:05"),
			"northLimit":  m[3] + " " + m[4],
			"southLimit":  m[5] + " " + m[6],
			"centralLine": m[7] + " " + m[8],
			"ratio":       ratio,
			"altitude":    altitude,
			"azimuth":     azimuth,
			"widthKm":     widthKm,
			"duration":    m[13],
		})
	}

	geometry := map[string]any{
		"precision":   "nasa_gsfc_besselian_path",
		"eclipseType": eclipseType,
		"pathUrl":     sourceURL,
	}
	if len(waypoints) > 0 {
		geometry["pathWaypoints"] = waypoints
		geometry["waypointCount"] = len(waypoints)
	}
	geomRaw, _ := json.Marshal(geometry)

	event := Event{
		ID:         fmt.Sprintf("nasa-gsfc-solar_eclipse-%s", at.UTC().Format("20060102")),
		Kind:       "solar_eclipse",
		Title:      fmt.Sprintf("%s·日食", eclipseType),
		TitleEN:    "SOLAR ECLIPSE " + strings.ToUpper(eclipseType),
		StartsAt:   at.UTC(),
		DateLabel:  at.UTC().Format("2006年1月2日"),
		Summary:    fmt.Sprintf("NASA/GSFC 路径页：%s型日食，含 Besselian 路径元素（%d 个路点）。", eclipseType, len(waypoints)),
		Origin:     "external_forecast",
		SourceCode: "nasa_gsfc_eclipse",
		SourceURL:  sourceURL,
		VerifiedAt: time.Now().UTC().Format(time.DateOnly),
		Geometry:   geomRaw,
	}
	coverageStart := time.Date(eventYear, month, day, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(0, 0, 1)
	return parsedEvents{events: []Event{event}, payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

// parseIAUMDC 导入 IAU MDC 的 established-shower 目录版本到资料快照。
// 目录提供名称、编号和辐射点等基础参数，但没有年度峰值和 ZHR 预测；不能凭内置
// 常量生成伪造的 external_forecast 事件。标准化目录随原始内容一起留在快照中，
// 供年度 IMO 年历解析与人工核验关联使用。
func parseIAUMDC(_ context.Context, body []byte, _ string, year int) (parsedEvents, error) {
	// IAU MDC 目录为 pipe 分隔文本，不是 JSON；快照统一由 saveSnapshot 封装。
	payloadJSON := iauMDCSnapshotPayload(body)
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(1, 0, 0)
	return parsedEvents{payload: body, payloadJSON: payloadJSON, coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}

// iauMDCSnapshotPayload 给原始目录补充一个小型、可查询的标准化索引，原文仍由
// sourceSnapshotPayload 的 base64 字段完整保留，内容哈希仍基于原始响应。
func iauMDCSnapshotPayload(body []byte) json.RawMessage {
	payload := sourceSnapshotPayload(body, "text/plain; charset=utf-8")
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return payload
	}
	document["format"] = "iau_mdc_established_showers"
	document["records"] = parseIAUMDCRecords(string(body))
	encoded, err := json.Marshal(document)
	if err != nil {
		return payload
	}
	return encoded
}

func parseIAUMDCRecords(text string) []map[string]any {
	seen := make(map[string]struct{})
	records := make([]map[string]any, 0)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 13 {
			continue
		}
		for index := range fields {
			fields[index] = strings.Trim(strings.TrimSpace(fields[index]), `"`)
		}
		iauNumber, code, designation := fields[1], fields[3], fields[6]
		if iauNumber == "" || code == "" || designation == "" {
			continue
		}
		key := iauNumber + ":" + code
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		ra, raErr := strconv.ParseFloat(fields[11], 64)
		dec, decErr := strconv.ParseFloat(fields[12], 64)
		if raErr != nil || decErr != nil {
			continue
		}
		seen[key] = struct{}{}
		records = append(records, map[string]any{
			"iauNumber":      iauNumber,
			"code":           code,
			"designation":    designation,
			"radiantRA":      ra,
			"radiantDec":     dec,
			"solarLongitude": strings.TrimSpace(fields[10]),
			"activity":       strings.TrimSpace(fields[7]),
		})
	}
	return records
}

// parseJPLCloseApproaches 解析 CNEOS close-approach API，收录未来 18 个月内距离
// 地球 0.05 AU 的小天体近掠。CNEOS 返回 fields/data 表格，这里按字段名而非列号读取。
func parseJPLCloseApproaches(_ context.Context, body []byte, sourceURL string, year int) (parsedEvents, error) {
	var payload struct {
		Fields []string            `json:"fields"`
		Data   [][]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return parsedEvents{}, fmt.Errorf("parse JPL close-approach JSON: %w", err)
	}
	indexes := make(map[string]int, len(payload.Fields))
	for index, field := range payload.Fields {
		indexes[field] = index
	}
	for _, required := range []string{"des", "cd", "dist"} {
		if _, ok := indexes[required]; !ok {
			return parsedEvents{payload: body, payloadJSON: json.RawMessage(body)}, fmt.Errorf("JPL close-approach data missing %s", required)
		}
	}
	stringAt := func(row []json.RawMessage, field string) string {
		index, ok := indexes[field]
		if !ok || index >= len(row) {
			return ""
		}
		var value string
		_ = json.Unmarshal(row[index], &value)
		return value
	}
	var events []Event
	for _, row := range payload.Data {
		designation, dateText, distanceText := stringAt(row, "des"), stringAt(row, "cd"), stringAt(row, "dist")
		at, err := time.Parse("2006-Jan-02 15:04", dateText)
		if err != nil || designation == "" {
			continue
		}
		distance, err := strconv.ParseFloat(distanceText, 64)
		if err != nil {
			continue
		}
		geometry := map[string]any{"precision": "jpl_cneos_cad", "designation": designation, "distanceAU": distance}
		for _, field := range []string{"dist_min", "dist_max", "v_rel", "v_inf", "h", "orbit_id", "t_sigma_f"} {
			if value := stringAt(row, field); value != "" {
				geometry[field] = value
			}
		}
		geomRaw, _ := json.Marshal(geometry)
		identifier := strings.NewReplacer(" ", "-", "/", "-", "(", "", ")", "").Replace(strings.ToLower(designation))
		events = append(events, Event{
			ID:         "jpl-cneos-close-approach-" + identifier + "-" + at.Format("20060102"),
			Kind:       "small_body_close_approach",
			Title:      designation + " 近地掠过",
			TitleEN:    designation + " CLOSE APPROACH",
			StartsAt:   at.UTC(),
			DateLabel:  at.UTC().Format("2006年1月2日"),
			Summary:    fmt.Sprintf("JPL CNEOS 预测该小天体将以 %.4f AU 的名义最近距离掠过地球。", distance),
			Origin:     "external_forecast",
			SourceCode: "jpl_small_bodies",
			SourceURL:  sourceURL,
			VerifiedAt: time.Now().UTC().Format(time.DateOnly),
			Geometry:   geomRaw,
		})
	}
	coverageStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	coverageEnd := coverageStart.AddDate(0, 18, 0)
	return parsedEvents{events: events, payload: body, payloadJSON: json.RawMessage(body), coverageStart: &coverageStart, coverageEnd: &coverageEnd}, nil
}
