package data_collector

import (
	"context"
	iofs "io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	fileDir "github.com/mswatermelon/GB_march_best_practic/file_dir_info"
	"github.com/stretchr/testify/assert"
)

func createSearchData() SearchData {
	return SearchData{
		MaxDepth:      2,
		DirToDepthMap: make(map[string]int),
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
	}
}

func TestIsDirectoryIncreaseMaxDepth(t *testing.T) {
	searchData := createSearchData()
	searchData.increaseMaxDepth()

	assert.Equal(t, searchData.MaxDepth, 4)
}

func TestIsDirectorySaveCurrentDir(t *testing.T) {
	searchData := createSearchData()
	dir, depth := "/dir/dir", 2
	searchData.saveCurrentDir(dir, depth)

	assert.Equal(t, searchData.DirToDepthMap[dir], depth)
}

func TestIsDirectoryRemoveCurrentDir(t *testing.T) {
	dirToDepthMap := make(map[string]int)
	dir, depth := "/dir/dir", 2
	dirToDepthMap[dir] = depth
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: dirToDepthMap,
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
	}
	searchData.removeCurrentDir(dir)

	assert.Equal(t, searchData.DirToDepthMap[dir], 0)
}

func TestIsDirectoryGetCurrentDir(t *testing.T) {
	dirToDepthMap := make(map[string]int)
	dir, depth := "/dir/dir", 2
	dirToDepthMap[dir] = depth
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: dirToDepthMap,
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
	}

	assert.Equal(t, searchData.getCurrentDir(), dir)
}

func TestIsDirectoryGetCurrentDirIfEmpty(t *testing.T) {
	searchData := createSearchData()

	assert.Equal(t, searchData.getCurrentDir(), "")
}

func TestIsDirectoryGetCurrentDepthIfEmpty(t *testing.T) {
	dir := "/dir/dir"
	searchData := createSearchData()

	assert.Equal(t, searchData.getCurrentDepth(dir), 0)
}

func TestIsDirectoryGetCurrentDepth(t *testing.T) {
	dirToDepthMap := make(map[string]int)
	dir, depth := "/dir/dir", 2
	dirToDepthMap[dir] = depth
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: dirToDepthMap,
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
	}

	assert.Equal(t, searchData.getCurrentDepth(dir), depth)
}

type PathData struct {
	name string
	path string
}

type DirStub struct {
	name string
}

func (dir *DirStub) Name() string {
	return dir.name
}

func (dir *DirStub) IsDir() bool {
	return false
}

func (dir *DirStub) Type() iofs.FileMode {
	var value iofs.FileMode
	return value
}

type FileInfoStub struct {
	name string
}

func (dir *FileInfoStub) Name() string {
	return dir.name
}

func (dir *FileInfoStub) Size() int64 {
	var value int64
	return value
}

func (dir *FileInfoStub) Mode() iofs.FileMode {
	var value iofs.FileMode
	return value
}

func (dir *FileInfoStub) ModTime() time.Time {
	var value time.Time
	return value
}

func (dir *FileInfoStub) IsDir() bool {
	return false
}

func (dir *FileInfoStub) Sys() interface{} {
	return nil
}

func (dir *DirStub) Info() (iofs.FileInfo, error) {
	return &FileInfoStub{
		name: dir.name,
	}, nil
}

func ReadDir(dirname string) ([]iofs.DirEntry, error) {
	f := DirStub{
		name: "main.go",
	}
	dir := []iofs.DirEntry{
		&f,
	}

	return dir, nil
}

func TestCollectData(t *testing.T) {
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: make(map[string]int),
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
	}

	searchData.ReadDir = ReadDir

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res := searchData.CollectData(ctx)
	wd, _ := os.Getwd()
	checkSlice := []PathData{
		{
			name: "main.go",
			path: filepath.Join(wd, "main.go"),
		},
	}
	for i, f := range res {
		assert.Equal(t, checkSlice[i].name, f.Name())
		assert.Equal(t, checkSlice[i].path, f.Path())
	}
}

func TestCollectDataNotFound(t *testing.T) {
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: make(map[string]int),
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".csv",
		Result:        make(chan []fileDir.FileInfo, 1),
	}

	searchData.ReadDir = ReadDir

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res := searchData.CollectData(ctx)
	assert.Equal(t, len(res), 0)
}
