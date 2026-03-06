package ossfs

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/messikiller/afero-oss/internal/mocks"
)

func TestNewOssFs(t *testing.T) {
	tests := []struct {
		name            string
		accessKeyId     string
		accessKeySecret string
		region          string
		bucket          string
		ossOptFuncs     []OSSOptionFunc
		expected        *Fs
	}{
		{
			name:            "valid credentials",
			accessKeyId:     "testKeyId",
			accessKeySecret: "testKeySecret",
			region:          "test-region",
			bucket:          "test-bucket",
			ossOptFuncs: []OSSOptionFunc{func(c *oss.Config) {
				c.WithEndpoint("testEndpoint")
			}, func(c *oss.Config) {
				c.WithUserAgent("testUA")
			}},
			expected: &Fs{
				bucketName:  "test-bucket",
				separator:   "/",
				autoSync:    false,
				openedFiles: make(map[string]afero.File),
				preloadFs:   afero.NewMemMapFs(),
				ctx:         context.Background(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOssFs(tt.accessKeyId, tt.accessKeySecret, tt.region, tt.bucket, tt.ossOptFuncs...)
			assert.NotNil(t, got.manager)
			assert.Equal(t, tt.expected.bucketName, got.bucketName)
			assert.Equal(t, tt.expected.separator, got.separator)
			assert.Equal(t, tt.expected.autoSync, got.autoSync)
			assert.NotNil(t, got.openedFiles)
			assert.NotNil(t, got.preloadFs)
			assert.NotNil(t, got.ctx)
			assert.Equal(t, "testEndpoint", *got.ossCfg.Endpoint)
			assert.Equal(t, "testUA", *got.ossCfg.UserAgent)
		})
	}
}

func TestFsWithContext(t *testing.T) {
	type bgMeta string
	tests := []struct {
		name     string
		fs       *Fs
		ctx      context.Context
		expected *Fs
	}{
		{
			name: "set new context",
			fs: &Fs{
				ctx: context.Background(),
			},
			ctx: context.WithValue(context.Background(), bgMeta("testKey"), bgMeta("testValue")),
			expected: &Fs{
				ctx: context.WithValue(context.Background(), bgMeta("testKey"), bgMeta("testValue")),
			},
		},
		{
			name: "set nil context",
			fs: &Fs{
				ctx: context.Background(),
			},
			ctx: nil,
			expected: &Fs{
				ctx: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fs.WithContext(tt.ctx)
			assert.Equal(t, tt.expected.ctx, got.ctx)
			assert.Equal(t, tt.fs, got)
		})
	}
}

func TestFsCreate(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("create simple success", func(t *testing.T) {
		m.EXPECT().
			PutObject(fs.ctx, bucket, "test.txt", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.Create("test.txt")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("create prefixed file path success", func(t *testing.T) {
		m.EXPECT().
			PutObject(fs.ctx, bucket, "path/to/test.txt", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.Create("/path/to/test.txt")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("create dir path success", func(t *testing.T) {
		m.EXPECT().
			PutObject(ctx, bucket, "path/to/test_dir/", strings.NewReader("")).
			Return(true, nil).
			Once()

		file, err := fs.Create("/path/to/test_dir/")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		assert.Equal(t, "path/to/test_dir/", file.Name())
		m.AssertExpectations(t)
	})

	t.Run("create failure", func(t *testing.T) {
		m.EXPECT().
			PutObject(fs.ctx, bucket, "test2.txt", strings.NewReader("")).
			Return(false, afero.ErrFileNotFound).
			Once()
		_, err := fs.Create("test2.txt")
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}

func TestFsMkdirAll(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("MkDirAll simple success", func(t *testing.T) {
		m.EXPECT().
			PutObject(fs.ctx, bucket, "path/to/test_dir/", strings.NewReader("")).
			Return(true, nil).
			Once()
		err := fs.MkdirAll("/path/to/test_dir/", defaultFileMode)
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("MkDirAll failure", func(t *testing.T) {
		m.EXPECT().
			PutObject(fs.ctx, bucket, "path/to/test_dir/", strings.NewReader("")).
			Return(false, afero.ErrFileClosed).
			Once()
		err := fs.MkdirAll("/path/to/test_dir/", defaultFileMode)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileClosed)
		m.AssertExpectations(t)
	})
}

func TestFsOpenFile(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:     m,
		bucketName:  bucket,
		ctx:         ctx,
		separator:   "/",
		openedFiles: make(map[string]afero.File),
	}

	t.Run("open existing file success", func(t *testing.T) {
		m.EXPECT().
			IsObjectExist(ctx, bucket, "test.txt").
			Return(true, nil).
			Once()
		file, err := fs.OpenFile("test.txt", os.O_RDONLY, 0o644)
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("open non-existing file with create flag success", func(t *testing.T) {
		m.EXPECT().
			IsObjectExist(ctx, bucket, "new.txt").
			Return(false, nil).
			Once()
		m.EXPECT().
			PutObject(ctx, bucket, "new.txt", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.OpenFile("new.txt", os.O_CREATE|os.O_RDWR, 0o644)
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("open file with truncate flag success", func(t *testing.T) {
		m.EXPECT().
			IsObjectExist(ctx, bucket, "trunc.txt").
			Return(true, nil).
			Once()
		m.EXPECT().
			PutObject(ctx, bucket, "trunc.txt", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.OpenFile("trunc.txt", os.O_TRUNC|os.O_RDWR, 0o644)
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("open existing file from cache", func(t *testing.T) {
		cachedFile := &File{name: "cached.txt", openFlag: os.O_RDONLY}
		fs.openedFiles["cached.txt"] = cachedFile
		file, err := fs.OpenFile("cached.txt", os.O_RDONLY, 0o644)
		assert.Nil(t, err)
		assert.Equal(t, cachedFile, file)
		assert.Implements(t, (*afero.File)(nil), file)
	})

	t.Run("open non-existing file without create flag fails", func(t *testing.T) {
		m.EXPECT().
			IsObjectExist(ctx, bucket, "nonexist.txt").
			Return(false, nil).
			Once()
		_, err := fs.OpenFile("nonexist.txt", os.O_RDONLY, 0o644)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})

	t.Run("open file with check exist error fails", func(t *testing.T) {
		m.EXPECT().
			IsObjectExist(ctx, bucket, "error.txt").
			Return(false, afero.ErrFileNotFound).
			Once()
		_, err := fs.OpenFile("error.txt", os.O_RDONLY, 0o644)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}

func TestFsRemove(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("remove file success", func(t *testing.T) {
		m.EXPECT().DeleteObject(fs.ctx, bucket, "test.txt").Return(nil).Once()
		err := fs.Remove("test.txt")
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("remove prefixed file success", func(t *testing.T) {
		m.EXPECT().DeleteObject(fs.ctx, bucket, "path/to/test.txt").Return(nil).Once()
		err := fs.Remove("/path/to/test.txt")
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("remove non-existent file", func(t *testing.T) {
		m.EXPECT().DeleteObject(fs.ctx, bucket, "nonexistent.txt").Return(afero.ErrFileNotFound).Once()
		err := fs.Remove("nonexistent.txt")
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}

func TestFsRemoveAll(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("remove non-empty directory", func(t *testing.T) {
		dirPath := "path/to/dir/"
		files := []os.FileInfo{
			NewFileInfo("path/to/dir/file1.txt", 100, time.Now()),
			NewFileInfo("path/to/dir/file2.txt", 200, time.Now()),
			NewFileInfo("path/to/dir/subdir/", 0, time.Now()),
		}

		m.EXPECT().ListAllObjects(ctx, bucket, dirPath).Return(files, nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, "path/to/dir/file1.txt").Return(nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, "path/to/dir/file2.txt").Return(nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, "path/to/dir/subdir/").Return(nil).Once()

		err := fs.RemoveAll(dirPath)
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("remove empty directory", func(t *testing.T) {
		dirPath := "empty/dir/"
		m.EXPECT().ListAllObjects(ctx, bucket, dirPath).Return([]os.FileInfo{}, nil).Once()

		err := fs.RemoveAll(dirPath)
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("remove non-existent path", func(t *testing.T) {
		nonExistentPath := "nonexistent/path/"
		m.EXPECT().ListAllObjects(ctx, bucket, nonExistentPath).Return([]os.FileInfo{}, nil).Once()

		err := fs.RemoveAll(nonExistentPath)
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("list objects failure", func(t *testing.T) {
		dirPath := "path/to/dir/"
		m.EXPECT().ListAllObjects(ctx, bucket, dirPath).Return(nil, afero.ErrFileNotFound).Once()

		err := fs.RemoveAll(dirPath)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})

	t.Run("delete object failure", func(t *testing.T) {
		dirPath := "path/to/dir/"
		files := []os.FileInfo{
			NewFileInfo("path/to/dir/file1.txt", 0, time.Now()),
		}

		m.EXPECT().ListAllObjects(ctx, bucket, dirPath).Return(files, nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, "path/to/dir/file1.txt").Return(afero.ErrFileNotFound).Once()

		err := fs.RemoveAll(dirPath)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}

func TestFsRename(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("successful rename", func(t *testing.T) {
		oldname := "old/file.txt"
		newname := "new/file.txt"

		m.EXPECT().CopyObject(ctx, bucket, oldname, newname).Return(nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, oldname).Return(nil).Once()

		err := fs.Rename(oldname, newname)
		assert.Nil(t, err)
		m.AssertExpectations(t)
	})

	t.Run("copy failure", func(t *testing.T) {
		oldname := "old/file.txt"
		newname := "new/file.txt"

		m.EXPECT().CopyObject(ctx, bucket, oldname, newname).Return(afero.ErrFileNotFound).Once()

		err := fs.Rename(oldname, newname)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})

	t.Run("delete failure after successful copy", func(t *testing.T) {
		oldname := "old/file.txt"
		newname := "new/file.txt"

		m.EXPECT().CopyObject(ctx, bucket, oldname, newname).Return(nil).Once()
		m.EXPECT().DeleteObject(ctx, bucket, oldname).Return(afero.ErrFileNotFound).Once()

		err := fs.Rename(oldname, newname)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}

func TestFsStat(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("stat file success", func(t *testing.T) {
		expectedInfo := mocks.NewMockFileInfo(t)
		m.EXPECT().GetObjectMeta(fs.ctx, bucket, "test.txt").Return(expectedInfo, nil).Once()
		info, err := fs.Stat("test.txt")
		assert.Nil(t, err)
		assert.Equal(t, expectedInfo, info)
		m.AssertExpectations(t)
	})

	t.Run("stat prefixed file path success", func(t *testing.T) {
		expectedInfo := mocks.NewMockFileInfo(t)
		m.EXPECT().GetObjectMeta(fs.ctx, bucket, "path/to/test.txt").Return(expectedInfo, nil).Once()
		info, err := fs.Stat("/path/to/test.txt")
		assert.Nil(t, err)
		assert.Equal(t, expectedInfo, info)
		m.AssertExpectations(t)
	})

	t.Run("stat dir path success", func(t *testing.T) {
		expectedInfo := mocks.NewMockFileInfo(t)
		m.EXPECT().GetObjectMeta(fs.ctx, bucket, "path/to/dir/").Return(expectedInfo, nil).Once()
		info, err := fs.Stat("/path/to/dir/")
		assert.Nil(t, err)
		assert.Equal(t, expectedInfo, info)
		m.AssertExpectations(t)
	})

	t.Run("stat non-existent file", func(t *testing.T) {
		m.EXPECT().GetObjectMeta(fs.ctx, bucket, "nonexistent.txt").Return(nil, os.ErrNotExist).Once()
		_, err := fs.Stat("nonexistent.txt")
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
		m.AssertExpectations(t)
	})
}

func TestFsName(t *testing.T) {
	fs := &Fs{}
	name := fs.Name()
	assert.Equal(t, "OssFs", name)
}

func TestFsChmod(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("chmod returns not implemented error", func(t *testing.T) {
		err := fs.Chmod("test.txt", 0o644)
		assert.NotNil(t, err)
		assert.EqualError(t, err, "OSS: method Chmod is not implemented")
	})
}

func TestFsChown(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("chown returns not implemented error", func(t *testing.T) {
		err := fs.Chown("test.txt", 1000, 1000)
		assert.NotNil(t, err)
		assert.EqualError(t, err, "OSS: method Chown is not implemented")
	})
}

func TestFsChtimes(t *testing.T) {
	m := mocks.NewMockObjectManager(t)
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("chtimes returns not implemented error", func(t *testing.T) {
		now := time.Now()
		err := fs.Chtimes("test.txt", now, now)
		assert.NotNil(t, err)
		assert.EqualError(t, err, "OSS: method Chtimes is not implemented")
	})
}
