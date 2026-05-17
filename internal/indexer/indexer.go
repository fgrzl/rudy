package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	searchoverlay "github.com/fgrzl/kv/pkg/search"
)

var ErrBinaryFile = errors.New("binary file")

type FileRecord struct {
	Hash       string    `json:"hash"`
	ChunkCount int       `json:"chunk_count"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

type ChunkPayload struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	ChunkIndex   int    `json:"chunk_index"`
	ChunkCount   int    `json:"chunk_count"`
	Offset       int    `json:"offset"`
	Length       int    `json:"length"`
	Hash         string `json:"hash"`
	Content      string `json:"content"`
}

type FileEntityBundle struct {
	Record  FileRecord
	Chunks  []searchoverlay.SearchEntity
	RelPath string
}

func BuildFileEntityBundle(rootDir, absPath string, chunkBytes, chunkOverlap int) (FileEntityBundle, error) {
	if chunkBytes <= 0 {
		chunkBytes = 4096
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkBytes {
		chunkOverlap = chunkBytes / 16
		if chunkOverlap < 64 {
			chunkOverlap = 64
		}
	}

	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return FileEntityBundle{}, err
	}
	absAbs, err := filepath.Abs(absPath)
	if err != nil {
		return FileEntityBundle{}, err
	}

	relPath, err := filepath.Rel(rootAbs, absAbs)
	if err != nil {
		return FileEntityBundle{}, err
	}
	relPath = filepath.ToSlash(relPath)
	if strings.HasPrefix(relPath, "..") {
		return FileEntityBundle{}, fmt.Errorf("%w: outside workspace root", ErrBinaryFile)
	}

	data, err := os.ReadFile(absAbs)
	if err != nil {
		return FileEntityBundle{}, err
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return FileEntityBundle{}, ErrBinaryFile
	}

	info, err := os.Stat(absAbs)
	if err != nil {
		return FileEntityBundle{}, err
	}

	content := string(data)
	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])

	chunks, offsets := splitTextIntoChunks(content, chunkBytes, chunkOverlap)
	if len(chunks) == 0 {
		chunks = []string{""}
		offsets = []int{0}
	}

	baseID := ChunkBaseID(relPath)
	entities := make([]searchoverlay.SearchEntity, 0, len(chunks))
	for idx, chunk := range chunks {
		payload := ChunkPayload{
			Path:         absAbs,
			RelativePath: relPath,
			ChunkIndex:   idx,
			ChunkCount:   len(chunks),
			Offset:       offsets[idx],
			Length:       len(chunk),
			Hash:         hash,
			Content:      chunk,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return FileEntityBundle{}, err
		}

		entities = append(entities, searchoverlay.SearchEntity{
			ID: ChunkEntityID(baseID, idx),
			Attributes: []searchoverlay.Attribute{
				{Field: "path", Value: relPath},
				{Field: "name", Value: filepath.Base(relPath)},
				{Field: "dir", Value: normalizeDir(filepath.Dir(relPath))},
				{Field: "ext", Value: strings.TrimPrefix(strings.ToLower(filepath.Ext(relPath)), ".")},
				{Field: "content", Value: chunk},
			},
			Payload: raw,
		})
	}

	return FileEntityBundle{
		Record: FileRecord{
			Hash:       hash,
			ChunkCount: len(chunks),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC(),
		},
		Chunks:  entities,
		RelPath: relPath,
	}, nil
}

func ChunkBaseID(relPath string) string {
	sum := sha256.Sum256([]byte(filepath.ToSlash(relPath)))
	return hex.EncodeToString(sum[:])
}

func ChunkEntityID(baseID string, index int) string {
	return fmt.Sprintf("%s:%06d", baseID, index)
}

func DecodeChunkPayload(raw []byte) (ChunkPayload, error) {
	var payload ChunkPayload
	if len(raw) == 0 {
		return payload, errors.New("empty payload")
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func BuildChatContext(hits []searchoverlay.SearchHit, maxChunks, maxChars int) string {
	if maxChunks <= 0 {
		maxChunks = 4
	}
	if maxChars <= 0 {
		maxChars = 12000
	}

	var b strings.Builder
	b.WriteString("Indexed workspace context:\n")
	count := 0
	chars := 0
	for _, hit := range hits {
		if count >= maxChunks || chars >= maxChars {
			break
		}
		payload, err := DecodeChunkPayload(hit.Payload)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(payload.Content)
		if content == "" {
			continue
		}
		if len(content) > 2500 {
			content = content[:2500] + "..."
		}
		entry := fmt.Sprintf("\n[%d] %s\n%s\n", count+1, payload.RelativePath, content)
		b.WriteString(entry)
		count++
		chars += len(entry)
	}
	if count == 0 {
		return ""
	}
	return b.String()
}

func splitTextIntoChunks(text string, size, overlap int) ([]string, []int) {
	if size <= 0 {
		size = 4096
	}
	if overlap < 0 || overlap >= size {
		overlap = size / 16
		if overlap < 64 {
			overlap = 64
		}
	}
	if text == "" {
		return []string{""}, []int{0}
	}

	chunks := make([]string, 0, (len(text)/size)+1)
	offsets := make([]int, 0, cap(chunks))
	start := 0
	for start < len(text) {
		end := start + size
		if end > len(text) {
			end = len(text)
		}
		if end < len(text) {
			window := text[start:end]
			if cut := strings.LastIndex(window, "\n\n"); cut > size/3 {
				end = start + cut + 2
			} else if cut := strings.LastIndex(window, "\n"); cut > size/2 {
				end = start + cut + 1
			}
		}

		chunk := text[start:end]
		if strings.TrimSpace(chunk) != "" {
			chunks = append(chunks, chunk)
			offsets = append(offsets, start)
		}

		if end >= len(text) {
			break
		}
		nextStart := end - overlap
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	if len(chunks) == 0 {
		return []string{""}, []int{0}
	}

	return chunks, offsets
}

func normalizeDir(dir string) string {
	if dir == "." || dir == "" {
		return ""
	}
	return filepath.ToSlash(dir)
}
