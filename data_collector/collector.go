package data_collector

import (
	"context"
	"fmt"
	fileDir "github.com/mswatermelon/GB_march_best_practic/file_dir_info"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type DataCollector interface {
	CollectData(ctx context.Context)  []fileDir.FileInfo
}

func NewCollector(
	maxDepth      int,
	wantExt       string,
	dirToDepthMap map[string]int,
	dirDataCh chan fileDir.DirData,
	result    chan []fileDir.FileInfo,
) DataCollector {
	return &SearchData{
		MaxDepth:      maxDepth,
		DirToDepthMap: dirToDepthMap,
		DirDataCh:     dirDataCh,
		WantExt:       wantExt,
		Result:        result,
	}
}

type SearchData struct {
	mu            sync.RWMutex
	MaxDepth      int
	DirToDepthMap map[string]int
	Wg            sync.WaitGroup
	WantExt       string
	DirDataCh chan fileDir.DirData
	Result    chan []fileDir.FileInfo
}

func (data *SearchData) CollectData(ctx context.Context) []fileDir.FileInfo {
	data.addWaitGroup()

	go func() {
		wd, _ := os.Getwd()
		data.listDirectory(ctx, wd, 0)
		data.waitForGroup()
		data.closeDataCh()
	}()

	go func() {
		data.writeDirectory()
	}()

	sigCh := make(chan os.Signal, 1)

	go func() {
		signalType := <-sigCh

		switch signalType {
		// По SIGTERM увеличить глубину поиска на +2
		case syscall.SIGTERM:
			log.Println("Search depth will be increased (+2)")
			data.increaseMaxDepth()
		// По SIGHUP вывести текущую директорию и текущую глубину поиска
		case syscall.SIGHUP:
			log.Println("You will see current directory and search depth")

			currentDir := data.getCurrentDir()
			fmt.Printf("\tDir: %s\t\t Depth: %d\n", currentDir, data.getCurrentDepth(currentDir))
		default:
			log.Println("Signal received, terminate...")
			os.Exit(0)
		}
	}()

	return <-data.Result
}

func (data *SearchData) addWaitGroup() {
	data.Wg.Add(1)
}

func (data *SearchData) waitForGroup() {
	data.Wg.Wait()
}

func (data *SearchData) closeDataCh() {
	close(data.DirDataCh)
}

func (data *SearchData) increaseMaxDepth() {
	data.mu.Lock()
	data.MaxDepth += 2
	data.mu.Unlock()
}

func (data *SearchData) saveCurrentDir(dir string, depth int) {
	data.mu.Lock()
	data.DirToDepthMap[dir] = depth
	data.mu.Unlock()
}

func (data *SearchData) removeCurrentDir(dir string) {
	data.mu.Lock()
	delete(data.DirToDepthMap, dir)
	data.mu.Unlock()
}

func (data *SearchData) getCurrentDir() string {
	data.mu.RLock()
	defer data.mu.RUnlock()
	if len(data.DirToDepthMap) != 0 {
		for key := range data.DirToDepthMap {
			return key
		}
	}
	return ""
}

func (data *SearchData) getCurrentDepth(dir string) int {
	data.mu.Lock()
	defer data.mu.Unlock()
	return data.DirToDepthMap[dir]
}

// Ограничить глубину поиска заданым числом
func (data *SearchData) listDirectory(ctx context.Context, dir string, depth int) {
	defer data.Wg.Done()
	select {
	case <-ctx.Done():
		data.DirDataCh <- fileDir.DirData{
			Files: nil,
			Err:   nil,
		}
		return
	default:
		time.Sleep(time.Second * 1)
		data.saveCurrentDir(dir, depth)
		defer data.removeCurrentDir(dir)
		depth++
		res, err := os.ReadDir(dir)
		if err != nil {
			data.DirDataCh <- fileDir.DirData{
				Files: nil,
				Err:   err,
			}
			return
		}
		for _, entry := range res {
			path := filepath.Join(dir, entry.Name())
			if filepath.Ext(entry.Name()) == data.WantExt {
				info, err := entry.Info()
				if err != nil {
					data.DirDataCh <- fileDir.DirData{
						Files: nil,
						Err:   err,
					}
				}
				result := make([]fileDir.FileInfo, 0)
				result = append(result, fileDir.NewFileInfo(info, path))
				data.DirDataCh <- fileDir.DirData{
					Files: result,
					Err:   nil,
				}
			} else if entry.IsDir() && depth <= data.MaxDepth {
				data.Wg.Add(1)
				go func() {
					data.listDirectory(ctx, path, depth)
				}()
			}
		}
	}
}

func (data *SearchData) writeDirectory() {
	result := make([]fileDir.FileInfo, 0)
	for n := range data.DirDataCh {
		result = append(result, n.Files...)
	}

	data.Result <- result
}
