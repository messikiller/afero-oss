package ossfs

import (
	"time"

	"github.com/messikiller/afero-oss/internal/utils"
)

type FileInfo struct {
	*utils.OssObjectMeta
}

func NewFileInfo(name string, size int64, updatedAt time.Time) *FileInfo {
	return &FileInfo{
		OssObjectMeta: utils.NewOssObjectMeta(name, size, updatedAt),
	}
}

func NewDirFileInfo(name string, updatedAt time.Time) *FileInfo {
	return &FileInfo{
		OssObjectMeta: utils.NewOssDirMeta(name, updatedAt),
	}
}
