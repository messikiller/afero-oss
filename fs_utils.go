package ossfs

import (
	"path"
	"strings"
)

func (fs *Fs) isDir(s string) bool {
	sep := fs.separator
	if fs.separator == "" {
		sep = "/"
	}
	return strings.HasSuffix(s, sep)
}

func (fs *Fs) ensureAsDir(s string) string {
	sep := fs.separator
	if fs.separator == "" {
		sep = "/"
	}
	s = fs.normFileName(s)
	if s == "" {
		return ""
	}
	if !strings.HasSuffix(s, sep) {
		s = s + sep
	}
	return s
}

func (fs *Fs) normFileName(s string) string {
	s = strings.ReplaceAll(s, "\\", "/")
	directory := strings.HasSuffix(s, "/")
	s = strings.TrimPrefix(path.Clean("/"+s), "/")
	if s == "." {
		s = ""
	}
	if directory && s != "" {
		s += "/"
	}
	return s
}
