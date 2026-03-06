package ossfs

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/messikiller/afero-oss/internal/mocks"
	"github.com/messikiller/afero-oss/internal/utils"
)

func getMockedFs(t *testing.T) *Fs {
	fs := NewOssFs("test-ak", "test-sk", "test-region", "test-bucket")
	fs.manager = mocks.NewMockObjectManager(t)
	return fs
}

func getMockedFile(name string, flag int, fs *Fs) *File {
	f, _ := NewOssFile(name, flag, fs)
	return f
}

func getMockedFileInfo(t *testing.T) *mocks.MockFileInfo {
	return mocks.NewMockFileInfo(t)
}

func TestNewOssFile(t *testing.T) {
	t.Run("create new file with read flag", func(t *testing.T) {
		fs := &Fs{}
		file, err := NewOssFile("testfile", os.O_RDONLY, fs)
		assert.NoError(t, err)
		assert.Equal(t, "testfile", file.name)
		assert.Equal(t, os.O_RDONLY, file.openFlag)
		assert.Equal(t, fs, file.fs)
		assert.False(t, file.dirty)
		assert.False(t, file.closed)
		assert.False(t, file.isDir)
		assert.False(t, file.preloaded)
		assert.Nil(t, file.preloadedFd)
	})

	t.Run("create new file with write flag", func(t *testing.T) {
		fs := &Fs{}
		file, err := NewOssFile("testfile", os.O_WRONLY, fs)
		assert.NoError(t, err)
		assert.Equal(t, "testfile", file.name)
		assert.Equal(t, os.O_WRONLY, file.openFlag)
		assert.Equal(t, fs, file.fs)
		assert.False(t, file.dirty)
		assert.False(t, file.closed)
		assert.False(t, file.isDir)
		assert.False(t, file.preloaded)
		assert.Nil(t, file.preloadedFd)
	})

	t.Run("create new directory", func(t *testing.T) {
		fs := &Fs{}
		file, err := NewOssFile("testdir/", os.O_RDONLY, fs)
		assert.NoError(t, err)
		assert.Equal(t, "testdir/", file.name)
		assert.Equal(t, os.O_RDONLY, file.openFlag)
		assert.Equal(t, fs, file.fs)
		assert.False(t, file.dirty)
		assert.False(t, file.closed)
		assert.True(t, file.isDir)
		assert.False(t, file.preloaded)
		assert.Nil(t, file.preloadedFd)
	})

	t.Run("normalize file name", func(t *testing.T) {
		fs := &Fs{}
		file, err := NewOssFile("/path/testfile", os.O_RDONLY, fs)
		assert.NoError(t, err)
		assert.Equal(t, "path/testfile", file.name)
	})
}

func TestFileIsReadable(t *testing.T) {
	fs := getMockedFs(t)
	f := getMockedFile("testfile", defaultFileFlag, fs)

	failCases := []int{
		os.O_WRONLY,
		os.O_WRONLY | os.O_APPEND,
		os.O_WRONLY | os.O_CREATE,
		os.O_WRONLY | os.O_APPEND | os.O_EXCL,
	}

	trueCases := []int{
		os.O_RDONLY,
		os.O_RDWR,
		os.O_RDONLY | os.O_CREATE,
		os.O_RDWR | os.O_CREATE | os.O_EXCL,
		os.O_RDWR | os.O_APPEND,
		os.O_APPEND,
		os.O_EXCL | os.O_TRUNC,
	}

	t.Run("unreadable flags return false", func(t *testing.T) {
		for i, c := range failCases {
			f.openFlag = c
			assert.False(t, f.isReadable(), fmt.Sprintf("false case failed: %v", i))
		}
	})

	t.Run("readable flags return true", func(t *testing.T) {
		for i, c := range trueCases {
			f.openFlag = c
			assert.True(t, f.isReadable(), fmt.Sprintf("true case failed: %v", i))
		}
	})

	t.Run("closed file return false", func(t *testing.T) {
		f.closed = true
		for _, c := range trueCases {
			f.openFlag = c
			assert.False(t, f.isReadable())
		}
	})
}

