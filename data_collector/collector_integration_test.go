//go:build integration

package data_collector

import (
	"context"
	"os"
	"testing"
	"time"

	fileDir "github.com/mswatermelon/GB_march_best_practic/file_dir_info"
	"github.com/stretchr/testify/assert"
)

func TestCollectDataIntegration(t *testing.T) {
	searchData := SearchData{
		MaxDepth:      2,
		DirToDepthMap: make(map[string]int),
		DirDataCh:     make(chan fileDir.DirData),
		WantExt:       ".go",
		Result:        make(chan []fileDir.FileInfo, 1),
		ReadDir:       os.ReadDir,
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res := searchData.CollectData(ctx)

	checkSlice := []PathData{
		{
			name: "collector.go",
			path: "C:\\Users\\V.Aristarkhova\\GolandProjects\\GB_march_best_practic\\data_collector\\collector.go",
		},
		{
			name: "collector_integration_test.go",
			path: "C:\\Users\\V.Aristarkhova\\GolandProjects\\GB_march_best_practic\\data_collector\\collector_integration_test.go",
		},
		{
			name: "collector_test.go",
			path: "C:\\Users\\V.Aristarkhova\\GolandProjects\\GB_march_best_practic\\data_collector\\collector_test.go",
		},
	}
	for i, f := range res {
		assert.Equal(t, checkSlice[i].name, f.Name())
		assert.Equal(t, checkSlice[i].path, f.Path())
	}
}
