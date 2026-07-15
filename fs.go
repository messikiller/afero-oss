package ossfs

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/spf13/afero"

	"github.com/messikiller/afero-oss/internal/utils"
)

const (
	defaultFileMode = 0o666
	defaultFileFlag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
)

var ErrWriteConflict = errors.New("OSS object changed since this file was opened")

type Fs struct {
	manager        utils.ObjectManager
	bucketName     string
	separator      string
	autoSync       bool
	maxPreloadSize int64
	preloadFs      afero.Fs
	ctx            context.Context
	ossCfg         *oss.Config
	versionsMu     sync.Mutex
	versions       map[string]uint64
}

// NewOssFs creates a new OSS-backed afero filesystem.
func NewOssFs(accessKeyID, accessKeySecret, region, bucket string, ossOpts ...OSSOptionFunc) *Fs {
	if accessKeyID == "" {
		panic("accessKeyId cannot be empty")
	}
	if accessKeySecret == "" {
		panic("accessKeySecret cannot be empty")
	}
	if region == "" {
		panic("region cannot be empty")
	}
	if bucket == "" {
		panic("bucket cannot be empty")
	}

	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret)).
		WithRegion(region)
	for _, option := range ossOpts {
		if option != nil {
			option(ossCfg)
		}
	}

	return &Fs{
		manager:        &utils.OssObjectManager{Client: oss.NewClient(ossCfg)},
		bucketName:     bucket,
		separator:      "/",
		autoSync:       false,
		maxPreloadSize: defaultMaxPreloadSize,
		preloadFs:      afero.NewMemMapFs(),
		ctx:            context.Background(),
		ossCfg:         ossCfg,
		versions:       make(map[string]uint64),
	}
}

func (fs *Fs) versionFor(name string) uint64 {
	fs.versionsMu.Lock()
	defer fs.versionsMu.Unlock()
	if fs.versions == nil {
		fs.versions = make(map[string]uint64)
	}
	return fs.versions[name]
}

func (fs *Fs) putObject(name string, reader io.Reader) error {
	fs.versionsMu.Lock()
	defer fs.versionsMu.Unlock()
	if fs.versions == nil {
		fs.versions = make(map[string]uint64)
	}
	if _, err := fs.manager.PutObject(fs.ctx, fs.bucketName, name, reader); err != nil {
		return err
	}
	fs.versions[name]++
	return nil
}

func (fs *Fs) deleteObject(name string) error {
	fs.versionsMu.Lock()
	defer fs.versionsMu.Unlock()
	if fs.versions == nil {
		fs.versions = make(map[string]uint64)
	}
	if err := fs.manager.DeleteObject(fs.ctx, fs.bucketName, name); err != nil {
		return err
	}
	fs.versions[name]++
	return nil
}

func (fs *Fs) copyObject(source, target string) error {
	fs.versionsMu.Lock()
	defer fs.versionsMu.Unlock()
	if fs.versions == nil {
		fs.versions = make(map[string]uint64)
	}
	if err := fs.manager.CopyObject(fs.ctx, fs.bucketName, source, target); err != nil {
		return err
	}
	fs.versions[target]++
	return nil
}

// WithPreloadFs sets the filesystem used for private per-handle preload files.
func (fs *Fs) WithPreloadFs(pfs afero.Fs) *Fs {
	if pfs != nil {
		fs.preloadFs = pfs
	}
	return fs
}

// WithMaxPreloadSize sets the largest object that may be preloaded. A value of
// zero disables the limit, which is appropriate for a disk-backed preload Fs.
func (fs *Fs) WithMaxPreloadSize(size int64) *Fs {
	if size < 0 {
		panic("max preload size cannot be negative")
	}
	fs.maxPreloadSize = size
	return fs
}

// WithAutoSync controls whether each write is immediately uploaded. Close
// always syncs dirty data regardless of this setting.
func (fs *Fs) WithAutoSync(enabled bool) *Fs {
	fs.autoSync = enabled
	return fs
}

// WithContext sets the context used by subsequent operations.
func (fs *Fs) WithContext(ctx context.Context) *Fs {
	if ctx == nil {
		ctx = context.Background()
	}
	fs.ctx = ctx
	return fs
}

func pathError(op, name string, err error) error {
	return &os.PathError{Op: op, Path: name, Err: err}
}

func (fs *Fs) Create(name string) (afero.File, error) {
	return fs.OpenFile(name, defaultFileFlag, defaultFileMode)
}

