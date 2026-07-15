package ossfs

import (
	"bytes"
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"

	"github.com/messikiller/afero-oss/internal/utils"
)

type memoryObjectManager struct {
	mu sync.Mutex

	objects map[string][]byte
	puts    map[string]int
	ranges  [][2]int64
	putErr  error
	getErr  error
}

func newMemoryObjectManager(objects map[string]string) *memoryObjectManager {
	m := &memoryObjectManager{objects: make(map[string][]byte), puts: make(map[string]int)}
	for key, value := range objects {
		m.objects[key] = []byte(value)
	}
	return m
}

func (m *memoryObjectManager) content(name string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.objects[name])
}

func (m *memoryObjectManager) has(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[name]
	return ok
}

func (m *memoryObjectManager) putCount(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.puts[name]
}

func (m *memoryObjectManager) GetObject(_ context.Context, _, name string) (io.Reader, utils.CleanUp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, nil, m.getErr
	}
	data, ok := m.objects[name]
	if !ok {
		return nil, nil, &os.PathError{Op: "get", Path: name, Err: os.ErrNotExist}
	}
	return bytes.NewReader(append([]byte(nil), data...)), func() {}, nil
}

func (m *memoryObjectManager) GetObjectPart(_ context.Context, _, name string, start, end int64) (io.Reader, utils.CleanUp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[name]
	if !ok {
		return nil, nil, &os.PathError{Op: "read", Path: name, Err: os.ErrNotExist}
	}
	m.ranges = append(m.ranges, [2]int64{start, end})
	if start >= int64(len(data)) {
		return bytes.NewReader(nil), func() {}, nil
	}
	if end >= int64(len(data)) {
		end = int64(len(data)) - 1
	}
	return bytes.NewReader(append([]byte(nil), data[start:end+1]...)), func() {}, nil
}

func (m *memoryObjectManager) DeleteObject(_ context.Context, _, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, name)
	return nil
}

func (m *memoryObjectManager) IsObjectExist(_ context.Context, _, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[name]
	return ok, nil
}

func (m *memoryObjectManager) PutObject(_ context.Context, _, name string, reader io.Reader) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.putErr != nil {
		err := m.putErr
		m.putErr = nil
		return false, err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}
	m.objects[name] = data
	m.puts[name]++
	return true, nil
}

func (m *memoryObjectManager) CopyObject(_ context.Context, _, srcName, targetName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[srcName]
	if !ok {
		return &os.PathError{Op: "rename", Path: srcName, Err: os.ErrNotExist}
	}
	m.objects[targetName] = append([]byte(nil), data...)
	return nil
}

func (m *memoryObjectManager) GetObjectMeta(_ context.Context, _, name string) (os.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[name]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
	return NewFileInfo(name, int64(len(data)), time.Unix(1, 0)), nil
}

func (m *memoryObjectManager) ListObjects(_ context.Context, _, prefix string, count int) ([]os.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries := make(map[string]os.FileInfo)
	for key, data := range m.objects {
		if key == prefix || !strings.HasPrefix(key, prefix) {
			continue
		}
		relative := strings.TrimPrefix(key, prefix)
		if slash := strings.Index(relative, "/"); slash >= 0 {
			name := relative[:slash]
			entries[name] = NewDirFileInfo(name, time.Unix(1, 0))
			continue
		}
		entries[relative] = NewFileInfo(relative, int64(len(data)), time.Unix(1, 0))
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	if count > 0 && len(names) > count {
		names = names[:count]
	}
	result := make([]os.FileInfo, len(names))
	for i, name := range names {
		result[i] = entries[name]
	}
	return result, nil
}

func (m *memoryObjectManager) ListAllObjects(_ context.Context, _, prefix string) ([]os.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0)
	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]os.FileInfo, 0, len(keys))
	for _, key := range keys {
		result = append(result, NewFileInfo(key, int64(len(m.objects[key])), time.Unix(1, 0)))
	}
	return result, nil
}

func newTestFs(objects map[string]string) (*Fs, *memoryObjectManager) {
	manager := newMemoryObjectManager(objects)
	return &Fs{
		manager:        manager,
		bucketName:     "bucket",
		separator:      "/",
		maxPreloadSize: defaultMaxPreloadSize,
		preloadFs:      afero.NewMemMapFs(),
		ctx:            context.Background(),
	}, manager
}
