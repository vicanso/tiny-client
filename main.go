package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/vicanso/tiny-client/service"
)

const (
	// LogLevelAll log all
	LogLevelAll = iota
	// LogLevelError log error
	LogLevelError
)

func main() {
	path := flag.String("path", ".", "search path")
	target := flag.String("target", "", "optim target path(new image will save to this path)")
	imageType := flag.String("type", "png,jpg,jpeg", "image ext list")
	logType := flag.String("log", "error", "log type: error or all")

	server := flag.String("server", "localhost:6010", "grpc server address")
	q := flag.String("quality", "80", "quality of image, 0-100")

	flag.Parse()

	if *target == "" {
		panic("target path can not be nil")
	}

	targetPath, err := filepath.Abs(*target)
	if err != nil {
		panic(err)
	}

	quality, err := strconv.Atoi(*q)
	if err != nil {
		panic(err)
	}
	if quality < 0 || quality > 100 {
		panic(errors.New("quality should be >= 0 and <= 100"))
	}

	logLevel := LogLevelAll
	if *logType == "error" {
		logLevel = LogLevelError
	}

	absPath, err := filepath.Abs(*path)
	if err != nil {
		panic(err)
	}
	startedAt := time.Now()
	arr := strings.Split(*imageType, ",")
	reg := ".(" + strings.Join(arr, "|") + ")$"
	matches, err := service.Glob(absPath, reg)
	if err != nil {
		panic(err)
	}
	conn, err := service.GetConnection(*server)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	limiter := make(chan bool, 5)
	var successCount uint32
	var failCount uint32
	var wg sync.WaitGroup
	var originalSizeCount uint64
	var newSizeCount uint64
	for _, file := range matches {
		limiter <- true
		go func(file string) {
			defer func() {
				wg.Done()
				<-limiter
			}()
			wg.Add(1)
			ext := filepath.Ext(file)
			shortFile := file[len(absPath):]
			buf, err := ioutil.ReadFile(file)
			if err != nil {
				fmt.Println("read file fail,", file, err)
				atomic.AddUint32(&failCount, 1)
				return
			}
			params := &service.OptimParams{
				Data:    buf,
				Type:    ext[1:],
				Quality: quality,
			}
			data, err := service.Optim(conn, params)
			if err != nil {
				fmt.Println("optim file fail,", shortFile, err)
				atomic.AddUint32(&failCount, 1)
				return
			}
			if len(data) > len(buf) {
				data = buf
			}
			newFile := targetPath + shortFile
			os.MkdirAll(filepath.Dir(newFile), os.ModePerm)
			err = os.WriteFile(targetPath+shortFile, data, 0666)
			if err != nil {
				fmt.Println("write file fail,", shortFile, err)
				atomic.AddUint32(&failCount, 1)
				return
			}
			newSize := uint64(len(data))
			originalSize := uint64(len(buf))
			atomic.AddUint64(&originalSizeCount, originalSize)
			atomic.AddUint64(&newSizeCount, newSize)
			if logLevel == LogLevelAll {
				fmt.Println(shortFile, humanize.Bytes(originalSize), "-->", humanize.Bytes(newSize))
			}
			atomic.AddUint32(&successCount, 1)
		}(file)
	}
	wg.Wait()

	template := `
********************************TINY********************************
Optimize Images is done, use:%s
Success(%d) Fail(%d) 
Space size reduce from %s to %s
********************************TINY********************************`
	fmt.Println(fmt.Sprintf(template,
		time.Since(startedAt).String(),
		successCount,
		failCount,
		humanize.Bytes(originalSizeCount),
		humanize.Bytes(newSizeCount),
	))
}
