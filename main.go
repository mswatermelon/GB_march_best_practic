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
	sync.Mutex
	depth int
	current int
	lastSignalType os.Signal
	waitCh *chan struct{}
}

// Ограничить глубину поиска заданым числом
func ListDirectory(ctx context.Context, dir string, data *SearchData) ([]FileInfo, error) {
	*data.waitCh <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		switch data.lastSignalType {
		// По SIGINT увеличить глубину поиска на +2
		case syscall.SIGINT:
			data.Lock()
			data.depth+=2
			data.Unlock()
		// По SIGHUP вывести текущую директорию и текущую глубину поиска
		case syscall.SIGHUP:
			fmt.Printf("\tDir: %s\t\t Depth: %s\n", dir, data.depth)

		}
		time.Sleep(time.Second * 10)
		var result []FileInfo
		res, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range res {
			data.current = 0
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				fmt.Println(data.current, data.depth, path)
				if data.current < data.depth {
					data.current++
					child, err := ListDirectory(ctx, path, data) // Дополнительно: вынести в горутину
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

func FindFiles(ctx context.Context, ext string, data *SearchData) (FileList, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	files, err := ListDirectory(ctx, wd, data)
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
	data := SearchData{depth: 2, waitCh: &waitCh}
	go func() {
		defer close(waitCh)
		res, err := FindFiles(ctx, wantExt, &data)
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

		data.Lock()
		data.lastSignalType = signalType
		data.Unlock()

		switch signalType {
		case syscall.SIGINT:
			log.Println("Search depth will be increased (+2)")
		// Обработать сигнал SIGHUP
		case syscall.SIGHUP:
			log.Println("You will see current directory and search depth")
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
