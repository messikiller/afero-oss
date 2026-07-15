package ossfs

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewOssFsOptions(t *testing.T) {
	fs := NewOssFs("ak", "sk", "region", "bucket").
		WithContext(context.Background()).
		WithAutoSync(true).
		WithMaxPreloadSize(0)
	require.NotNil(t, fs.ctx)
	require.True(t, fs.autoSync)
	require.Zero(t, fs.maxPreloadSize)
	require.Equal(t, "OssFs", fs.Name())
}

func TestOpenDoesNotCreateMissingObject(t *testing.T) {
	fs, manager := newTestFs(nil)
	_, err := fs.Open("missing.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, manager.has("missing.txt"))
}

func TestOpenFileFlags(t *testing.T) {
	t.Run("exclusive create preserves existing data", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file.txt": "original"})
		_, err := fs.OpenFile("file.txt", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		require.ErrorIs(t, err, os.ErrExist)
		require.Equal(t, "original", manager.content("file.txt"))
	})

	t.Run("read only truncate is rejected", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file.txt": "original"})
		_, err := fs.OpenFile("file.txt", os.O_RDONLY|os.O_TRUNC, 0)
		require.ErrorIs(t, err, syscall.EINVAL)
		require.Equal(t, "original", manager.content("file.txt"))
	})

	t.Run("create truncates existing object", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file.txt": "original"})
		file, err := fs.Create("file.txt")
		require.NoError(t, err)
		require.Equal(t, "", manager.content("file.txt"))
		require.NoError(t, file.Close())
	})
}

func TestOpenReturnsIndependentHandles(t *testing.T) {
	fs, _ := newTestFs(map[string]string{"file.txt": "data"})
	first, err := fs.Open("file.txt")
	require.NoError(t, err)
	second, err := fs.Open("file.txt")
	require.NoError(t, err)
	require.NotSame(t, first, second)
	require.NoError(t, first.Close())

	buf := make([]byte, 4)
	_, err = second.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "data", string(buf))
}

func TestDirectoryLifecycle(t *testing.T) {
	fs, manager := newTestFs(nil)
	require.NoError(t, fs.MkdirAll("a/b/c", 0o755))
	for _, key := range []string{"a/", "a/b/", "a/b/c/"} {
		require.True(t, manager.has(key), key)
	}

	info, err := fs.Stat("a/b")
	require.NoError(t, err)
	require.True(t, info.IsDir())

	file, err := fs.Open("a/b/")
	require.NoError(t, err)
	entries, err := file.Readdir(-1)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "c", entries[0].Name())
	require.True(t, entries[0].IsDir())
}

func TestMkdirRejectsExistingPathAndCreatesNewMarker(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"existing/": ""})
	require.ErrorIs(t, fs.Mkdir("existing", 0o755), os.ErrExist)
	require.NoError(t, fs.Mkdir("new", 0o755))
	require.True(t, manager.has("new/"))
}

func TestImplicitDirectoryAndRootListing(t *testing.T) {
	fs, _ := newTestFs(map[string]string{
		"one.txt":        "1",
		"dir/a.txt":      "a",
		"dir/sub/b.txt":  "b",
		"other/data.txt": "d",
	})
	info, err := fs.Stat("dir")
	require.NoError(t, err)
	require.True(t, info.IsDir())

	root, err := fs.Open("/")
	require.NoError(t, err)
	names, err := root.Readdirnames(-1)
	require.NoError(t, err)
	require.Equal(t, []string{"dir", "one.txt", "other"}, names)
}

func TestRemoveAndRemoveAll(t *testing.T) {
	t.Run("remove rejects non-empty directory", func(t *testing.T) {
		fs, _ := newTestFs(map[string]string{"dir/": "", "dir/file": "data"})
		err := fs.Remove("dir")
		require.ErrorIs(t, err, syscall.ENOTEMPTY)
	})

	t.Run("RemoveAll handles files", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file": "data", "other": "keep"})
		require.NoError(t, fs.RemoveAll("file"))
		require.False(t, manager.has("file"))
		require.True(t, manager.has("other"))
	})

	t.Run("RemoveAll root clears every key", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"a": "1", "dir/b": "2"})
		require.NoError(t, fs.RemoveAll("/"))
		require.False(t, manager.has("a"))
		require.False(t, manager.has("dir/b"))
	})
}

func TestRenameNormalizesAndRecurses(t *testing.T) {
	fs, manager := newTestFs(map[string]string{
		"old/":         "",
		"old/file.txt": "data",
		"old/sub/x":    "x",
	})
	require.NoError(t, fs.Rename("/old/", "new//"))
	for _, key := range []string{"new/", "new/file.txt", "new/sub/x"} {
		require.True(t, manager.has(key), key)
	}
	for _, key := range []string{"old/", "old/file.txt", "old/sub/x"} {
		require.False(t, manager.has(key), key)
	}
}

func TestRenameEdgeCases(t *testing.T) {
	t.Run("same path is a no-op", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file": "data"})
		require.NoError(t, fs.Rename("/file", "file"))
		require.Equal(t, "data", manager.content("file"))
	})

	t.Run("file cannot replace directory", func(t *testing.T) {
		fs, manager := newTestFs(map[string]string{"file": "data", "dir/": ""})
		err := fs.Rename("file", "dir")
		require.ErrorIs(t, err, syscall.EISDIR)
		require.True(t, manager.has("file"))
	})

	t.Run("directory cannot move into itself", func(t *testing.T) {
		fs, _ := newTestFs(map[string]string{"dir/": "", "dir/file": "data"})
		err := fs.Rename("dir", "dir/sub")
		require.ErrorIs(t, err, syscall.EINVAL)
	})
}

func TestPathNormalization(t *testing.T) {
	fs, _ := newTestFs(nil)
	require.Equal(t, "a/c/", fs.normFileName("/a//b/../c/"))
	require.Equal(t, "", fs.normFileName("/"))
	require.Equal(t, "a/b", fs.normFileName(`a\b`))
}

func TestAferoHelpers(t *testing.T) {
	fs, manager := newTestFs(map[string]string{"existing.txt": "old"})
	require.NoError(t, afero.WriteFile(fs, "dir/new.txt", []byte("new"), 0o600))
	require.Equal(t, "new", manager.content("dir/new.txt"))

	data, err := afero.ReadFile(fs, "dir/new.txt")
	require.NoError(t, err)
	require.Equal(t, "new", string(data))

	entries, err := afero.ReadDir(fs, ".")
	require.NoError(t, err)
	require.Equal(t, []string{"dir", "existing.txt"}, []string{entries[0].Name(), entries[1].Name()})

	_, err = afero.ReadFile(fs, "missing.txt")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, manager.has("missing.txt"))
}

func TestUnsupportedMetadataOperations(t *testing.T) {
	fs, _ := newTestFs(nil)
	require.Error(t, fs.Chmod("x", 0))
	require.Error(t, fs.Chown("x", 0, 0))
	require.Error(t, fs.Chtimes("x", infoTime(), infoTime()))
}

func infoTime() (zeroTime time.Time) { return }
