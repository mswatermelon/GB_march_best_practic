package file_dir_info

import (
	"os"
)

type FileInfo interface {
	os.FileInfo
	Path() string
}

type fileInfo struct {
	os.FileInfo
	path string
}

func (fi fileInfo) Path() string {
	return fi.path
}

func NewFileInfo(info os.FileInfo, path string) FileInfo {
	return &fileInfo{info, path}
}
