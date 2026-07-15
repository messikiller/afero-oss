package ossfs

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestReadRangeShortReadAndOffset(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "abcdef"})
	file, err := fs.Open("file")
	require.NoError(t, err)
	f := file.(*File)

	first := make([]byte, 4)
	n, err := f.Read(first)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "abcd", string(first))

	second := make([]byte, 4)
	n, err = f.Read(second)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 2, n)
	require.Equal(t, "ef", string(second[:n]))
	require.Equal(t, int64(6), f.offset)
	require.Equal(t, [][2]int64{{0, 3}, {4, 7}}, manager.ranges)
}

func TestReadAtRejectsNegativeOffsetAndHandlesEmptyBuffer(t *testing.T) {
	fs, _ := newTestFs(map[string]string{"file": "data"})
	file, err := fs.Open("file")
	require.NoError(t, err)
	_, err = file.ReadAt(make([]byte, 1), -1)
	require.ErrorIs(t, err, syscall.ERANGE)
	n, err := file.Read(nil)
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestWriteThenReadUsesLocalPreload(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "abcdef"})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("XY"), 2)
	require.NoError(t, err)

	buf := make([]byte, 6)
	n, err := file.ReadAt(buf, 0)
	require.NoError(t, err)
	require.Equal(t, 6, n)
	require.Equal(t, "abXYef", string(buf))
	require.Equal(t, "abcdef", manager.content("file"), "unsynced data must remain local")

	require.NoError(t, file.Sync())
	require.Equal(t, "abXYef", manager.content("file"))
}

func TestTruncateShrinksAndExtendsWithoutLargeAllocation(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "abcdef"})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	require.ErrorIs(t, file.Truncate(-1), afero.ErrOutOfRange)
	require.NoError(t, file.Truncate(3))

	info, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, int64(3), info.Size())
	require.NoError(t, file.Truncate(5))
	require.NoError(t, file.Close())
	require.Equal(t, []byte{'a', 'b', 'c', 0, 0}, []byte(manager.content("file")))
}

func TestSyncUploadsOnlyDirtyData(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "old"})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("new"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	require.Equal(t, 1, manager.putCount("file"))
	require.NoError(t, file.Sync())
	require.NoError(t, file.Close())
	require.Equal(t, 1, manager.putCount("file"))
}

func TestCloseFailureKeepsDirtyHandleRetryable(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "old"})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("new"), 0)
	require.NoError(t, err)
	manager.putErr = errors.New("upload failed")

	err = file.Close()
	require.EqualError(t, err, "upload failed")
	require.False(t, file.(*File).closed)
	require.NoError(t, file.Close())
	require.Equal(t, "new", manager.content("file"))
}

func TestAutoSyncPropagatesUploadError(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "old"})
	fs.WithAutoSync(true)
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	manager.putErr = errors.New("upload failed")
	_, err = file.WriteAt([]byte("new"), 0)
	require.EqualError(t, err, "upload failed")
	require.True(t, file.(*File).dirty)
	require.NoError(t, file.Sync())
	require.Equal(t, "new", manager.content("file"))
}

func TestPreloadFailureCleansTemporaryFile(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "data"})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	manager.getErr = errors.New("download failed")
	_, err = file.Write([]byte("x"))
	require.EqualError(t, err, "download failed")
	entries, err := afero.ReadDir(fs.preloadFs, ".")
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestPreloadLimitIsConfigurable(t *testing.T) {
	fs, _ := newTestFs(map[string]string{"file": "1234"})
	fs.WithMaxPreloadSize(3)
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	_, err = file.Write([]byte("x"))
	require.ErrorContains(t, err, "file too large for preload")

	fs.WithMaxPreloadSize(0)
	file, err = fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	_, err = file.Write([]byte("x"))
	require.NoError(t, err)
}

func TestHandlesUseIndependentPreloadFiles(t *testing.T) {
	fs, _ := newTestFs(map[string]string{"file": "data"})
	firstRaw, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	secondRaw, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	first := firstRaw.(*File)
	second := secondRaw.(*File)
	_, err = first.WriteAt([]byte("A"), 0)
	require.NoError(t, err)
	_, err = second.WriteAt([]byte("B"), 1)
	require.NoError(t, err)
	require.NotEqual(t, first.preloadedKey, second.preloadedKey)
	require.NoError(t, first.Close())
	require.False(t, second.closed)
	require.ErrorIs(t, second.Close(), ErrWriteConflict)
	require.False(t, second.closed, "a conflicted handle keeps its local data for recovery")
}

func TestConcurrentWritesOnOneHandle(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": strings.Repeat("0", 64)})
	file, err := fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, writeErr := file.WriteAt([]byte{byte('A' + i)}, int64(i))
			errs <- writeErr
		}(i)
	}
	wg.Wait()
	close(errs)
	for writeErr := range errs {
		require.NoError(t, writeErr)
	}
	require.NoError(t, file.Close())
	require.Equal(t, "ABCDEFGH", manager.content("file")[:8])
}

func TestAppendAndSeekBeyondEnd(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"file": "abc"})
	file, err := fs.OpenFile("file", os.O_RDWR|os.O_APPEND, 0)
	require.NoError(t, err)
	_, err = file.Write([]byte("d"))
	require.NoError(t, err)
	_, err = file.Write([]byte("e"))
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "abcde", manager.content("file"))

	file, err = fs.OpenFile("file", os.O_RDWR, 0)
	require.NoError(t, err)
	off, err := file.Seek(10, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(10), off)
}

func TestReaddirMaintainsPerHandleCursor(t *testing.T) {
	fs, _ := newTestFs(map[string]string{
		"dir/a": "a",
		"dir/b": "b",
		"dir/c": "c",
	})
	file, err := fs.Open("dir")
	require.NoError(t, err)
	first, err := file.Readdirnames(2)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, first)
	second, err := file.Readdirnames(2)
	require.NoError(t, err)
	require.Equal(t, []string{"c"}, second)
	_, err = file.Readdirnames(1)
	require.ErrorIs(t, err, io.EOF)
}

func TestClosedFileOperationsFail(t *testing.T) {
	fs, _ := newTestFs(map[string]string{"file": "data"})
	file, err := fs.Open("file")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	_, err = file.Read(make([]byte, 1))
	require.ErrorIs(t, err, afero.ErrFileClosed)
	require.ErrorIs(t, file.Close(), afero.ErrFileClosed)
}
