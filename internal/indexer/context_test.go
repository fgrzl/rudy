package indexer

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	searchoverlay "github.com/fgrzl/kv/pkg/search"
)

func TestBuildChatContextHonorsByteBudget(t *testing.T) {
	payload, _ := json.Marshal(ChunkPayload{RelativePath: strings.Repeat("path/", 30), Content: strings.Repeat("界", 1000)})
	hits := []searchoverlay.SearchHit{{Payload: payload}, {Payload: payload}}
	for _, limit := range []int{1, 64, 256, 3072} {
		got := BuildChatContext(hits, 2, limit)
		if len(got) > limit {
			t.Fatalf("context uses %d bytes, budget %d", len(got), limit)
		}
		if !utf8.ValidString(got) {
			t.Fatal("truncated context contains invalid UTF-8")
		}
	}
}
