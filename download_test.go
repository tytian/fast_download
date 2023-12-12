package main

import (
	"log"
	"testing"
	"time"
)

func TestDownloader(t *testing.T) {
	startTime := time.Now()
	var url string //下载文件的地址
	url = "https://markdown.com.cn/basic-syntax/links.html"
	downloader := NewFileDownloader(url, "", "", 10)
	if err := downloader.Run(); err != nil {
		log.Fatal(err)
	}
	log.Printf("\n文件下载完成耗时: %f second\n", time.Now().Sub(startTime).Seconds())
}
