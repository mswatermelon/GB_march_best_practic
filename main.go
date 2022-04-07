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

func (fi fileInfo) Path() string {
	return fi.path
}

type SearchData struct {
	mu            sync.RWMutex
	maxDepth      int
	dirToDepthMap map[string]int
	waitCh        *chan struct{}
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
func (data *SearchData) ListDirectory(ctx context.Context, dir string, depth int) ([]FileInfo, error) {
	*data.waitCh <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		time.Sleep(time.Second * 2)
		var result []FileInfo
		data.SaveCurrentDir(dir, depth)
		defer data.RemoveCurrentDir(dir)
		depth++
		res, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range res {
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				if depth <= data.maxDepth {
					child, err := data.ListDirectory(ctx, path, depth) // Дополнительно: вынести в горутину
					if err != nil {
						return nil, err
					}
					result = append(result, child...)
				}
			} else {
				info, err := entry.Info()
				if err != nil {
					return nil, err
				}
				result = append(result, fileInfo{info, path})
			}
		}
		return result, nil
	}
}

func (data *SearchData) FindFiles(ctx context.Context, ext string) (FileList, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	files, err := data.ListDirectory(ctx, wd, 0)
	if err != nil {
		return nil, err
	}
	fl := make(FileList, len(files))
	for _, file := range files {
		if filepath.Ext(file.Name()) == ext {
			fl[file.Name()] = TargetFile{
				Name: file.Name(),
				Path: file.Path(),
			}
		}
	}
	return fl, nil
}

func main() {
	const wantExt = ".go"
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	waitCh := make(chan struct{})
	data := SearchData{maxDepth: 2, waitCh: &waitCh, dirToDepthMap: make(map[string]int)}
	go func() {
		defer close(waitCh)
		res, err := data.FindFiles(ctx, wantExt)
		if err != nil {
			log.Printf("Error on search: %v\n", err)
			os.Exit(1)
		}
		for _, f := range res {
			fmt.Printf("\tName: %s\t\t Path: %s\n", f.Name, f.Path)
		}
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
		}
	}()
	// Дополнительно: Ожидание всех горутин перед завершением
	for range waitCh {
		<-waitCh
	}
	cancel()
	log.Println("Done")
}
