package downloader

import (
		"fmt"
		"godl/config"
		"godl/extractor/youtube"
		"godl/httpclient"
		"godl/progress"
		"io"
		"net/http"
		"os"
		"os/exec"
		"sort"
		"strings"
		"sync"
		"sync/atomic"
		"time"
)


type Downloader struct {
		Client      *http.Client
		configs			*config.Config
		Url         string
		outFile			string
		config      config.DownloaderConfig
}


type ConfigDownloader struct {
		Url 			string
		Threads 	int
		DiskCache int
}

type Chunk struct {
		Data  []byte
		Offset int64
}

var downloaderClient = httpclient.DownloaderClient


var bufPool = sync.Pool{
		New: func() interface{} { return make([]byte, 32*1024) }, // 32KB
}

func NewDownloader(cfg *config.Config) Downloader {
		return Downloader{
				configs: cfg,
				config: *cfg.DownloaderCfg,
				Url: cfg.Url,
				outFile: cfg.OutFile,

		}
}

func (d *Downloader)startGlobalDiskCache(file *os.File, chunkChan <-chan Chunk, done chan<- bool) {
		cache := make(map[int64][]byte)
		currentSize := 0
		maxCacheSize := d.config.DiskCache

		mergedData := make([]byte, maxCacheSize)

		flush := func() {
				if len(cache) == 0 {
						return
				}

				offsets := make([]int64, 0, len(cache))
				for o := range cache {
						offsets = append(offsets, o)
				}
				sort.Slice(offsets, func(i, j int) bool {
						return offsets[i] < offsets[j]
				})

				for i := 0; i < len(offsets); {
						startOffset := offsets[i]

						firstChunk := cache[startOffset]
						copy(mergedData, firstChunk)
						totalMerged := len(firstChunk)
						nextOffset := startOffset + int64(len(cache[startOffset]))
						//fmt.Printf("startOffset : %d, nextOffset : %d\n", startOffset, nextOffset)

						j := i + 1
						for j < len(offsets) && offsets[j] == nextOffset {
								//fmt.Printf("offsets(j) == nextOffset : %v\n",offsets[j] == nextOffset)
								nextChunk := cache[offsets[j]]

								if int64(totalMerged + len(nextChunk)) <= maxCacheSize {
										copy(mergedData[totalMerged:], nextChunk)
										totalMerged += len(nextChunk)

										nextOffset += int64(len(cache[offsets[j]]))
										j++
								} else { break }
						}

						file.WriteAt(mergedData[:totalMerged], startOffset)
						//fmt.Printf("flushing %d to offset: %d\n", len(mergedData[:totalMerged]), startOffset)
						i = j 
				}

				// Reset cache setelah sukses flush
				cache = make(map[int64][]byte)
				currentSize = 0
				file.Sync() // Sync untuk paksa write ke disk 
		}

		for chunk := range chunkChan {
				cache[chunk.Offset] = chunk.Data
				currentSize += len(chunk.Data)

				if int64(currentSize) >= maxCacheSize {
						flush()
				}
		}

		//Flush terakhir kalau cache kurang dari 16MMb
		flush()

		done <- true
}

func (d *Downloader)downloadAll(client *http.Client, url string, file *os.File, downloaded *int64, chunkChan chan<- Chunk) error {

		maxRetry := 5

		for attempt := 0; attempt < maxRetry; attempt++ {

				req, _ := httpclient.NewDefaultWebRequest(url)

				resp, err := client.Do(req)
				if err != nil {
						fmt.Println("retry (request error):", err)
						time.Sleep(time.Second)
						continue
				}

				//buf := make([]byte, 32*1024)
				var offset int64 = 0
				buf := bufPool.Get().([]byte)
				defer bufPool.Put(buf)

				for {
						n, err := resp.Body.Read(buf)

						if n > 0 {

								dataCopy := make([]byte, n)
								copy(dataCopy, buf[:n])

								chunkChan<-Chunk{Offset: offset, Data: dataCopy}
								offset += int64(n)
								atomic.AddInt64(downloaded, int64(n))
						}

						if err == io.EOF {
								resp.Body.Close()
								return nil
						}

						if err != nil {
								fmt.Println("connection error:", err)
								resp.Body.Close()
								time.Sleep(time.Second)
								break // keluar loop read → retry request
						}
				}
		}

		return fmt.Errorf("Download failed after retries")
}


