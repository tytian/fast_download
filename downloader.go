package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// filePart 文件分片
type filePart struct {
	Index int    //文件分片的序号
	From  int    //开始byte
	To    int    //解决byte
	Data  []byte //http下载得到的文件内容
}

// FileDownloader 文件下载器
type FileDownloader struct {
	fileSize       int
	url            string
	outputFileName string
	totalPart      int //下载线程
	outputDir      string
	doneFilePart   []filePart
}


// NewFileDownloader .
func NewFileDownloader(url, outputFileName, outputDir string, totalPart int) *FileDownloader {
	if outputDir == "" {
		wd, err := os.Getwd() //获取当前工作目录
		if err != nil {
			log.Println(err)
		}
		outputDir = wd
	}
	return &FileDownloader{
		fileSize:       0,
		url:            url,
		outputFileName: outputFileName,
		outputDir:      outputDir,
		totalPart:      totalPart,
		doneFilePart:   make([]filePart, totalPart),
	}

}

func parseFileInfoFrom(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)

		if err != nil {
			panic(err)
		}
		return params["filename"]
	}
	filename := filepath.Base(resp.Request.URL.Path)
	return filename
}

// head 获取要下载的文件的基本信息(header) 使用HTTP Method Head
func (d *FileDownloader) head() (int, error) {
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode > 299 {
		return 0, errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}
	//检查是否支持 断点续传
	//https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Ranges
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, errors.New("服务器不支持文件断点续传")
	}

	d.outputFileName = parseFileInfoFrom(resp)
	//https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Length
	return strconv.Atoi(resp.Header.Get("Content-Length"))
}

// Run 开始下载任务
func (d *FileDownloader) Run() error {
	fileTotalSize, err := d.head()
	if err != nil {
		return err
	}
	d.fileSize = fileTotalSize
	log.Printf("文件大小： %dM", fileTotalSize/1024/1024)
	jobs := make([]filePart, d.totalPart)
	eachSize := fileTotalSize / d.totalPart

	for i := range jobs {
		jobs[i].Index = i
		if i == 0 {
			jobs[i].From = 0
		} else {
			jobs[i].From = jobs[i-1].To + 1
		}
		if i < d.totalPart-1 {
			jobs[i].To = jobs[i].From + eachSize
		} else {
			//the last filePart
			jobs[i].To = fileTotalSize - 1
		}
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(job filePart) {
			defer wg.Done()
			err := d.downloadPart(job)
			if err != nil {
				log.Println("下载文件失败:", err, job)
			}
		}(j)

	}
	wg.Wait()
	log.Println("所有分片文件下载完成 !!! ")
	return d.mergeFileParts()
}

// 下载分片
func (d FileDownloader) downloadPart(c filePart) error {
	req, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	log.Printf("开始[%d]下载from:%d to:%d\n", c.Index, c.From, c.To)

	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", c.From, c.To))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("服务器错误状态码: %v", resp.StatusCode))
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bs) != (c.To - c.From + 1) {
		return errors.New("下载文件分片长度错误")
	}
	c.Data = bs
	d.doneFilePart[c.Index] = c
	log.Printf("完成[%d]下载from:%d to:%d\n", c.Index, c.From, c.To)
	return nil

}

// getNewRequest 创建一个request
func (d FileDownloader) getNewRequest(method string) (*http.Request, error) {
	req, err := http.NewRequest(
		method,
		d.url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	return req, nil
}

// mergeFileParts 合并下载的文件
func (d FileDownloader) mergeFileParts() error {
	log.Println("开始合并文件")
	path := filepath.Join(d.outputDir, d.outputFileName)
	mergedFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer mergedFile.Close()
	hash := sha256.New()
	totalSize := 0
	for _, s := range d.doneFilePart {
		mergedFile.Write(s.Data)
		hash.Write(s.Data)
		totalSize += len(s.Data)
	}
	if totalSize != d.fileSize {
		return errors.New("文件不完整")
	}
	//https://download.jetbrains.com/go/goland-2023.3.exe?_gl=1*mup9mj*_ga*MTEwMTg4NzEyMy4xNzAyMzc2MzI4*_ga_9J976DJZ68*MTcwMjM3NjMyNy4xLjEuMTcwMjM3NjQzMC42MC4wLjA.&_ga=2.41896738.1284767415.1702376328-1101887123.1702376328
	if hex.EncodeToString(hash.Sum(nil)) != "a45dfcc33d6ff7b41a2a0bd9a52a15138845e99bbc97d7aab782c233914acb4e" {
		return errors.New("文件损坏")
	} else {
		log.Println("文件SHA-256校验成功")
	}
	return nil

}

