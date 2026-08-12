package dailyimage

import "time"

// CuratedWindows is deliberately a local editorial registry. The European
// galleries are not scraped at runtime: every entry keeps its official page,
// direct screen-size asset and credit line together for review.
func CuratedWindows(_ time.Time) []ImageWindow {
	return []ImageWindow{
		sourceWindow(
			"eso", "eso", "ESO Picture of the Week", "A very hungry planet", "2025-08-26",
			"https://cdn.eso.org/images/screen/potw2534a.jpg", "ESO/R. F. van Capelleveen et al.",
			"CC BY 4.0；使用时须保留完整署名", "https://www.eso.org/public/images/potw2534a/",
			"VLT 首次清晰捕捉到一颗仍嵌在多环原行星盘中的年轻行星。",
		),
		sourceWindow(
			"esa-webb", "esa-webb", "ESA/Webb Picture of the Month", "A starburst shines in infrared", "2025-06-30",
			"https://cdn.esawebb.org/archives/images/screen/potm2506a.jpg", "ESA/Webb, NASA & CSA, A. Bolatto",
			"CC BY 4.0；使用时须保留完整署名", "https://esawebb.org/images/potm2506a/",
			"韦布以近红外视角展示雪茄星系 M82 中被尘埃遮蔽的旺盛恒星形成。",
		),
		sourceWindow(
			"esa-hubble", "esa-hubble", "ESA/Hubble Picture of the Week", "The smouldering heart of a celestial cigar", "2025-09-15",
			"https://cdn.esahubble.org/archives/images/screen/potw2537a.jpg", "ESA/Hubble & NASA, W. D. Vacca",
			"CC BY 4.0；使用时须保留完整署名", "https://esahubble.org/images/potw2537a/",
			"哈勃聚焦 M82 的炽热核心，显现出恒星形成区与交错的气体尘埃带。",
		),
	}
}
