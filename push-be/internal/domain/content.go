package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Node struct {
	Type    string          `json:"type"`
	Attrs   json.RawMessage `json:"attrs"`
	Marks   []Mark          `json:"marks"`
	Text    string          `json:"text"`
	Content []Node          `json:"content"`
}

type Mark struct {
	Type  string          `json:"type"`
	Attrs json.RawMessage `json:"attrs"`
}

var allowedNodeTypes = map[string]bool{
	"doc":         true,
	"paragraph":   true,
	"heading":     true,
	"bulletList":  true,
	"orderedList": true,
	"listItem":    true,
	"text":        true,
}

var blockNodeTypes = map[string]bool{
	"paragraph": true,
	"heading":   true,
	"listItem":  true,
}

func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func HashJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return HashBytes(b)
}

func ValidateHTTPSURL(raw string) *Error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Validation("url must be http(s)")
	}
	if u.User != nil {
		return Validation("url must not contain credentials")
	}
	if u.Host == "" {
		return Validation("url must have a host")
	}
	return nil
}

type blockAttrs struct {
	BlockID string `json:"blockId"`
}

type linkAttrs struct {
	Href string `json:"href"`
}

func ValidateDocumentContent(content json.RawMessage, blocks []Block) *Error {
	var root Node
	if err := json.Unmarshal(content, &root); err != nil {
		return Validation("content must be valid TipTap JSON")
	}
	if root.Type != "doc" {
		return ValidationField("content", "root node must be doc")
	}
	extracted := map[string]string{}
	order := []string{}
	if err := walkNode(&root, extracted, &order); err != nil {
		return err
	}
	seenBlockIDs := map[string]bool{}
	for _, b := range blocks {
		if b.ID == "" {
			return ValidationField("blocks", "block id is required")
		}
		if seenBlockIDs[b.ID] {
			return ValidationField("blocks", "duplicate block id "+b.ID)
		}
		seenBlockIDs[b.ID] = true
		text, ok := extracted[b.ID]
		if !ok {
			return ValidationField("blocks", "block "+b.ID+" has no matching content node")
		}
		if text != b.Text {
			return ValidationField("blocks", "block "+b.ID+" text does not match content node text")
		}
	}
	for _, id := range order {
		if !seenBlockIDs[id] {
			return ValidationField("content", "content node "+id+" has no matching block")
		}
	}
	return nil
}

func walkNode(n *Node, extracted map[string]string, order *[]string) *Error {
	if !allowedNodeTypes[n.Type] {
		return ValidationField("content", "unsupported node type "+n.Type)
	}
	for _, m := range n.Marks {
		if m.Type != "link" {
			return ValidationField("content", "unsupported mark type "+m.Type)
		}
		var la linkAttrs
		if err := json.Unmarshal(m.Attrs, &la); err != nil || la.Href == "" {
			return ValidationField("content", "link mark requires href")
		}
		if e := ValidateHTTPSURL(la.Href); e != nil {
			return e
		}
	}
	if blockNodeTypes[n.Type] {
		var attrs blockAttrs
		if len(n.Attrs) == 0 {
			return ValidationField("content", "block node of type "+n.Type+" requires attrs.blockId")
		}
		if err := json.Unmarshal(n.Attrs, &attrs); err != nil || attrs.BlockID == "" {
			return ValidationField("content", "block node of type "+n.Type+" requires attrs.blockId")
		}
		if _, dup := extracted[attrs.BlockID]; dup {
			return ValidationField("content", "duplicate blockId "+attrs.BlockID)
		}
		extracted[attrs.BlockID] = nodeText(n)
		*order = append(*order, attrs.BlockID)
	}
	for i := range n.Content {
		if err := walkNode(&n.Content[i], extracted, order); err != nil {
			return err
		}
	}
	return nil
}

func nodeText(n *Node) string {
	out := ""
	var rec func(*Node)
	rec = func(c *Node) {
		if c.Type == "text" {
			out += c.Text
			return
		}
		for i := range c.Content {
			rec(&c.Content[i])
		}
	}
	rec(n)
	return out
}

func ExtractBlockText(content json.RawMessage) (map[string]string, *Error) {
	var root Node
	if err := json.Unmarshal(content, &root); err != nil {
		return nil, Validation("content must be valid TipTap JSON")
	}
	extracted := map[string]string{}
	order := []string{}
	if err := walkNode(&root, extracted, &order); err != nil {
		return nil, err
	}
	return extracted, nil
}

func CodePointSlice(s string, start, end int) (string, bool) {
	if start < 0 || end < 0 || start >= end {
		return "", false
	}
	runes := []rune(s)
	if end > len(runes) || start >= len(runes) {
		return "", false
	}
	return string(runes[start:end]), true
}

func CodePointLen(s string) int {
	return utf8.RuneCountInString(s)
}

func BuildExcerptContent(title string, entries []ExcerptEntry) (json.RawMessage, []Block) {
	nodes := []Node{}
	blocks := []Block{}
	i := 0
	for _, e := range entries {
		i++
		blockID := fmt.Sprintf("block-%d", i)
		attrs, _ := json.Marshal(map[string]string{"blockId": blockID})
		nodes = append(nodes, Node{
			Type:    "paragraph",
			Attrs:   attrs,
			Content: []Node{{Type: "text", Text: e.Text}},
		})
		blocks = append(blocks, Block{
			ID:   blockID,
			Text: e.Text,
			EvidenceRefs: []EvidenceRef{{
				EvidenceID: e.EvidenceID,
				Start:      e.Start,
				End:        e.End,
			}},
			ClaimStatus: ClaimSupported,
		})
	}
	doc := Node{Type: "doc", Content: nodes}
	raw, _ := json.Marshal(doc)
	return raw, blocks
}

type ExcerptEntry struct {
	EvidenceID uuid.UUID
	Text       string
	Start      int
	End        int
}
