package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/spf13/afero"
)

const ossDirSeparator = "/"

var (
	ossDefaultFileMode fs.FileMode = 0o644
	ossDefaultDirMode  fs.FileMode = 0o755
)

type OssObjectManager struct {
	ObjectManager
	Client *oss.Client
}

func mapOSSError(op, name string, err error) error {
	if err == nil {
		return nil
	}
	var serviceErr *oss.ServiceError
	if errors.As(err, &serviceErr) && (serviceErr.StatusCode == 404 || serviceErr.Code == "NoSuchKey") {
		return &os.PathError{Op: op, Path: name, Err: os.ErrNotExist}
	}
	return &os.PathError{Op: op, Path: name, Err: err}
}

func (m *OssObjectManager) GetObject(ctx context.Context, bucket, name string) (io.Reader, CleanUp, error) {
	res, err := m.Client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	})
	if err != nil {
		return nil, nil, mapOSSError("get", name, err)
	}
	if res == nil || res.Body == nil {
		return nil, nil, &os.PathError{Op: "get", Path: name, Err: io.ErrUnexpectedEOF}
	}
	return res.Body, func() { _ = res.Body.Close() }, nil
}

func (m *OssObjectManager) GetObjectPart(ctx context.Context, bucket, name string, start, end int64) (io.Reader, CleanUp, error) {
	if start < 0 || end < start {
		return nil, nil, afero.ErrOutOfRange
	}
	res, err := m.Client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket:        oss.Ptr(bucket),
		Key:           oss.Ptr(name),
		Range:         oss.Ptr(fmt.Sprintf("bytes=%d-%d", start, end)),
		RangeBehavior: oss.Ptr("standard"),
	})
	if err != nil {
		return nil, nil, mapOSSError("read", name, err)
	}
	if res == nil || res.Body == nil {
		return nil, nil, &os.PathError{Op: "read", Path: name, Err: io.ErrUnexpectedEOF}
	}
	return res.Body, func() { _ = res.Body.Close() }, nil
}

func (m *OssObjectManager) DeleteObject(ctx context.Context, bucket, name string) error {
	_, err := m.Client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	})
	return mapOSSError("remove", name, err)
}

func (m *OssObjectManager) IsObjectExist(ctx context.Context, bucket, name string) (bool, error) {
	exists, err := m.Client.IsObjectExist(ctx, bucket, name)
	return exists, mapOSSError("stat", name, err)
}

func (m *OssObjectManager) PutObject(ctx context.Context, bucket, name string, reader io.Reader) (bool, error) {
	_, err := m.Client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
		Body:   reader,
	})
	if err != nil {
		return false, mapOSSError("write", name, err)
	}
	return true, nil
}

func (m *OssObjectManager) CopyObject(ctx context.Context, bucket, srcName, targetName string) error {
	_, err := m.Client.CopyObject(ctx, &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(bucket),
		Key:          oss.Ptr(targetName),
		SourceKey:    oss.Ptr(srcName),
		SourceBucket: oss.Ptr(bucket),
		StorageClass: oss.StorageClassStandard,
	})
	return mapOSSError("rename", srcName, err)
}

func (m *OssObjectManager) GetObjectMeta(ctx context.Context, bucket, name string) (os.FileInfo, error) {
	res, err := m.Client.HeadObject(ctx, &oss.HeadObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	})
	if err != nil {
		return nil, mapOSSError("stat", name, err)
	}
	if res == nil || res.LastModified == nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: io.ErrUnexpectedEOF}
	}
	return newOssObjectMeta(displayName(name), res.ContentLength, *res.LastModified, strings.HasSuffix(name, ossDirSeparator)), nil
}

func displayName(name string) string {
	trimmed := strings.TrimSuffix(name, ossDirSeparator)
	if trimmed == "" {
		return ossDirSeparator
	}
	return path.Base(trimmed)
}

