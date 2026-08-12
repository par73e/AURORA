package dailyimage

// CuratedWindows is deliberately a local editorial registry. The European
// galleries are not scraped at runtime: every entry keeps its official page,
// direct screen-size asset and credit line together for review.
func CuratedWindows() []ImageWindow {
	return []ImageWindow{
		sourceWindow(
			"eso", "eso", "ESO Picture of the Week", "A very hungry planet", "2025-08-26",
			"https://cdn.eso.org/images/screen/potw2534a.jpg", "ESO/R. F. van Capelleveen et al.",
			"CC BY 4.0；使用时须保留完整署名", "https://www.eso.org/public/images/potw2534a/",
			"VLT 首次清晰捕捉到一颗仍嵌在多环原行星盘中的年轻行星。",
		),
		sourceWindow(
			"eso-potw2533", "eso", "ESO Picture of the Week", "Milky Way spectator", "2025-08-18",
			"https://cdn.eso.org/images/screen/potw2533a.jpg", "C. Letelier/ESO",
			"CC BY 4.0；使用时须保留完整署名", "https://www.eso.org/public/images/potw2533a/",
			"银河横跨正在建设中的极大望远镜，连接地面观测与深空。",
		),
		sourceWindow(
			"eso-potw2532", "eso", "ESO Picture of the Week", "Pointing out the stars above the residencia", "2025-08-11",
			"https://cdn.eso.org/images/screen/potw2532a.jpg", "ESO/N. Schafer",
			"CC BY 4.0；使用时须保留完整署名", "https://www.eso.org/public/images/potw2532a/",
			"VLT 的激光导星系统从帕瑞纳天文台直指夜空。",
		),
		sourceWindow(
			"esa-webb", "esa-webb", "ESA/Webb Picture of the Month", "A starburst shines in infrared", "2025-06-30",
			"https://cdn.esawebb.org/archives/images/screen/potm2506a.jpg", "ESA/Webb, NASA & CSA, A. Bolatto",
			"CC BY 4.0；使用时须保留完整署名", "https://esawebb.org/images/potm2506a/",
			"韦布以近红外视角展示雪茄星系 M82 中被尘埃遮蔽的旺盛恒星形成。",
		),
		sourceWindow(
			"esa-webb-potm2507", "esa-webb", "ESA/Webb Picture of the Month", "A fresh look at a classic deep field", "2025-08-01",
			"https://cdn.esawebb.org/archives/images/screen/potm2507a.jpg", "ESA/Webb, NASA & CSA, G. Östlin, P. G. Perez-Gonzalez, J. Melinder, the JADES Collaboration, the MIDIS collaboration, M. Zamani (ESA/Webb)",
			"CC BY 4.0；使用时须保留完整署名", "https://esawebb.org/images/potm2507a/",
			"韦布重新凝视经典深场，在狭小视野中显现数千个遥远星系。",
		),
		sourceWindow(
			"esa-webb-potm2508", "esa-webb", "ESA/Webb Picture of the Month", "Dusty wisps round a dusty disc", "2025-08-29",
			"https://cdn.esawebb.org/archives/images/screen/potm2508a.jpg", "ESA/Webb, NASA & CSA, M. Villenave et al.",
			"CC BY 4.0；使用时须保留完整署名", "https://esawebb.org/images/potm2508a/",
			"一面近乎侧向的原行星盘，让尘埃演化与行星形成的早期过程显形。",
		),
		sourceWindow(
			"esa-hubble", "esa-hubble", "ESA/Hubble Picture of the Week", "The smouldering heart of a celestial cigar", "2025-09-15",
			"https://cdn.esahubble.org/archives/images/screen/potw2537a.jpg", "ESA/Hubble & NASA, W. D. Vacca",
			"CC BY 4.0；使用时须保留完整署名", "https://esahubble.org/images/potw2537a/",
			"哈勃聚焦 M82 的炽热核心，显现出恒星形成区与交错的气体尘埃带。",
		),
		sourceWindow(
			"esa-hubble-potw2533", "esa-hubble", "ESA/Hubble Picture of the Week", "Noteworthy nearby spiral", "2025-08-18",
			"https://cdn.esahubble.org/archives/images/screen/potw2533a.jpg", "ESA/Hubble & NASA, R. Chandar, J. Lee and the PHANGS-HST team",
			"CC BY 4.0；使用时须保留完整署名", "https://esahubble.org/images/potw2533a/",
			"近邻螺旋星系 NGC 2835 的星臂中，蓝色恒星、粉红色星形成区与尘埃带并存。",
		),
	}
}
