package ossfs

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	"github.com/spf13/afero"
)

const (
	maxPreloadSize = 100 * 1024 * 1024 // 100MB max file size for preload
	flagMask       = 0x3               // Mask for extracting read/write mode
	flagMax        = 0x2               // Maximum value before masking needed
)

type File struct {
	name     string
	fs       *Fs
	openFlag int
	offset   int64
	isDir    bool

	// Whether the file is written.
	dirty bool

	// Whether the file is closed.
	closed bool

	// Whether the file is preloaded downto local.
	preloaded   bool
	preloadedFd afero.File

	mu sync.Mutex
}

// NewOssFile creates a new File instance, the name of file will be normalized.
func NewOssFile(name string, flag int, fs *Fs) (*File, error) {
	return &File{
		name:        fs.normFileName(name),
		fs:          fs,
		openFlag:    flag,
		offset:      0,
		dirty:       false,
		closed:      false,
		isDir:       fs.isDir(fs.normFileName(name)),
		preloaded:   false,
		preloadedFd: nil,
	}, nil
}

// preload will preload the file to specific preload-filesystem.
func (f *File) preload() error {
	fi, err := f.fs.manager.GetObjectMeta(f.fs.ctx, f.fs.bucketName, f.name)
	if err != nil {
		return err
	}

	if fi.Size() > maxPreloadSize {
		return fmt.Errorf("file too large for preload: %d bytes (max: %d bytes)", fi.Size(), maxPreloadSize)
	}

	pfs := f.fs.preloadFs
	if _, err := pfs.Stat(f.name); err == nil {
		if e := pfs.Remove(f.name); e != nil {
			return e
		}
	}
	pfd, err := f.fs.preloadFs.Create(f.name)
	if err != nil {
		return err
	}

	r, clean, e := f.fs.manager.GetObject(f.fs.ctx, f.fs.bucketName, f.name)
	if e != nil {
		return e
	}
	defer clean()

	if _, err := io.Copy(pfd, r); err != nil {
		return err
	}

	if _, err := pfd.Seek(f.offset, io.SeekStart); err != nil {
		return err
	}

	f.preloadedFd = pfd
	f.preloaded = true
	return nil
}

// getFileInfo returns the FileInfo of file.
func (f *File) getFileInfo() (os.FileInfo, error) {
	if f.dirty {
		if f.preloadedFd == nil {
			return nil, syscall.EACCES
		}
		return f.preloadedFd.Stat()
	}
	return f.fs.Stat(f.name)
}

// isReadable returns whether the file is readable by openFlag of the file instance.
func (f *File) isReadable() bool {
	if f.closed {
		return false
	}
	masked := f.openFlag
	if masked > flagMax {
		masked = f.openFlag & flagMask
	}
	return masked == os.O_RDONLY || masked == os.O_RDWR
}

// isWriteable returns whether the file is writeable by openFlag of the file instance.
func (f *File) isWriteable() bool {
	if f.closed {
		return false
	}
	masked := f.openFlag
	if masked > flagMax {
		masked = f.openFlag & flagMask
	}
	return masked == os.O_WRONLY || masked == os.O_RDWR
}

// isAppendOnly returns whether the file is append-only by openFlag of the file instance.
func (f *File) isAppendOnly() bool {
	return f.isWriteable() && f.openFlag&os.O_APPEND != 0
}

// Read reads up to len(p) bytes from the File, it implements interface: io.Reader.
func (f *File) Read(p []byte) (int, error) {
	if !f.isReadable() || f.isDir {
		return 0, syscall.EPERM
	}
	n, err := f.ReadAt(p, f.offset)
	if err != nil {
		return n, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.offset += int64(n)
	return n, err
}

// ReadAt reads len(p) bytes from the File starting at byte offset off into p.
// It implements interface: io.ReaderAt.
func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if !f.isReadable() || f.isDir {
		return 0, syscall.EPERM
	}
	reader, cleanUp, err := f.fs.manager.GetObjectPart(f.fs.ctx, f.fs.bucketName, f.name, off, off+int64(len(p)))
	if err != nil {
		return 0, err
	}
	defer cleanUp()
	return reader.Read(p)
}

