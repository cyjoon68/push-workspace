package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func docJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func para(blockID, text string) map[string]any {
	return map[string]any{
		"type":    "paragraph",
		"attrs":   map[string]any{"blockId": blockID},
		"content": []map[string]any{{"type": "text", "text": text}},
	}
}

func TestValidateDocumentContentValid(t *testing.T) {
	content := docJSON(t, map[string]any{
		"type":    "doc",
		"content": []map[string]any{para("b1", "hello"), para("b2", "world")},
	})
	blocks := []Block{
		{ID: "b1", Text: "hello", ClaimStatus: ClaimSupported},
		{ID: "b2", Text: "world", ClaimStatus: ClaimSupported},
	}
	if err := ValidateDocumentContent(content, blocks); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDocumentContentRootNotDoc(t *testing.T) {
	content := docJSON(t, map[string]any{"type": "paragraph"})
	if err := ValidateDocumentContent(content, nil); err == nil || err.Code != CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestValidateDocumentContentUnsupportedNode(t *testing.T) {
	content := docJSON(t, map[string]any{
		"type":    "doc",
		"content": []map[string]any{{"type": "script"}},
	})
	if err := ValidateDocumentContent(content, nil); err == nil {
		t.Fatal("want error for unsupported node type")
	}
}

func TestValidateDocumentContentBlockMismatch(t *testing.T) {
	content := docJSON(t, map[string]any{
		"type":    "doc",
		"content": []map[string]any{para("b1", "hello")},
	})
	if err := ValidateDocumentContent(content, []Block{{ID: "b1", Text: "different"}}); err == nil {
		t.Fatal("want error for mismatched block text")
	}
	if err := ValidateDocumentContent(content, []Block{{ID: "b9", Text: "hello"}}); err == nil {
		t.Fatal("want error for unknown block id")
	}
	if err := ValidateDocumentContent(content, nil); err == nil {
		t.Fatal("want error for content node without block")
	}
}

func TestValidateDocumentContentDuplicateBlockID(t *testing.T) {
	content := docJSON(t, map[string]any{
		"type":    "doc",
		"content": []map[string]any{para("b1", "a"), para("b1", "b")},
	})
	blocks := []Block{{ID: "b1", Text: "a"}, {ID: "b1", Text: "b"}}
	if err := ValidateDocumentContent(content, blocks); err == nil {
		t.Fatal("want error for duplicate blockId in content")
	}
}

func TestValidateDocumentContentLinkMark(t *testing.T) {
	bad := docJSON(t, map[string]any{
		"type": "doc",
		"content": []map[string]any{{
			"type":  "paragraph",
			"attrs": map[string]any{"blockId": "b1"},
			"content": []map[string]any{{
				"type": "text", "text": "x",
				"marks": []map[string]any{{"type": "link", "attrs": map[string]any{"href": "javascript:alert(1)"}}},
			}},
		}},
	})
	if err := ValidateDocumentContent(bad, []Block{{ID: "b1", Text: "x"}}); err == nil {
		t.Fatal("want error for non-http link href")
	}
}

func TestValidateHTTPSURL(t *testing.T) {
	for _, u := range []string{"https://example.com/x", "http://a.b/path?q=1"} {
		if err := ValidateHTTPSURL(u); err != nil {
			t.Errorf("ValidateHTTPSURL(%q) = %v", u, err)
		}
	}
	for _, u := range []string{"ftp://x", "https://user:pw@x/", "notaurl", ""} {
		if err := ValidateHTTPSURL(u); err == nil {
			t.Errorf("ValidateHTTPSURL(%q) should fail", u)
		}
	}
}

func TestCodePointSlice(t *testing.T) {
	s, ok := CodePointSlice("héllo wörld", 0, 5)
	if !ok || s != "héllo" {
		t.Fatalf("CodePointSlice = %q, %v", s, ok)
	}
	if _, ok := CodePointSlice("abc", 2, 2); ok {
		t.Fatal("empty slice should fail")
	}
	if CodePointLen("한국어") != 3 {
		t.Fatal("CodePointLen wrong")
	}
}

func TestBuildExcerptContentRoundTrip(t *testing.T) {
	eid := uuid.New()
	content, blocks := BuildExcerptContent("t", []ExcerptEntry{
		{EvidenceID: eid, Text: "built a service", Start: 0, End: 15},
	})
	if err := ValidateDocumentContent(content, blocks); err != nil {
		t.Fatalf("generated content must validate: %v", err)
	}
	if blocks[0].EvidenceRefs[0].EvidenceID != eid {
		t.Fatal("evidence ref lost")
	}
	if blocks[0].ClaimStatus != ClaimSupported {
		t.Fatal("claim status must be SUPPORTED")
	}
}

func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{CreatedAt: mustTime(t), ID: uuid.New()}
	dec, err := DecodeCursor(c.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if !dec.CreatedAt.Equal(c.CreatedAt) || dec.ID != c.ID {
		t.Fatal("cursor round trip mismatch")
	}
	if _, err := DecodeCursor("!!!notbase64!!!"); err == nil {
		t.Fatal("bad cursor should fail")
	}
}

func TestNewPage(t *testing.T) {
	items := []int{1, 2, 3}
	p := NewPage(items, 2, func(i int) Cursor {
		return Cursor{CreatedAt: mustTime(t), ID: uuid.New()}
	})
	if !p.HasMore || p.NextCursor == nil || len(p.Items) != 2 {
		t.Fatal("expected truncated page with cursor")
	}
	p = NewPage(items[:1], 2, func(i int) Cursor { return Cursor{} })
	if p.HasMore || p.NextCursor != nil {
		t.Fatal("expected single page without cursor")
	}
}

func TestEffectiveLimit(t *testing.T) {
	if (PageRequest{}).EffectiveLimit() != DefaultLimit {
		t.Fatal("default limit wrong")
	}
	if (PageRequest{Limit: 500}).EffectiveLimit() != MaxLimit {
		t.Fatal("limit should clamp to max")
	}
	if (PageRequest{Limit: 7}).EffectiveLimit() != 7 {
		t.Fatal("limit should pass through")
	}
}
