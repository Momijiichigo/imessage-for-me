package messaging

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore persists last-seen timestamps to disk in a JSON map.
type FileStore struct {
	path string
	mu   sync.RWMutex
	seen map[string]time.Time
}

func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{path: path, seen: make(map[string]time.Time)}
	if path == "" {
		return nil, errors.New("store path is empty")
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (f *FileStore) LastSeen(chat string) time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.seen[chat]
}

func (f *FileStore) SetLastSeen(chat string, ts time.Time) error {
	if chat == "" {
		return errors.New("chat identifier is empty")
	}
	f.mu.Lock()
	f.seen[chat] = ts
	f.mu.Unlock()
	return f.save()
}

func (f *FileStore) load() error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			f.seen[k] = parsed
		}
	}
	return nil
}

func (f *FileStore) save() error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	tmp := make(map[string]string, len(f.seen))
	for k, v := range f.seen {
		tmp[k] = v.Format(time.RFC3339Nano)
	}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(f.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(tmp)
}
