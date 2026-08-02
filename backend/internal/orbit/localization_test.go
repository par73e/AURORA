package orbit

import "testing"

func TestLocalizeEnglishLaunchEvent(t *testing.T) {
	event := LaunchEvent{
		Name:               "Falcon 9 Block 5 | Starlink Group 17-53",
		StatusName:         "Go for Launch",
		PadName:            "Space Launch Complex 4E",
		LocationName:       "Vandenberg SFB, CA, USA",
		MissionName:        "Starlink Group 17-53",
		MissionType:        "Communications",
		MissionDescription: "A batch of 24 satellites for the Starlink mega-constellation - SpaceX's project for space-based Internet communication system.",
	}

	LocalizeLaunchEvent(&event)

	if event.NameZH != "猎鹰9号 Block 5｜星链第 17-53 组" {
		t.Fatalf("unexpected Chinese name: %q", event.NameZH)
	}
	if event.StatusNameZH != "准备发射" {
		t.Fatalf("unexpected Chinese status: %q", event.StatusNameZH)
	}
	if event.LocationNameZH != "美国加利福尼亚州范登堡太空军基地" {
		t.Fatalf("unexpected Chinese location: %q", event.LocationNameZH)
	}
	if event.MissionDescriptionZH != "本次任务计划发射 24 颗星链卫星，用于扩充 SpaceX 的全球卫星互联网星座。" {
		t.Fatalf("unexpected Chinese description: %q", event.MissionDescriptionZH)
	}
	if !event.HasOriginal {
		t.Fatal("English source should expose the original copy")
	}
}

func TestLocalizeChineseLaunchEventDoesNotExposeOriginal(t *testing.T) {
	event := LaunchEvent{
		Name:               "长征五号｜嫦娥七号",
		StatusName:         "等待确认",
		LocationName:       "中国文昌航天发射场",
		MissionName:        "嫦娥七号",
		MissionType:        "行星科学任务",
		MissionDescription: "计划前往月球南极开展科学探测。",
	}

	LocalizeLaunchEvent(&event)

	if event.NameZH != event.Name || event.MissionDescriptionZH != event.MissionDescription {
		t.Fatal("Chinese source content should remain unchanged")
	}
	if event.HasOriginal {
		t.Fatal("Chinese source should not expose a redundant original-copy button")
	}
}

func TestLocalizeUnknownEnglishDescriptionUsesHonestFallback(t *testing.T) {
	event := LaunchEvent{
		Name:               "Example Vehicle | Example Mission",
		StatusName:         "To Be Determined",
		MissionName:        "Example Mission",
		MissionType:        "Technology",
		MissionDescription: "A new description that is not in the curated glossary.",
	}

	LocalizeLaunchEvent(&event)

	if event.MissionDescriptionZH != "这是一项技术验证任务，中文任务资料正在整理。" {
		t.Fatalf("unexpected fallback: %q", event.MissionDescriptionZH)
	}
	if !event.HasOriginal {
		t.Fatal("untranslated English source should keep an original-copy entry")
	}
}
