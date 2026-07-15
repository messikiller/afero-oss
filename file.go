package ossfs

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/spf13/afero"
)

const (
	defaultMaxPreloadSize = 100 * 1024 * 1024 // 100MB
	flagMask              = 0x3
)

type File struct {
	name     string
	fs       *Fs
	openFlag int
	offset   int64
	isDir    bool

	dirty        bool
	closed       bool
	preloaded    bool
	preloadedFd  afero.File
	preloadedKey string

	dirEntries []os.FileInfo
	dirOffset  int
	version    uint64

	mu sync.Mutex
}

// NewOssFile creates a new File instance. Each instance owns an independent
// file offset and preload file.
func NewOssFile(name string, flag int, fs *Fs) (*File, error) {
	normalized := fs.normFileName(name)
	return &File{
		name:     normalized,
		fs:       fs,
		openFlag: flag,
		isDir:    normalized == "" || fs.isDir(normalized),
		version:  fs.versionFor(normalized),
	}, nil
}

func (f *File) preloadLocked() (err error) {
	if f.preloaded {
		return nil
	}

	fi, err := f.fs.manager.GetObjectMeta(f.fs.ctx, f.fs.bucketName, f.name)
	if err != nil {
		return err
	}
	if limit := f.fs.maxPreloadSize; limit > 0 && fi.Size() > limit {
		return fmt.Errorf("file too large for preload: %d bytes (max: %d bytes)", fi.Size(), limit)
	}

	pfd, err := afero.TempFile(f.fs.preloadFs, ".", "afero-oss-preload-")
	if err != nil {
		return err
	}
	key := pfd.Name()
	defer func() {
		if err != nil {
			_ = pfd.Close()
			_ = f.fs.preloadFs.Remove(key)
		}
	}()

	r, clean, err := f.fs.manager.GetObject(f.fs.ctx, f.fs.bucketName, f.name)
	if err != nil {
		return err
	}
	if clean != nil {
		defer clean()
	}
	if _, err = io.Copy(pfd, r); err != nil {
		return err
	}
	if _, err = pfd.Seek(0, io.SeekStart); err != nil {
		return err
	}

	f.preloadedFd = pfd
	f.preloadedKey = key
	f.preloaded = true
	return nil
}

func (f *File) getFileInfoLocked() (os.FileInfo, error) {
	if f.closed {
		return nil, afero.ErrFileClosed
	}
	if f.preloaded && f.preloadedFd != nil {
		fi, err := f.preloadedFd.Stat()
		if err != nil {
			return nil, err
		}
		return NewFileInfo(path.Base(f.name), fi.Size(), fi.ModTime()), nil
	}
	return f.fs.Stat(f.name)
}

func (f *File) readableLocked() bool {
	if f.closed {
		return false
	}
	mode := f.openFlag & flagMask
	return mode == os.O_RDONLY || mode == os.O_RDWR
}

func (f *File) writeableLocked() bool {
	if f.closed {
		return false
	}
	mode := f.openFlag & flagMask
	return mode == os.O_WRONLY || mode == os.O_RDWR
}

func (f *File) appendOnlyLocked() bool {
	return f.writeableLocked() && f.openFlag&os.O_APPEND != 0
}

func (f *File) readAtLocked(p []byte, off int64) (int, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if !f.readableLocked() || f.isDir {
		return 0, syscall.EPERM
	}
	if off < 0 {
		return 0, syscall.ERANGE
	}
	if len(p) == 0 {
		return 0, nil
	}
	if f.preloaded && f.preloadedFd != nil {
		return f.preloadedFd.ReadAt(p, off)
	}

	reader, clean, err := f.fs.manager.GetObjectPart(
		f.fs.ctx, f.fs.bucketName, f.name, off, off+int64(len(p))-1,
	)
	if err != nil {
		return 0, err
	}
	if clean != nil {
		defer clean()
	}
	n, err := io.ReadFull(reader, p)
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}

// Read reads up to len(p) bytes and advances the file offset by the number of
// bytes actually read, including short reads that return io.EOF.
func (f *File) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	n, err := f.readAtLocked(p, f.offset)
	f.offset += int64(n)
	return n, err
}

// ReadAt reads len(p) bytes starting at off without changing the file offset.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.readAtLocked(p, off)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if (!f.readableLocked() && !f.writeableLocked()) || f.isDir {
		return 0, syscall.EPERM
	}
	fi, err := f.getFileInfoLocked()
	if err != nil {
		return 0, err
	}

	var next int64
	switch whence {
	case io.SeekCurrent:
		next = f.offset + offset
	case io.SeekStart:
		next = offset
	case io.SeekEnd:
		next = fi.Size() + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %v", whence)
	}
	if next < 0 {
		return 0, afero.ErrOutOfRange
	}
	f.offset = next
	return next, nil
}