func TestFileIsWritable(t *testing.T) {
	fs := getMockedFs(t)
	f := getMockedFile("testfile", defaultFileFlag, fs)

	trueCases := []int{
		os.O_WRONLY,
		os.O_WRONLY | os.O_APPEND,
		os.O_WRONLY | os.O_CREATE,
		os.O_WRONLY | os.O_APPEND | os.O_EXCL,
		os.O_RDWR,
		os.O_RDWR | os.O_APPEND,
	}

	failCases := []int{
		os.O_RDONLY,
		os.O_RDONLY | os.O_CREATE,
		os.O_RDONLY | os.O_CREATE | os.O_EXCL,
		os.O_RDONLY | os.O_TRUNC,
	}

	t.Run("unwritable flags return false", func(t *testing.T) {
		for i, c := range failCases {
			f.openFlag = c
			assert.False(t, f.isWriteable(), fmt.Sprintf("false case failed: %v", i))
		}
	})

	t.Run("writable flags return true", func(t *testing.T) {
		for i, c := range trueCases {
			f.openFlag = c
			assert.True(t, f.isWriteable(), fmt.Sprintf("true case failed: %v", i))
		}
	})

	t.Run("closed file return false", func(t *testing.T) {
		f.closed = true
		for _, c := range trueCases {
			f.openFlag = c
			assert.False(t, f.isWriteable())
		}
	})
}

func TesFileRead(t *testing.T) {
	t.Run("Read with unreadable flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)

		p := make([]byte, 0)
		_, e := f.Read(p)

		assert.Error(t, e)
		assert.NotNil(t, e)
	})

	t.Run("Read on directory return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir", os.O_RDONLY, fs)
		f.isDir = true

		p := make([]byte, 10)
		_, e := f.Read(p)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Read on closed file return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDONLY, fs)
		f.closed = true

		p := make([]byte, 10)
		_, e := f.Read(p)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Successful read updates offset", func(t *testing.T) {
		fs := getMockedFs(t)
		var cu utils.CleanUp = func() {}
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader("testdata"), cu, nil)

		f := getMockedFile("testfile", os.O_RDONLY, fs)
		p := make([]byte, 8)
		n, err := f.Read(p)

		assert.NoError(t, err)
		assert.Equal(t, 8, n)
		assert.Equal(t, int64(8), f.offset)
	})

	t.Run("ReadAt error propagates", func(t *testing.T) {
		fs := getMockedFs(t)
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, nil, syscall.EIO)

		f := getMockedFile("testfile", os.O_RDONLY, fs)
		p := make([]byte, 8)
		_, err := f.Read(p)

		assert.Error(t, err)
		assert.Equal(t, syscall.EIO, err)
	})
}

func TestFileReadAt(t *testing.T) {
	t.Run("ReadAt with unreadable flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)

		p := make([]byte, 0)
		_, e := f.ReadAt(p, 0)

		assert.Error(t, e)
		assert.NotNil(t, e)
	})

	t.Run("ReadAt on dir return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("/path/to/dir/", os.O_WRONLY, fs)

		p := make([]byte, 0)
		_, e := f.ReadAt(p, 0)

		assert.Error(t, e)
		assert.NotNil(t, e)
	})

	t.Run("ReadAt success", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDONLY, fs)

		p := make([]byte, 4)

		var cu utils.CleanUp = func() {}
		off := int64(5)
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectPart(f.fs.ctx, f.fs.bucketName, f.name, off, off+int64(len(p))).
			Return(strings.NewReader("test result"), cu, nil)

		n, e := f.ReadAt(p, off)

		assert.Nil(t, e)
		assert.Equal(t, 4, n)
		assert.Equal(t, "test", string(p))
	})
}

