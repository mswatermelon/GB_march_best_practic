package main

//Исходники задания для первого занятия у других групп https://github.com/t0pep0/GB_best_go

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"go.uber.org/zap"
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

type FileSearcher struct {
	logger *zap.Logger
}

func NewFileSearcher(logger *zap.Logger) *FileSearcher {
	logger.Debug("Creating file searcher")
	return &FileSearcher{
		logger: logger,
	}
}

//Ограничить глубину поиска заданым числом, по SIGUSR2 увеличить глубину поиска на +2
func (f *FileSearcher) listDirectory(ctx context.Context, dir string) ([]FileInfo, error) {
	select {
	case <-ctx.Done():
		f.logger.Info("Context is done, skipping dir", zap.String("dir", dir))
		return nil, nil
	default:
		//По SIGUSR1 вывести текущую директорию и текущую глубину поиска
		time.Sleep(time.Second * 10)
		var result []FileInfo
		res, err := os.ReadDir(dir)
		if err != nil {
			f.logger.Error("Could not read directory", zap.Error(err),
				zap.String("dir", dir))
			return nil, err
		}
		for _, entry := range res {
			f.logger.Debug("Investigate if entry is file or folder", zap.String("name", entry.Name()))
			path := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				f.logger.Debug("This is a dir", zap.String("name", entry.Name()))
				child, err := f.listDirectory(ctx, path) //Дополнительно: вынести в горутину
				if err != nil {
					f.logger.Error("Could not list directory", zap.Error(err))
					return result, err
				}
				result = append(result, child...)
			} else {
				f.logger.Debug("This is file", zap.String("name", entry.Name()))
				info, err := entry.Info()
				if err != nil {
					f.logger.Error("Could not read file info", zap.Error(err))
					return nil, err
				}
				result = append(result, fileInfo{info, path})
				f.logger.Debug("File's path", zap.String("path", path))
			}
		}
		return result, nil
	}
}

func (f *FileSearcher) findFiles(ctx context.Context, ext string) (FileList, error) {
	wd, err := os.Getwd()
	if err != nil {
		f.logger.Error("Could not get working directory", zap.Error(err))
		return nil, err
	}
	files, err := f.listDirectory(ctx, wd)
	if err != nil {
		if len(files) == 0 {
			f.logger.Error("Error on get file list", zap.Error(err))
			return nil, err
		}
		f.logger.Warn("Error on get part file list", zap.Error(err))
	}
	fl := make(FileList, len(files))
	for _, file := range files {
		f.logger.Debug("Checking file...", zap.String("name", file.Name()))
		fileExt := filepath.Ext(file.Name())
		f.logger.Debug("Compare extentions",
			zap.String("target_ext", ext),
			zap.String("current", fileExt))
		if fileExt == ext {
			name := file.Name()
			path := file.Path()
			f.logger.Debug("Took into account the file",
				zap.String("name", name),
				zap.String("path", path))
			fl[name] = TargetFile{
				Name: name,
				Path: path,
			}
		}
	}
	return fl, nil
}

var (
	GitHash = ""
	BuildTime = ""
	Version = ""
)

func main() {
	const (
		wantExt = ".go"
		production = "PRODUCTION"
		env = "ENV"
	)
	var logger *zap.Logger
	curEnv := os.Getenv(env)
	var err error
	if curEnv == production {
		logCfg := zap.NewProductionConfig()
		logCfg.OutputPaths = []string{"stderr"}
		logger, err = logCfg.Build()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		log.Fatal("Failed to initialize logger", err)
	}
	logger.Info("Starting", zap.Int("pid", os.Getpid()),
		zap.String("commit_hash", GitHash), zap.String("BuildTime", BuildTime),
			zap.String("version", Version))
	logger.Debug("We are in environment:", zap.String("env", curEnv))

	defer logger.Sync()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	//Обработать сигнал SIGUSR1
	waitCh := make(chan struct{})
	fileSearcher := NewFileSearcher(logger)
	go func() {
		res, err := fileSearcher.findFiles(ctx, wantExt)
		if err != nil {
			logger.Error("Error on search: ", zap.Error(err))
			os.Exit(1)
		}
		for _, f := range res {
			fmt.Printf("\tName: %s\t\t Path: %s\n", f.Name, f.Path)
		}
		waitCh <- struct{}{}
	}()
	go func() {
		<-sigCh
		logger.Info("Signal received, terminate...")
		cancel()
	}()
	logger.Debug("Waiting all goroutines will be finished...")
	//Дополнительно: Ожидание всех горутин перед завершением
	<-waitCh
	logger.Info("Done")
}
