package main

// Исходники задания для первого занятия у других групп https://github.com/t0pep0/GB_best_go

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type TargetFile struct {
	Path string
	Name string
}

type FileList map[string]TargetFile

type FileInfo interface {
	os.FileInfo
	Path() string
}

type fileInfo struct {
	os.FileInfo
	path string
}

type DirData struct {
	files []FileInfo
	err   error
}

func (fi fileInfo) Path() string {
	return fi.path
}

type SearchData struct {
	mu            sync.RWMutex
	maxDepth      int
	dirToDepthMap map[string]int
	wg            sync.WaitGroup
	wantExt       string
	dirDataCh     chan DirData
	result        chan []FileInfo
}

func (data *SearchData) IncreaseMaxDepth() {
	data.mu.Lock()
	data.maxDepth += 2
	data.mu.Unlock()
}

func (data *SearchData) SaveCurrentDir(dir string, depth int) {
	data.mu.Lock()
	data.dirToDepthMap[dir] = depth
	data.mu.Unlock()
}

func (data *SearchData) RemoveCurrentDir(dir string) {
	data.mu.Lock()
	delete(data.dirToDepthMap, dir)
	data.mu.Unlock()
}

func (data *SearchData) GetCurrentDir() string {
	data.mu.RLock()
	defer data.mu.RUnlock()
	if len(data.dirToDepthMap) != 0 {
		for key := range data.dirToDepthMap {
			return key
		}
	}
	return ""
}

func (data *SearchData) GetCurrentDepth(dir string) int {
	data.mu.Lock()
	defer data.mu.Unlock()
	return data.dirToDepthMap[dir]
}

// Ограничить глубину поиска заданым числом
func (data *SearchData) ListDirectory(ctx context.Context, dir string, depth int) {
	defer data.wg.Done()
	select {
	case <-ctx.Done():
		data.dirDataCh <- DirData{
			files: nil,
			err:   nil,
		}
		return
	default:
		time.Sleep(time.Second * 1)
		data.SaveCurrentDir(dir, depth)
		defer data.RemoveCurrentDir(dir)
		depth++
		res, err := os.ReadDir(dir)
		if err != nil {
			data.dirDataCh <- DirData{
				files: nil,
				err:   err,
			}
			return
		}
		for _, entry := range res {
			path := filepath.Join(dir, entry.Name())
			if filepath.Ext(entry.Name()) == data.wantExt {
				info, err := entry.Info()
				if err != nil {
					data.dirDataCh <- DirData{
						files: nil,
						err:   err,
					}
				}
				result := make([]FileInfo, 0)
				result = append(result, fileInfo{info, path})
				data.dirDataCh <- DirData{
					files: result,
					err:   nil,
				}
			} else if entry.IsDir() && depth <= data.maxDepth {
				data.wg.Add(1)
				go func() {
					data.ListDirectory(ctx, path, depth)
				}()
			}
		}
	}
}

func (data *SearchData) WriteDirectory() {
	result := make([]FileInfo, 0)
	for n := range data.dirDataCh {
		result = append(result, n.files...)
	}

	data.result <- result
}

func main() {
	const wantExt = ".go"
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	data := SearchData{
		maxDepth:      2,
		dirToDepthMap: make(map[string]int),
		dirDataCh:     make(chan DirData),
		wantExt:       wantExt,
		result:        make(chan []FileInfo, 1),
	}

	data.wg.Add(1)

	go func() {
		wd, _ := os.Getwd()
		data.ListDirectory(ctx, wd, 0)
		data.wg.Wait()
		close(data.dirDataCh)
	}()

	go func() {
		data.WriteDirectory()
	}()

	go func() {
		signalType := <-sigCh

		switch signalType {
		// По SIGTERM увеличить глубину поиска на +2
		case syscall.SIGTERM:
			log.Println("Search depth will be increased (+2)")
			data.IncreaseMaxDepth()
		// По SIGHUP вывести текущую директорию и текущую глубину поиска
		case syscall.SIGHUP:
			log.Println("You will see current directory and search depth")

			currentDir := data.GetCurrentDir()
			fmt.Printf("\tDir: %s\t\t Depth: %d\n", currentDir, data.GetCurrentDepth(currentDir))
		default:
			log.Println("Signal received, terminate...")
			cancel()
			os.Exit(0)
		}
	}()
	res := <-data.result
	for _, f := range res {
		fmt.Printf("\tName: %s\t\t Path: %s\n", f.Name(), f.Path())
	}
	log.Println("Done")
}