func TestFileSeek(t *testing.T) {
	t.Run("Seek on unreadable/unwritable file returns error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY|os.O_APPEND, fs)
		f.closed = true

		_, err := f.Seek(0, io.SeekStart)
		assert.Error(t, err)
		assert.Equal(t, syscall.EPERM, err)
	})

	t.Run("Seek on directory returns error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir", os.O_RDONLY, fs)
		f.isDir = true

		_, err := f.Seek(0, io.SeekStart)
		assert.Error(t, err)
		assert.Equal(t, syscall.EPERM, err)
	})

	t.Run("Seek with invalid whence returns error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDWR, fs)
		fi := getMockedFileInfo(t)
		fi.On("Size").Return(int64(0))
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		_, err := f.Seek(0, 3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid whence value")
	})

	t.Run("Seek beyond file size returns error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDWR, fs)
		fi := getMockedFileInfo(t)
		fi.On("Size").Return(int64(100))
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		_, err := f.Seek(101, io.SeekStart)
		assert.Error(t, err)
		assert.Equal(t, afero.ErrOutOfRange, err)
	})

	t.Run("Seek to negative offset returns error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDWR, fs)
		fi := getMockedFileInfo(t)
		fi.On("Size").Return(int64(100))
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		_, err := f.Seek(-1, io.SeekStart)
		assert.Error(t, err)
		assert.Equal(t, afero.ErrOutOfRange, err)
	})

	t.Run("Successful SeekStart updates offset", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDWR, fs)
		fi := getMockedFileInfo(t)
		fi.On("Size").Return(int64(100))
		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		newOffset, err := f.Seek(50, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(50), newOffset)
		assert.Equal(t, int64(50), f.offset)
	})

	// 	t.Run("Successful SeekCurrent updates offset", func(t *testing.T) {
	// 		fs := getMockedFs(t)
	// 		f := getMockedFile("testfile", os.O_RDWR, fs)
	// 		f.offset = 20
	// 		fi := getMockedFileInfo(t)
	// 		fi.On("Size").Return(int64(100))
	// 		fs.manager.(*mocks.ObjectManager).
	// 			On("GetObjectMeta", mock.Anything, mock.Anything, mock.Anything).
	// 			Return(fi, nil)

	// 		newOffset, err := f.Seek(30, io.SeekCurrent)
	// 		assert.NoError(t, err)
	// 		assert.Equal(t, int64(50), newOffset)
	// 		assert.Equal(t, int64(50), f.offset)
	// 	})

	// 	t.Run("Successful SeekEnd updates offset", func(t *testing.T) {
	// 		fs := getMockedFs(t)
	// 		f := getMockedFile("testfile", os.O_RDWR, fs)
	// 		fi := getMockedFileInfo(t)
	// 		fi.On("Size").Return(int64(100))
	// 		fs.manager.(*mocks.ObjectManager).
	// 			On("GetObjectMeta", mock.Anything, mock.Anything, mock.Anything).
	// 			Return(fi, nil)

	// 		newOffset, err := f.Seek(-10, io.SeekEnd)
	// 		assert.NoError(t, err)
	// 		assert.Equal(t, int64(90), newOffset)
	// 		assert.Equal(t, int64(90), f.offset)
	// 	})

	// 	t.Run("Stat error propagates", func(t *testing.T) {
	// 		fs := getMockedFs(t)
	// 		f := getMockedFile("testfile", os.O_RDWR, fs)
	// 		fs.manager.(*mocks.ObjectManager).
	// 			On("GetObjectMeta", mock.Anything, mock.Anything, mock.Anything).
	// 			Return(nil, syscall.EIO)

	//		_, err := f.Seek(0, io.SeekStart)
	//		assert.Error(t, err)
	//		assert.Equal(t, syscall.EIO, err)
	//	})
}

func TestFileDoWriteAt(t *testing.T) {
	t.Run("WriteAt on dir return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("/path/to/dir/", os.O_WRONLY, fs)
		f.isDir = true

		p := make([]byte, 0)
		_, e := f.doWriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("WriteAt with preload error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)

		p := make([]byte, 0)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(f.fs.ctx, f.fs.bucketName, f.name).
			Return(nil, fmt.Errorf("preload error"))

		_, e := f.doWriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, "preload error", e.Error())
	})

	t.Run("WriteAt success", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)

		fi := getMockedFileInfo(t)
		fi.EXPECT().Size().Return(int64(0))

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader(""), utils.CleanUp(func() {}), nil)

		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove(f.name)

		p := []byte("test data")
		n, e := f.doWriteAt(p, 0)

		assert.Nil(t, e)
		assert.Equal(t, len(p), n)
		assert.True(t, f.dirty)

		f.preloadedFd.Seek(0, io.SeekStart)
		s, _ := io.ReadAll(f.preloadedFd)
		assert.Equal(t, "test data", string(s))
	})

	t.Run("WriteAt at non zero position success", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)

		originalContent := "abcdefg"
		fi := getMockedFileInfo(t)
		fi.EXPECT().Size().Return(int64(len(originalContent)))

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader(originalContent), utils.CleanUp(func() {}), nil)

		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove(f.name)

		n, e := f.doWriteAt([]byte("ABCD"), 2)

		assert.Equal(t, 4, n)
		assert.NoError(t, e)

		f.preloadedFd.Seek(0, io.SeekStart)
		s, _ := io.ReadAll(f.preloadedFd)

		assert.Equal(t, "abABCDg", string(s))
	})
}

