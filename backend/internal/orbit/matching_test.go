package orbit

import "testing"

func TestNormalizeSiteName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Vandenberg SFB, CA, USA", "vandenberg sfb ca usa"},
		{"Cape Canaveral SFS, FL, USA", "cape canaveral sfs fl usa"},
		{"Wenchang Space Launch Site, People's Republic of China", "wenchang space launch site people s republic of china"},
		{"Alcântara Space Center", "alcantara space center"},
		{"Guiana Space Centre, French Guiana", "guiana space centre french guiana"},
		{"LC-39A, Kennedy Space Center", "lc 39a kennedy space center"},
		{"Rocket Lab Launch Complex 1, Mahia Peninsula, New Zealand", "rocket lab launch complex 1 mahia peninsula new zealand"},
		{"", ""},
		{"  --==  --  ", ""},
	}
	for _, c := range cases {
		if got := normalizeSiteName(c.in); got != c.want {
			t.Errorf("normalizeSiteName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMatchLaunchSiteAlias(t *testing.T) {
	cases := []struct {
		key       string
		contained bool
		want      string
	}{
		// 位置名：前缀语义，最长前缀优先
		{"cape canaveral sfs fl usa", false, "cape-canaveral"},
		{"cape canaveral fl usa", false, "cape-canaveral"},
		{"vandenberg sfb ca usa", false, "vandenberg"},
		// 词边界：naro 不匹配 narok；vandenberg 单独由站点全名分支捕获
		{"naro space center", false, "naro"},
		{"narok somalia", false, ""},
		{"vandenberg", false, ""},
		{"vandenberga", false, ""},
		// 工位名：包含语义（站点名常在 pad 末尾/中部）
		{"lc 39a kennedy space center", true, ""}, // 由站点全名包含分支捕获（非别名）
		{"slc 4e vandenberg sfb", true, "vandenberg"},
		{"space launch complex 40 cape canaveral sfs", true, "cape-canaveral"},
		{"spacex landing zone 1", true, ""},
		{"", false, ""},
	}
	for _, c := range cases {
		if got := matchLaunchSiteAlias(c.key, c.contained); got != c.want {
			t.Errorf("matchLaunchSiteAlias(%q, contained=%v) = %q, want %q", c.key, c.contained, got, c.want)
		}
	}
}

func TestSiteNameMatchPredicates(t *testing.T) {
	wenchang := "wenchang space launch site"
	kennedy := "kennedy space center"
	cases := []struct {
		name      string
		key       string
		siteNorm  string
		contained bool
		want      bool
	}{
		{"location 前缀命中", "wenchang space launch site people s republic of china", wenchang, false, true},
		{"location 前缀未命中", "xichang satellite launch center china", wenchang, false, false},
		{"location 词边界（不含尾部空格）", "wenchang space launch sites", wenchang, false, false},
		{"pad 包含命中（站点名在末尾）", "lc 39a kennedy space center", kennedy, true, true},
		{"pad 包含命中（站点名在开头）", "kennedy space center launch complex 39a", kennedy, true, true},
		{"pad 包含命中（站点名在中部）", "lc 39a kennedy space center pad b", kennedy, true, true},
		{"pad 包含词边界", "kennedy space centers museum", kennedy, true, false},
		{"pad 未命中", "space launch complex 40", kennedy, true, false},
	}
	for _, c := range cases {
		got := siteNamePrefixMatch(c.key, c.siteNorm)
		if c.contained {
			got = siteNameContainedMatch(c.key, c.siteNorm)
		}
		if got != c.want {
			t.Errorf("%s: match(%q, %q, contained=%v) = %v, want %v", c.name, c.key, c.siteNorm, c.contained, got, c.want)
		}
	}
}

func TestFoldDiacritic(t *testing.T) {
	cases := map[rune]rune{
		'â': 'a', 'ç': 'c', 'é': 'e', 'í': 'i', 'ñ': 'n', 'ô': 'o', 'ü': 'u', 'ÿ': 'y',
		'a': 'a', // 非变音符原样返回
	}
	for in, want := range cases {
		if got := foldDiacritic(in); got != want {
			t.Errorf("foldDiacritic(%q) = %q, want %q", in, got, want)
		}
	}
}