func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	n := fs.normFileName(name)
	if n == "" {
		return pathError("mkdir", name, os.ErrExist)
	}
	if _, err := fs.Stat(n); err == nil {
		return pathError("mkdir", name, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	return fs.putObject(fs.ensureAsDir(n), strings.NewReader(""))
}

func (fs *Fs) MkdirAll(name string, perm os.FileMode) error {
	dir := strings.TrimSuffix(fs.ensureAsDir(name), fs.separator)
	if dir == "" {
		return nil
	}
	parts := strings.Split(dir, fs.separator)
	for i := range parts {
		component := strings.Join(parts[:i+1], fs.separator)
		fileExists, err := fs.manager.IsObjectExist(fs.ctx, fs.bucketName, component)
		if err != nil {
			return err
		}
		if fileExists {
			return pathError("mkdir", component, syscall.ENOTDIR)
		}
		dirKey := fs.ensureAsDir(component)
		exists, err := fs.manager.IsObjectExist(fs.ctx, fs.bucketName, dirKey)
		if err != nil {
			return err
		}
		if !exists {
			if err := fs.putObject(dirKey, strings.NewReader("")); err != nil {
				return err
			}
		}
	}
	return nil
}

// Open opens an existing path for reading. It never creates an OSS object.
func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	n := fs.normFileName(name)
	fi, statErr := fs.Stat(n)
	exists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, statErr
	}

	if exists && flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0 {
		return nil, pathError("open", name, os.ErrExist)
	}
	if !exists && flag&os.O_CREATE == 0 {
		return nil, pathError("open", name, os.ErrNotExist)
	}
	if n == "" && !exists {
		return nil, pathError("open", name, os.ErrNotExist)
	}

	resolved := n
	if exists && fi.IsDir() {
		resolved = fs.ensureAsDir(n)
		if flag&flagMask != os.O_RDONLY || flag&(os.O_APPEND|os.O_TRUNC) != 0 {
			return nil, pathError("open", name, syscall.EISDIR)
		}
	}

	writeMode := flag&flagMask == os.O_WRONLY || flag&flagMask == os.O_RDWR
	if flag&os.O_TRUNC != 0 && !writeMode {
		return nil, pathError("open", name, syscall.EINVAL)
	}
	if !exists {
		if err := fs.putObject(resolved, strings.NewReader("")); err != nil {
			return nil, err
		}
	}
	if exists && flag&os.O_TRUNC != 0 {
		if err := fs.putObject(resolved, strings.NewReader("")); err != nil {
			return nil, err
		}
	}

	return NewOssFile(resolved, flag, fs)
}

func (fs *Fs) Remove(name string) error {
	n := fs.normFileName(name)
	fi, err := fs.Stat(n)
	if err != nil {
		return err
	}
	key := n
	if fi.IsDir() {
		if n == "" {
			return pathError("remove", name, syscall.EBUSY)
		}
		key = fs.ensureAsDir(n)
		objects, err := fs.manager.ListAllObjects(fs.ctx, fs.bucketName, key)
		if err != nil {
			return err
		}
		for _, object := range objects {
			if object.Name() != key {
				return pathError("remove", name, syscall.ENOTEMPTY)
			}
		}
	}
	return fs.deleteObject(key)
}

func (fs *Fs) RemoveAll(name string) error {
	n := fs.normFileName(name)
	fi, err := fs.Stat(n)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !fi.IsDir() {
		return fs.deleteObject(n)
	}

	prefix := ""
	if n != "" {
		prefix = fs.ensureAsDir(n)
	}
	objects, err := fs.manager.ListAllObjects(fs.ctx, fs.bucketName, prefix)
	if err != nil {
		return err
	}
	for _, object := range objects {
		if err := fs.deleteObject(object.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Fs) Rename(oldname, newname string) error {
	oldKey := fs.normFileName(oldname)
	newKey := fs.normFileName(newname)
	if oldKey == "" || newKey == "" {
		return pathError("rename", oldname, syscall.EINVAL)
	}
	if oldKey == newKey {
		return nil
	}
	fi, err := fs.Stat(oldKey)
	if err != nil {
		return err
	}
	if fi.IsDir() && fs.ensureAsDir(oldKey) == fs.ensureAsDir(newKey) {
		return nil
	}
	targetInfo, targetErr := fs.Stat(newKey)
	if targetErr != nil && !os.IsNotExist(targetErr) {
		return targetErr
	}
	if targetErr == nil && fi.IsDir() != targetInfo.IsDir() {
		if fi.IsDir() {
			return pathError("rename", newname, syscall.ENOTDIR)
		}
		return pathError("rename", newname, syscall.EISDIR)
	}
	if !fi.IsDir() {
		if err := fs.copyObject(oldKey, newKey); err != nil {
			return err
		}
		return fs.deleteObject(oldKey)
	}

	oldPrefix := fs.ensureAsDir(oldKey)
	newPrefix := fs.ensureAsDir(newKey)
	if strings.HasPrefix(newPrefix, oldPrefix) {
		return pathError("rename", newname, syscall.EINVAL)
	}
	if targetErr == nil {
		targetObjects, err := fs.manager.ListAllObjects(fs.ctx, fs.bucketName, newPrefix)
		if err != nil {
			return err
		}
		for _, object := range targetObjects {
			if object.Name() != newPrefix {
				return pathError("rename", newname, syscall.ENOTEMPTY)
			}
		}
	}
	objects, err := fs.manager.ListAllObjects(fs.ctx, fs.bucketName, oldPrefix)
	if err != nil {
		return err
	}
	for _, object := range objects {
		target := newPrefix + strings.TrimPrefix(object.Name(), oldPrefix)
		if err := fs.copyObject(object.Name(), target); err != nil {
			return err
		}
	}
	for _, object := range objects {
		if err := fs.deleteObject(object.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	n := fs.normFileName(name)
	if n == "" {
		return NewDirFileInfo("", time.Time{}), nil
	}
	fi, err := fs.manager.GetObjectMeta(fs.ctx, fs.bucketName, n)
	if err == nil {
		return fi, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	dir := fs.ensureAsDir(n)
	fi, err = fs.manager.GetObjectMeta(fs.ctx, fs.bucketName, dir)
	if err == nil {
		return fi, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	entries, err := fs.manager.ListObjects(fs.ctx, fs.bucketName, dir, 1)
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		return NewDirFileInfo(n, time.Time{}), nil
	}
	return nil, pathError("stat", name, os.ErrNotExist)
}

func (fs *Fs) Name() string { return "OssFs" }

func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	return errors.New("OSS: method Chmod is not implemented")
}

func (fs *Fs) Chown(name string, uid, gid int) error {
	return errors.New("OSS: method Chown is not implemented")
}

func (fs *Fs) Chtimes(name string, atime, mtime time.Time) error {
	return errors.New("OSS: method Chtimes is not implemented")
}