func TestFileWrite(t *testing.T) {
	t.Run("Write with unwritable flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDONLY, fs)

		p := []byte("test")
		_, e := f.Write(p)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Write on directory return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir", os.O_WRONLY, fs)
		f.isDir = true

		p := []byte("test")
		_, e := f.Write(p)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Write on closed file return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)
		f.closed = true

		p := []byte("test")
		_, e := f.Write(p)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Successful write updates offset", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)
		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove("testfile")

		fi := getMockedFileInfo(t)
		fi.EXPECT().Size().Return(int64(0))

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader(""), utils.CleanUp(func() {}), nil)

		p := []byte("testdata")
		n, err := f.Write(p)

		assert.NoError(t, err)
		assert.Equal(t, 8, n)
		assert.Equal(t, int64(8), f.offset)
		assert.True(t, f.dirty)

		f.preloadedFd.Seek(0, io.SeekStart)
		s, _ := io.ReadAll(f.preloadedFd)
		assert.Equal(t, "testdata", string(s))
	})

	t.Run("Append mode writes at end of file", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY|os.O_APPEND, fs)
		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove("testfile")
		fi := getMockedFileInfo(t)

		originalContent := "this is original content"

		fi.EXPECT().Size().Return(int64(len(originalContent))).Times(2)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil).Times(2)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader(originalContent), utils.CleanUp(func() {}), nil)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		p := []byte("data")
		n, err := f.Write(p)

		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, int64(0), f.offset)

		f.preloadedFd.Seek(0, io.SeekStart)
		s, _ := io.ReadAll(f.preloadedFd)
		assert.Equal(t, originalContent+"data", string(s))
	})
}

func TestFileWriteAt(t *testing.T) {
	t.Run("WriteAt with unwritable flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDONLY, fs)

		p := []byte("test")
		_, e := f.WriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("WriteAt on directory return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir", os.O_WRONLY, fs)
		f.isDir = true

		p := []byte("test")
		_, e := f.WriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("WriteAt on closed file return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)
		f.closed = true

		p := []byte("test")
		_, e := f.WriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("WriteAt with append flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY|os.O_APPEND, fs)

		p := []byte("test")
		_, e := f.WriteAt(p, 0)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Successful WriteAt updates content at offset", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)
		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove("testfile")

		originalContent := "original content"
		fi := getMockedFileInfo(t)
		fi.EXPECT().Size().Return(int64(len(originalContent)))

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObjectMeta(mock.Anything, mock.Anything, mock.Anything).
			Return(fi, nil)

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			GetObject(mock.Anything, mock.Anything, mock.Anything).
			Return(strings.NewReader(originalContent), utils.CleanUp(func() {}), nil)

		p := []byte("test")
		n, err := f.WriteAt(p, 8)

		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.True(t, f.dirty)

		f.preloadedFd.Seek(0, io.SeekStart)
		s, _ := io.ReadAll(f.preloadedFd)
		assert.Equal(t, "originaltesttent", string(s))
	})

	t.Run("WriteAt with negative offset return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_WRONLY, fs)
		fs.preloadFs = afero.NewMemMapFs()
		defer fs.preloadFs.Remove("testfile")

		p := []byte("test")
		_, err := f.WriteAt(p, -1)

		assert.Error(t, err)
	})
}

func TestFileReaddir(t *testing.T) {
	t.Run("Readdir with unreadable flag return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir", os.O_WRONLY, fs)
		f.isDir = true

		_, e := f.Readdir(10)

		assert.Error(t, e)
		assert.Equal(t, syscall.EPERM, e)
	})

	t.Run("Readdir on non-dir return error", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testfile", os.O_RDONLY, fs)
		f.isDir = false

		_, e := f.Readdir(10)

		assert.Error(t, e)
	})

	t.Run("Readdir success", func(t *testing.T) {
		fs := getMockedFs(t)
		f := getMockedFile("testdir/", os.O_RDONLY, fs)

		fi1 := getMockedFileInfo(t)
		fi2 := getMockedFileInfo(t)

		expectedFis := []os.FileInfo{
			fi1,
			fi2,
		}

		fs.manager.(*mocks.MockObjectManager).
			EXPECT().
			ListObjects(fs.ctx, fs.bucketName, fs.ensureAsDir(f.name), 10).
			Return(expectedFis, nil)

		fis, e := f.Readdir(10)

		assert.Nil(t, e)
		assert.Equal(t, len(expectedFis), len(fis))
		assert.Equal(t, expectedFis, fis)
	})
}