func (d *Downloader)downloadChunk(client *http.Client, url string, start, end int64, file *os.File, downloaded *int64, chunkChan chan<- Chunk) error {
		maxRetry := 5
		currentStart := start

		for attempt := 0; attempt < maxRetry; attempt++ {

				req, _ := httpclient.NewDefaultWebRequest(url)
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", currentStart, end))

				resp, err := client.Do(req)
				if err != nil {
						fmt.Println("retry (request error):", err)
						time.Sleep(time.Second)
						continue
				}

				if resp.StatusCode != http.StatusPartialContent {
						resp.Body.Close()
						return fmt.Errorf("range not supported")
				}

				//buf := make([]byte, 32*1024)
				buf := bufPool.Get().([]byte)
				defer bufPool.Put(buf)
				offset := currentStart

				for {
						n, err := resp.Body.Read(buf)

						if n > 0 {

								dataCopy := make([]byte, n)
								copy(dataCopy, buf[:n])

								chunkChan<-Chunk{Offset: offset, Data: dataCopy}
								offset += int64(n)
								currentStart = offset
								atomic.AddInt64(downloaded, int64(n))
						}

						if err == io.EOF {
								resp.Body.Close()
								return nil
						}

						if err != nil {
								fmt.Println("connection error, retrying from:", currentStart, err)
								resp.Body.Close()
								time.Sleep(time.Second)
								break // keluar loop read → retry request
						}
				}
		}

		return fmt.Errorf("chunk failed after retries")
}

func (d *Downloader) downloadAudioAndVideo(target youtube.AudioAndVideo) error {
		targets := []youtube.Target{target.Audio, target.Video}

		for idx := range targets {
				t := targets[idx]

				cl := t.FileSize
				threads := d.calcThreads(cl)
				threads = d.config.Gorountines;

				f, err := os.OpenFile(t.FileName, os.O_CREATE|os.O_RDWR, 0600)
				if err != nil {
						return err
				}
				chunkChan := make(chan Chunk, 100)
				doneWriter := make(chan bool)

				go d.startGlobalDiskCache(f, chunkChan,	doneWriter)

				f.Truncate(cl)

				var wg sync.WaitGroup

				chunk := cl / int64(threads)

				var downloaded int64 = 0
				fmt.Printf("[Downloader] Downloading: %s\n", t.FileName)
				timeStart := time.Now()

				for i := 0; i < threads; i++ {
						wg.Add(1)

						start := int64(i) * chunk
						end := start + chunk - 1

						if i == threads-1 {
								end = cl - 1
						}

						go func(start, end int64) {
								defer wg.Done()
								fmt.Println("gorountine: ", i, ", start: ", start, ", end: ", end)
								d.downloadChunk(downloaderClient, t.Url, start, end, f, &downloaded, chunkChan)
						}(start, end)
				}

				go progress.ShowProgress(cl, &downloaded)

				wg.Wait()
				close(chunkChan)
				<-doneWriter
				fmt.Printf("\n[Info] Downloaded %s, size: %d, ", t.FileName, cl)
				fmt.Printf("in: %s\n", time.Since(timeStart))
				f.Close()
		}	
		fmt.Printf("[Downloader] Merging files with ffmpeg: %s %s\n", targets[0].FileName, targets[1].FileName)

		var outputFile string = d.outFile
		if outputFile == "[godl]videoplayback.mp4" {
			part := strings.Split(target.Audio.FileName, ".")
			outputFile = part[0] + ".mp4"
		}

		_,err := os.Stat(outputFile)
		if os.IsExist(err){
				outputFile += time.DateTime
		}

		cmd := exec.Command(
				"ffmpeg",
				"-i", targets[0].FileName,
				"-i", targets[1].FileName,
				"-c:v", "copy",
				"-c:a", "aac",
				outputFile,
		)

		err = cmd.Run()
		if err != nil {
				fmt.Printf("error while merging file: %s\n", err)
				return err
		}

		fmt.Printf("[Info] Removing audio & video files\n")
		err = os.Remove(target.Audio.FileName)
		if err != nil {
				fmt.Printf("error while removing file: %s, err: %s\n", target.Audio.FileName, err.Error())
		}

		err = os.Remove(target.Video.FileName)
		if err != nil {
				fmt.Printf("error while removing file: %s, err: %s\n", target.Video.FileName, err.Error())
		}

		return nil
}


