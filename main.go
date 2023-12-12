package main

import (
	"log"
	"time"
)

func main() {
	startTime := time.Now()
	var url string //下载文件的地址
	url = "https://download.jetbrains.com/go/goland-2023.3.exe"
	downloader := NewFileDownloader(url, "", "", 10)
	if err := downloader.Run(); err != nil {
		log.Fatal(err)
	}
	log.Printf("\n文件下载完成耗时: %f second\n", time.Now().Sub(startTime).Seconds())
}

