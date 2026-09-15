package astronomyevent

import (
	"encoding/json"
	"testing"
)

func TestNormalizedEventDocumentsDefaultsEmptyObjects(t *testing.T) {
	geometry, presentation, err := normalizedEventDocuments(Event{})
	if err != nil {
		t.Fatalf("normalizedEventDocuments() error = %v", err)
	}
	if string(geometry) != `{}` || string(presentation) != `{}` {
		t.Fatalf("documents = %s, %s; want {}, {}", geometry, presentation)
	}
}

func TestNormalizedEventDocumentsPreservesObjects(t *testing.T) {
	event := Event{Geometry: json.RawMessage(`{"object":"mars"}`), Presentation: json.RawMessage(`{"accent":"red"}`)}
	geometry, presentation, err := normalizedEventDocuments(event)
	if err != nil {
		t.Fatalf("normalizedEventDocuments() error = %v", err)
	}
	if string(geometry) != string(event.Geometry) || string(presentation) != string(event.Presentation) {
		t.Fatalf("valid objects changed: %s, %s", geometry, presentation)
	}
}

func TestNormalizedEventDocumentsRejectsNonObjects(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`not-json`)} {
		if _, _, err := normalizedEventDocuments(Event{Geometry: raw}); err == nil {
			t.Fatalf("normalizedEventDocuments(%s) error=nil, want error", raw)
		}
	}
}
