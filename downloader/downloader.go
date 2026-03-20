package downloader

import (
	"fmt"
	"goDownloader/config"
	"goDownloader/extractor"
	"goDownloader/httpclient"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)
//TODO : buat fungsi download paralel untuk direct url 
var Progress chan int64

type Downloader struct {
	Url string
	ParalelDownload bool
	TotalDetik int
	Threads int
	Progress chan int64
}

var tr = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	MaxConnsPerHost:     20,
	IdleConnTimeout:     30 * time.Second,
}


type Chunk struct {
	Data  []byte
	Offset int64
}

var downloaderClient = httpclient.DownloaderClient


var bufPool = sync.Pool{
    New: func() interface{} { return make([]byte, 32*1024) }, // 32KB
}

func startGlobalDiskCache(file *os.File, chunkChan <-chan Chunk, maxCacheSize int, done chan<- bool) {
    cache := make(map[int64][]byte)
    currentSize := 0

    mergedData := make([]byte, maxCacheSize)

    // Fungsi internal untuk menulis semua yang ada di RAM ke Disk
    flush := func() {
        if len(cache) == 0 {
            return
        }

        // Ambil semua offset dan urutkan agar penulisan bersifat Sequential (mirip HDD head movement)
        offsets := make([]int64, 0, len(cache))
        for o := range cache {
            offsets = append(offsets, o)
        }
        sort.Slice(offsets, func(i, j int) bool {
            return offsets[i] < offsets[j]
        })

        // Proses Merging: Gabungkan chunk yang bersebelahan sebelum WriteAt
        for i := 0; i < len(offsets); {
            startOffset := offsets[i]

	    firstChunk := cache[startOffset]
	    copy(mergedData, firstChunk)
	    totalMerged := len(firstChunk)
	    nextOffset := startOffset + int64(len(cache[startOffset]))
	    //log.Printf("startOffset : %d, nextOffset : %d\n", startOffset, nextOffset)

            j := i + 1
            for j < len(offsets) && offsets[j] == nextOffset {
		//log.Printf("offsets(j) == nextOffset : %v\n",offsets[j] == nextOffset)
		nextChunk := cache[offsets[j]]

		if totalMerged + len(nextChunk) <= maxCacheSize {
			copy(mergedData[totalMerged:], nextChunk)
			totalMerged += len(nextChunk)

			nextOffset += int64(len(cache[offsets[j]]))
			j++
		} else { break }
            }

            // Panggil syscall WriteAt SEKALI untuk blok besar hasil penggabungan
	    file.WriteAt(mergedData[:totalMerged], startOffset)
	    log.Printf("flushing %d to offset: %d\n", len(mergedData[:totalMerged]), startOffset)
            i = j 
        }

        // Reset cache setelah sukses flush
        cache = make(map[int64][]byte)
        currentSize = 0
        file.Sync() // Memaksa OS untuk commit data ke physical storage
    }

    // Loop ini akan terus berjalan selama chunkChan terbuka
    for chunk := range chunkChan {
        cache[chunk.Offset] = chunk.Data
        currentSize += len(chunk.Data)

        // Flush otomatis jika sudah mencapai batas 16MB
        if currentSize >= maxCacheSize {
            flush()
        }
    }

    // KUNCI UTAMA: Loop di atas akan berhenti saat channel ditutup (close).
    // Kita panggil flush sekali lagi untuk menulis sisa data (<16MB).
    flush()
    
    // Beritahu main goroutine bahwa writer sudah selesai tugasnya
    done <- true
}

func downloadAll(client *http.Client, url string, file *os.File, downloaded *int64, chunkChan chan<- Chunk) error {

    maxRetry := 5

    for attempt := 0; attempt < maxRetry; attempt++ {

        req, _ := httpclient.NewDefaultWebRequest(url)

        resp, err := client.Do(req)
        if err != nil {
            log.Println("retry (request error):", err)
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
                log.Println("connection error:", err)
                resp.Body.Close()
                time.Sleep(time.Second)
                break // keluar loop read → retry request
            }
        }
    }

    return fmt.Errorf("Download failed after retries")
}


func downloadChunk(client *http.Client, url string, start, end int64, file *os.File, downloaded *int64, chunkChan chan<- Chunk) error {
    maxRetry := 5
    currentStart := start

    for attempt := 0; attempt < maxRetry; attempt++ {

        req, _ := httpclient.NewDefaultWebRequest(url)
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", currentStart, end))

        resp, err := client.Do(req)
        if err != nil {
            log.Println("retry (request error):", err)
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
                log.Println("connection error, retrying from:", currentStart, err)
                resp.Body.Close()
                time.Sleep(time.Second)
                break // keluar loop read → retry request
            }
        }
    }

    return fmt.Errorf("chunk failed after retries")
}


