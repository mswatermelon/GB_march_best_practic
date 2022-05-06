package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	collector "github.com/mswatermelon/GB_march_best_practic/data_collector"
	fileDir "github.com/mswatermelon/GB_march_best_practic/file_dir_info"
)

func main() {
	const wantExt = ".go"
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	dataCollector := collector.NewCollector(
		2,
		wantExt,
		make(map[string]int),
		make(chan fileDir.DirData),
		make(chan []fileDir.FileInfo, 1),
	)

	res := dataCollector.CollectData(ctx)
	fileDir.OutputData(res)

	log.Println("Done")
}
