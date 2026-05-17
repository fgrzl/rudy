package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fgrzl/localagent/internal/config"
	"github.com/fgrzl/localagent/internal/indexer"
	searchoverlay "github.com/fgrzl/kv/pkg/search"
	"github.com/fgrzl/kv/pkg/storage/pebble"
)

type Agent struct {
	cfg      config.Config
	log      *slog.Logger
	mu       sync.RWMutex
	search   searchoverlay.SearchOverlay
	manifest map[string]indexer.FileRecord
}

type Summary struct {
	FilesIndexed  int `json:"files_indexed"`
	FilesSkipped  int `json:"files_skipped"`
	ChunksIndexed int `json:"chunks_indexed"`
	ChunksDeleted int `json:"chunks_deleted"`
}

func New(cfg config.Config, log *slog.Logger) (*Agent, error) {
	if log == nil {
		log = slog.Default()
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.WorkspaceDir, 0o755); err != nil {
		return nil, err
	}
	search, err := openSearch(cfg, log)
	if err != nil {
		return nil, err
	}
	return &Agent{
		cfg:      cfg,
		log:      log,
		search:   search,
		manifest: loadManifest(cfg.ManifestPath),
	}, nil
}

func (a *Agent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeLocked()
}

func (a *Agent) Search(ctx context.Context, query string, limit int) ([]searchoverlay.SearchHit, error) {
	a.mu.RLock()
	search := a.search
	a.mu.RUnlock()
	if search == nil {
		return nil, errors.New("search index is not initialized")
	}
	return search.Search(ctx, searchoverlay.Query{Text: query, Limit: limit})
}

func (a *Agent) SearchContext(ctx context.Context, query string, limit int) (string, error) {
	hits, err := a.Search(ctx, query, limit)
	if err != nil {
		return "", err
	}
	return indexer.BuildChatContext(hits, a.cfg.ContextChunkLimit, 12000), nil
}

func (a *Agent) Rebuild(ctx context.Context) (Summary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.closeLocked(); err != nil {
		return Summary{}, err
	}
	if err := os.RemoveAll(a.cfg.StoreDir); err != nil {
		return Summary{}, err
	}
	if err := os.Remove(a.cfg.ManifestPath); err != nil && !os.IsNotExist(err) {
		return Summary{}, err
	}

	search, err := openSearch(a.cfg, a.log)
	if err != nil {
		return Summary{}, err
	}
	a.search = search
	a.manifest = map[string]indexer.FileRecord{}

	return a.indexWorkspaceLocked(ctx)
}

func (a *Agent) IndexFile(ctx context.Context, absPath string) (Summary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.indexFileLocked(ctx, absPath)
}

func (a *Agent) indexWorkspaceLocked(ctx context.Context) (Summary, error) {
	root := a.cfg.WorkspaceDir
	var summary Summary

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			a.log.Warn("skipping path during index walk", "path", path, "err", walkErr)
			summary.FilesSkipped++
			return nil
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if shouldSkipFile(path) {
			summary.FilesSkipped++
			return nil
		}
		fileSummary, err := a.indexFileLocked(ctx, path)
		if err != nil {
			if errors.Is(err, indexer.ErrBinaryFile) {
				summary.FilesSkipped++
				return nil
			}
			a.log.Warn("skipping file during rebuild", "path", path, "err", err)
			summary.FilesSkipped++
			return nil
		}
		summary.FilesIndexed += fileSummary.FilesIndexed
		summary.ChunksIndexed += fileSummary.ChunksIndexed
		summary.ChunksDeleted += fileSummary.ChunksDeleted
		return nil
	})
	if err != nil {
		return summary, err
	}

	if err := saveManifest(a.cfg.ManifestPath, a.manifest); err != nil {
		return summary, err
	}

	return summary, nil
}

func (a *Agent) indexFileLocked(ctx context.Context, absPath string) (Summary, error) {
	search := a.search
	if search == nil {
		return Summary{}, errors.New("search index is not initialized")
	}

	bundle, err := indexer.BuildFileEntityBundle(a.cfg.WorkspaceDir, absPath, a.cfg.ChunkBytes, a.cfg.ChunkOverlap)
	if err != nil {
		return Summary{}, err
	}

	prev, hadPrev := a.manifest[bundle.RelPath]
	if hadPrev && prev.Hash == bundle.Record.Hash && prev.ChunkCount == bundle.Record.ChunkCount && prev.Size == bundle.Record.Size {
		return Summary{}, nil
	}

	deleted := 0
	if hadPrev && prev.ChunkCount > bundle.Record.ChunkCount {
		base := indexer.ChunkBaseID(bundle.RelPath)
		for idx := bundle.Record.ChunkCount; idx < prev.ChunkCount; idx++ {
			if err := search.Delete(ctx, indexer.ChunkEntityID(base, idx)); err == nil {
				deleted++
			}
		}
	}

	if len(bundle.Chunks) > 0 {
		if err := search.BatchIndex(ctx, bundle.Chunks); err != nil {
			return Summary{}, err
		}
	}

	a.manifest[bundle.RelPath] = bundle.Record
	if err := saveManifest(a.cfg.ManifestPath, a.manifest); err != nil {
		return Summary{}, err
	}

	return Summary{FilesIndexed: 1, ChunksIndexed: len(bundle.Chunks), ChunksDeleted: deleted}, nil
}

func (a *Agent) closeLocked() error {
	if a.search == nil {
		return nil
	}
	err := a.search.Close()
	a.search = nil
	return err
}

func openSearch(cfg config.Config, log *slog.Logger) (searchoverlay.SearchOverlay, error) {
	store, err := pebble.NewPebbleStore(cfg.StoreDir)
	if err != nil {
		return nil, err
	}
	return searchoverlay.New(store, cfg.IndexName, log), nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".cache", ".idea", ".vscode", ".next", ".venv", "__pycache__", "node_modules", "dist", "build", "vendor":
		return true
	default:
		return false
	}
}

func shouldSkipFile(path string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "~") || strings.HasPrefix(base, ".#") || base == ".DS_Store" {
		return true
	}
	return false
}

func loadManifest(path string) map[string]indexer.FileRecord {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]indexer.FileRecord{}
	}
	manifest := map[string]indexer.FileRecord{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return map[string]indexer.FileRecord{}
	}
	return manifest
}

func saveManifest(path string, manifest map[string]indexer.FileRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