func (f *File) doWriteAtLocked(p []byte, off int64) (int, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if f.isDir {
		return 0, syscall.EPERM
	}
	if off < 0 {
		return 0, syscall.ERANGE
	}
	if len(p) == 0 {
		return 0, nil
	}
	if err := f.preloadLocked(); err != nil {
		return 0, err
	}
	n, err := f.preloadedFd.WriteAt(p, off)
	if n > 0 {
		f.dirty = true
	}
	if err == nil && f.fs.autoSync && f.dirty {
		err = f.syncLocked()
	}
	return n, err
}

func (f *File) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if !f.writeableLocked() {
		return 0, syscall.EPERM
	}
	off := f.offset
	if f.appendOnlyLocked() {
		fi, err := f.getFileInfoLocked()
		if err != nil {
			return 0, err
		}
		off = fi.Size()
	}
	n, err := f.doWriteAtLocked(p, off)
	if !f.appendOnlyLocked() {
		f.offset += int64(n)
	}
	return n, err
}

func (f *File) WriteAt(p []byte, off int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if !f.writeableLocked() || f.appendOnlyLocked() {
		return 0, syscall.EPERM
	}
	return f.doWriteAtLocked(p, off)
}

func (f *File) syncLocked() error {
	if f.closed {
		return afero.ErrFileClosed
	}
	if !f.dirty || !f.preloaded || f.preloadedFd == nil {
		return nil
	}
	off, err := f.preloadedFd.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	defer func() { _, _ = f.preloadedFd.Seek(off, io.SeekStart) }()
	if _, err := f.preloadedFd.Seek(0, io.SeekStart); err != nil {
		return err
	}
	f.fs.versionsMu.Lock()
	defer f.fs.versionsMu.Unlock()
	if f.fs.versions == nil {
		f.fs.versions = make(map[string]uint64)
	}
	if f.fs.versions[f.name] != f.version {
		return ErrWriteConflict
	}
	if _, err := f.fs.manager.PutObject(f.fs.ctx, f.fs.bucketName, f.name, f.preloadedFd); err != nil {
		return err
	}
	f.fs.versions[f.name]++
	f.version = f.fs.versions[f.name]
	f.dirty = false
	return nil
}

// Close persists dirty content, then closes and removes this handle's private
// preload file. If persistence fails, the handle remains open so Sync/Close can
// be retried without losing the local changes.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return afero.ErrFileClosed
	}
	if err := f.syncLocked(); err != nil {
		return err
	}
	var closeErr error
	if f.preloadedFd != nil {
		if err := f.preloadedFd.Close(); err != nil {
			closeErr = err
		}
		if err := f.fs.preloadFs.Remove(f.preloadedKey); err != nil && !os.IsNotExist(err) && closeErr == nil {
			closeErr = err
		}
	}
	f.preloadedFd = nil
	f.preloadedKey = ""
	f.preloaded = false
	f.closed = true
	return closeErr
}

func (f *File) Name() string { return f.name }

func (f *File) loadDirEntriesLocked() error {
	if f.dirEntries != nil {
		return nil
	}
	entries, err := f.fs.manager.ListObjects(f.fs.ctx, f.fs.bucketName, f.fs.ensureAsDir(f.name), 0)
	if err != nil {
		return err
	}
	f.dirEntries = entries
	return nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil, afero.ErrFileClosed
	}
	if !f.readableLocked() || !f.isDir {
		return nil, syscall.EPERM
	}
	if err := f.loadDirEntriesLocked(); err != nil {
		return nil, err
	}
	if f.dirOffset >= len(f.dirEntries) {
		if count > 0 {
			return nil, io.EOF
		}
		return []os.FileInfo{}, nil
	}
	end := len(f.dirEntries)
	if count > 0 && f.dirOffset+count < end {
		end = f.dirOffset + count
	}
	entries := append([]os.FileInfo(nil), f.dirEntries[f.dirOffset:end]...)
	f.dirOffset = end
	return entries, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	entries, err := f.Readdir(n)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, err
}

func (f *File) Stat() (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getFileInfoLocked()
}

func (f *File) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.syncLocked()
}

func (f *File) Truncate(size int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return afero.ErrFileClosed
	}
	if !f.writeableLocked() || f.isDir {
		return syscall.EPERM
	}
	if size < 0 {
		return afero.ErrOutOfRange
	}
	if err := f.preloadLocked(); err != nil {
		return err
	}
	if err := f.preloadedFd.Truncate(size); err != nil {
		return err
	}
	f.dirty = true
	if f.fs.autoSync {
		return f.syncLocked()
	}
	return nil
}

func (f *File) WriteString(s string) (int, error) { return f.Write([]byte(s)) }