func (d *Downloader)downloadYoutubeVideo(url string, config *config.Config) error {
		ytExtractor := youtube.NewYoutubeExtractor(config)
		target, err := ytExtractor.ExtractUrl(url)
		if err != nil {
				return err
		}

		switch target.(type) {
		case youtube.AudioAndVideo:
				return d.downloadAudioAndVideo(target.(youtube.AudioAndVideo))
		}

		return nil
}

func (d *Downloader)getHeaderResponse(url string) (*http.Response, error) {
		req, err := httpclient.NewDefaultWebRequest(url)
		req.Method = "HEAD"
		if err != nil {
				return nil, err
		}

		resp, err := downloaderClient.Do(req)
		if err != nil {
				return nil, err
		}

		resp.Body.Close()

		return resp, nil
}

func (d *Downloader)downloadDirectUrl(url string, config *config.Config) error {

		resp, err := d.getHeaderResponse(url)
		if err != nil {
				return fmt.Errorf("error while requesting header response: %s\n", err)
		}

		contentLength := resp.ContentLength

		_, err = os.Stat(config.OutFile)
		if os.IsExist(err){
				config.OutFile += time.DateOnly
		}

		f, err := os.OpenFile(config.OutFile, os.O_CREATE | os.O_RDWR, 0600)
		if err != nil {
				return fmt.Errorf("error while opening file: %s", err)
		}
		defer f.Close()

		chunkChan := make(chan Chunk, 100)
		doneWriter := make(chan bool)
		var downloaded int64

		var supportRanges bool

		if len(resp.Header["Accept-Ranges"]) > 0 {
				if resp.Header["Accept-Ranges"][0] == "bytes" {
						supportRanges = true
				}
		}

		if !supportRanges {
				fmt.Printf("Server does not support paralel request, downloading all file in 1 connection")
				go d.startGlobalDiskCache(f, chunkChan, doneWriter)
				go progress.ShowProgress(contentLength, &downloaded)
				timeStart := time.Now()
				err = d.downloadAll(downloaderClient, url, f, &downloaded, chunkChan)
				if err != nil {
						return err
				}

				close(chunkChan)
				<-doneWriter
				fmt.Printf("[Info] Downloaded: %s in %s\n", config.OutFile, time.Since(timeStart))


		} else {
				fmt.Printf("[Downloader] Downloading %s in %d connection\n", f.Name(), d.config.Gorountines)
				go d.startGlobalDiskCache(f, chunkChan, doneWriter)	
				var wg sync.WaitGroup
				timeStart := time.Now()
				for i := 0; i < d.config.Gorountines; i++ {
						wg.Add(1)
						chunk := contentLength / int64(d.config.Gorountines)
						start := chunk * int64(i)
						end := start + chunk - 1

						if i == d.config.Gorountines - 1 { end = contentLength }

						go func(start, end int64){
								err = d.downloadChunk(
										downloaderClient,
										url, 
										start,
										end, 
										f,
										&downloaded,
										chunkChan,
								)

								defer wg.Done()
						}(start, end)

						if err != nil {
								return err
						}
				}
				go progress.ShowProgress(contentLength, &downloaded)
				wg.Wait()
				close(chunkChan)
				<-doneWriter

				fmt.Printf("[Info] Downloaded %s in %s\n", f.Name(), time.Since(timeStart))
		}


		return nil
}


func (d *Downloader)StartDownload(url string, config *config.Config) error {
		if strings.Contains(url, "youtube") || strings.Contains(url, "youtu.be"){
				return d.downloadYoutubeVideo(url, config)
		} 

		fmt.Printf("[Downloader] Unknown url: trying to download file directly")

		return d.downloadDirectUrl(url, config)
}


func (d *Downloader)calcThreads(fileSize int64) int {
		const chunkSize = 2 * 1024 * 1024 // 2MB

		threads := int(fileSize / chunkSize)

		if threads < 1 {
				threads = 1
		}

		if threads > 16 {
				threads = 16
		}

		return threads
}