// Seek sets the offset for the next Read or Write on file to offset,
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if (!f.isReadable() && !f.isWriteable()) || f.isDir {
		return 0, syscall.EPERM
	}
	fi, err := f.getFileInfo()
	if err != nil {
		return 0, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	max := fi.Size()
	var newOffset int64
	switch whence {
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekStart:
		newOffset = offset
	case io.SeekEnd:
		newOffset = max + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %v", whence)
	}
	if newOffset < 0 || newOffset > max {
		return 0, afero.ErrOutOfRange
	}
	f.offset = newOffset
	return f.offset, nil
}

// Write writes len(p) bytes to the File. It implements interface: io.Writer.
func (f *File) Write(p []byte) (int, error) {
	if !f.isWriteable() {
		return 0, syscall.EPERM
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isAppendOnly() {
		fi, err := f.getFileInfo()
		if err != nil {
			return 0, err
		}
		return f.doWriteAt(p, fi.Size())
	}

	n, e := f.doWriteAt(p, f.offset)
	if e != nil {
		return 0, e
	}
	f.offset += int64(n)
	return n, e
}

// doWriteAt write len(p) bytes at the offset of the File. It will preload file into preload-filesystem.
func (f *File) doWriteAt(p []byte, off int64) (int, error) {
	if f.isDir {
		return 0, syscall.EPERM
	}

	if !f.preloaded {
		if err := f.preload(); err != nil {
			return 0, err
		}
	}

	n, e := f.preloadedFd.WriteAt(p, off)
	f.dirty = true
	if f.fs.autoSync {
		f.Sync()
	}
	return n, e
}

// WriteAt writes len(p) bytes to the File starting at byte offset off.
// It implements interface: io.WriterAt.
func (f *File) WriteAt(p []byte, off int64) (int, error) {
	if !f.isWriteable() || f.isAppendOnly() {
		return 0, syscall.EPERM
	}
	if off < 0 {
		return 0, syscall.ERANGE
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.doWriteAt(p, off)
}

// Close will close the file and remove it from opened files map.
// It implements interface: io.Closer.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var firstErr error

	if err := f.Sync(); err != nil {
		firstErr = err
	}

	f.fs.filesMu.Lock()
	delete(f.fs.openedFiles, f.name)
	f.fs.filesMu.Unlock()

	if f.preloaded {
		if err := f.fs.preloadFs.Remove(f.name); err != nil && firstErr == nil {
			firstErr = err
		}
		if f.preloadedFd != nil {
			if err := f.preloadedFd.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		f.preloadedFd = nil
		f.preloaded = false
	}
	f.dirty = false
	f.closed = true
	return firstErr
}

func (f *File) Name() string {
	return f.name
}

// Readdir read count files from the directory.
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isReadable() || !f.isDir {
		return nil, syscall.EPERM
	}

	fis, err := f.fs.manager.ListObjects(f.fs.ctx, f.fs.bucketName, f.fs.ensureAsDir(f.name), count)
	return fis, err
}

// Readdirnames read n file names form the directory.
func (f *File) Readdirnames(n int) ([]string, error) {
	if !f.isReadable() || !f.isDir {
		return nil, syscall.EPERM
	}

	fis, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var fNames []string
	for _, fi := range fis {
		fNames = append(fNames, fi.Name())
	}

	return fNames, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.getFileInfo()
}

// Sync will sync the preloaded file into cloud storage.
func (f *File) Sync() error {
	if f.preloaded && f.preloadedFd != nil {
		off, err := f.preloadedFd.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		if _, err := f.preloadedFd.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := f.fs.manager.PutObject(f.fs.ctx, f.fs.bucketName, f.name, f.preloadedFd); err != nil {
			return err
		}
		if _, err := f.preloadedFd.Seek(off, io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) Truncate(size int64) error {
	if !f.isWriteable() || f.isDir {
		return syscall.EPERM
	}
	p := make([]byte, size)
	_, err := f.Write(p)
	return err
}

func (f *File) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}