func downloadYoutubeVideo(url string, config config.Config) error {
	target, err := extractor.ExtractUrl(url, config)
	if err != nil {
		return err
	}



	for idx := range target {
		t := target[idx]

		cl := t.FileSize
		threads := calcThreads(cl)
		threads = 4;
		if config.Threads != 4 {
			threads = config.Threads
		}
		log.Printf("Threads: %d\n", threads)

		f, err := os.OpenFile(t.FileName, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		chunkChan := make(chan Chunk, 100)
		doneWriter := make(chan bool)

		go startGlobalDiskCache(f, 
					chunkChan,
					16*1024*1024,
					doneWriter)

		defer f.Close()

		f.Truncate(cl)

		var wg sync.WaitGroup

		chunk := cl / int64(threads)

		var downloaded int64 = 0
		log.Printf("Downloading: %s\n", t.FileName)
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
				downloadChunk(downloaderClient, t.Url, start, end, f, &downloaded, chunkChan)
			}(start, end)
		}

		go showProgress(cl, &downloaded)

		wg.Wait()
		close(chunkChan)
		<-doneWriter
		log.Printf("Berhasil mendownload %s, size: %d\n", t.FileName, cl)
		log.Printf("Dalam waktu: %s\n", time.Since(timeStart))
	}	
	log.Printf("Merging files with ffmpeg: %s %s\n", target[0].FileName, target[1].FileName)

	var outputFile string = config.OutFile

	_,err = os.Stat(outputFile)
	if os.IsExist(err){
		outputFile += time.DateTime
	}

	cmd := exec.Command(
	    "ffmpeg",
	    "-i", target[0].FileName,
	    "-i", target[1].FileName,
	    "-c:v", "copy",
	    "-c:a", "aac",
	    outputFile,
	)

	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("error while merging file: %s\n", err)
	}

	cmd = exec.Command("rm", target[0].FileName, target[1].FileName)

	cmd.Stderr = os.Stderr

	err = cmd.Run()
	log.Printf("removing audio & video files\n")
	if err != nil {
		log.Printf("error while removing files: %s\n", err)
	}

	return nil
}

func downloadDirectUrl(url string, config config.Config) error {
	req, err := httpclient.NewDefaultWebRequest(url)
	if err != nil {
		return fmt.Errorf("error while creating request: %s", err)
	}

	resp, err := downloaderClient.Do(req) 
	if err != nil {
		return fmt.Errorf("error while requesting: %s\n", err)
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

	if resp.Header["Accept-Ranges"][0] == "bytes" {
		log.Printf("Server does not support paralel request, downloading all file in 1 connection")
		go startGlobalDiskCache(f, chunkChan, 16*1024*1024, doneWriter)
		go showProgress(contentLength, &downloaded)
		timeStart := time.Now()
		err = downloadAll(downloaderClient, url, f, &downloaded, chunkChan)
		if err != nil {
			return err
		}

		close(chunkChan)
		<-doneWriter
		log.Printf("Berhasil mendownload file: %s dalam waktu %s\n", config.OutFile, time.Since(timeStart))

	}

	return nil
}


func StartDownload(url string, config config.Config) error {
	if strings.Contains(url, "youtube") || strings.Contains(url, "youtu.be"){
		return downloadYoutubeVideo(url, config)
	} 
	log.Printf("Unknown url: trying to download file directly")

	return downloadDirectUrl(url, config)
}


func calcThreads(fileSize int64) int {
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


func showProgress(total int64, downloaded *int64) {
	var lastBytes int64 = 0
	var lastTime = time.Now()

	for {
		now := time.Now()
		done := atomic.LoadInt64(downloaded)

		percent := float64(done) / float64(total) * 100

		elapsed := now.Sub(lastTime).Seconds()
		delta := done - lastBytes
		speed := float64(delta) / elapsed

		lastBytes = done
		lastTime = now

		// ETA
		var eta float64 = -1
		if speed > 0 {
			eta = float64(total-done) / speed
		}

		bar := renderBar(percent, 20)

		fmt.Printf("\rETA %s [%s] %5.1f%% | %s/s | %s/%s",
		formatTime(eta),
		bar,
		percent,
		formatBytes(speed),
		formatBytes(float64(done)),
		formatBytes(float64(total)),
	)

		if done >= total {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nDone ✅")
}

func formatBytes(b float64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%.0f B", b)
	}

	div, exp := float64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", b/div, "KMGTPE"[exp])
}

func renderBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("-", width-filled)
}


func formatTime(seconds float64) string {
	if seconds < 0 || seconds > 86400 {
		return "--:--"
	}

	s := int(seconds) % 60
	m := (int(seconds) / 60) % 60
	h := int(seconds) / 3600

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}

	return fmt.Sprintf("%02d:%02d", m, s)
}