func appendEntry(entries []os.FileInfo, seen map[string]bool, entry *OssObjectMeta, count int) ([]os.FileInfo, bool) {
	key := fmt.Sprintf("%t:%s", entry.dir, entry.name)
	if seen[key] {
		return entries, false
	}
	seen[key] = true
	entries = append(entries, entry)
	return entries, count > 0 && len(entries) >= count
}

func (m *OssObjectManager) ListObjects(ctx context.Context, bucket, prefix string, count int) ([]os.FileInfo, error) {
	paginator := m.Client.NewListObjectsV2Paginator(&oss.ListObjectsV2Request{
		Bucket:    oss.Ptr(bucket),
		Delimiter: oss.Ptr(ossDirSeparator),
		Prefix:    oss.Ptr(prefix),
	})
	entries := make([]os.FileInfo, 0)
	seen := make(map[string]bool)

	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, mapOSSError("readdir", prefix, err)
		}
		for _, obj := range page.Contents {
			key := oss.ToString(obj.Key)
			if key == prefix {
				continue
			}
			relative := strings.TrimPrefix(key, prefix)
			entry := newOssObjectMeta(displayName(relative), obj.Size, oss.ToTime(obj.LastModified), strings.HasSuffix(key, ossDirSeparator))
			var full bool
			entries, full = appendEntry(entries, seen, entry, count)
			if full {
				break
			}
		}
		if count > 0 && len(entries) >= count {
			break
		}
		for _, commonPrefix := range page.CommonPrefixes {
			key := oss.ToString(commonPrefix.Prefix)
			relative := strings.TrimPrefix(key, prefix)
			entry := newOssObjectMeta(displayName(relative), 0, time.Time{}, true)
			var full bool
			entries, full = appendEntry(entries, seen, entry, count)
			if full {
				break
			}
		}
		if count > 0 && len(entries) >= count {
			break
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (m *OssObjectManager) ListAllObjects(ctx context.Context, bucket, prefix string) ([]os.FileInfo, error) {
	paginator := m.Client.NewListObjectsV2Paginator(&oss.ListObjectsV2Request{
		Bucket: oss.Ptr(bucket),
		Prefix: oss.Ptr(prefix),
	})
	entries := make([]os.FileInfo, 0)
	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, mapOSSError("readdir", prefix, err)
		}
		for _, obj := range page.Contents {
			key := oss.ToString(obj.Key)
			entries = append(entries, newOssObjectMeta(key, obj.Size, oss.ToTime(obj.LastModified), strings.HasSuffix(key, ossDirSeparator)))
		}
	}
	return entries, nil
}

type OssObjectMeta struct {
	os.FileInfo
	name           string
	size           int64
	lastModifiedAt time.Time
	dir            bool
}

func newOssObjectMeta(name string, size int64, updatedAt time.Time, dir bool) *OssObjectMeta {
	return &OssObjectMeta{name: name, size: size, lastModifiedAt: updatedAt, dir: dir}
}

func NewOssObjectMeta(name string, size int64, updatedAt time.Time) *OssObjectMeta {
	return newOssObjectMeta(name, size, updatedAt, strings.HasSuffix(name, ossDirSeparator))
}

func NewOssDirMeta(name string, updatedAt time.Time) *OssObjectMeta {
	return newOssObjectMeta(displayName(name), 0, updatedAt, true)
}

func (objMeta *OssObjectMeta) IsDir() bool { return objMeta.dir }

func (objMeta *OssObjectMeta) ModTime() time.Time { return objMeta.lastModifiedAt }

func (objMeta *OssObjectMeta) Mode() fs.FileMode {
	if objMeta.dir {
		return ossDefaultDirMode | fs.ModeDir
	}
	return ossDefaultFileMode
}

func (objMeta *OssObjectMeta) Name() string { return objMeta.name }

func (objMeta *OssObjectMeta) Size() int64 { return objMeta.size }

func (objMeta *OssObjectMeta) Sys() any { return nil }
