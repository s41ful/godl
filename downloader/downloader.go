package downloader

import (
	"errors"
	"fmt"
	"godl/config"
	"godl/core"
	"godl/extractor"
	"godl/httpclient"
	"godl/logger"
	"godl/postproccessor"
	"godl/progress"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Downloader struct {
	client         *http.Client
	configs        *config.Config
	url            string
	outFile        string
	config         config.DownloaderConfig
	postprocessors []postproccessor.Postprocessor
	logger         *logger.Logger
}

func NewDownloader(cfg *config.Config) Downloader {
	return Downloader{
		client:         httpclient.NewClient(false, cfg.DownloaderCfg.MaxRetries),
		configs:        cfg,
		config:         *cfg.DownloaderCfg,
		url:            cfg.Url,
		outFile:        cfg.OutFile,
		postprocessors: postproccessor.GetALlPP(),
		logger:         cfg.Logger,
	}
}

func (d *Downloader) downloadAll(url string, file *os.File, downloaded *int64) error {

	maxRetry := 5

	for attempt := 0; attempt < maxRetry; attempt++ {

		req, _ := httpclient.NewDefaultWebRequest(url)

		resp, err := d.client.Do(req)
		if err != nil {
			d.logger.Printf(logger.LOG_LEVEL_WARN, "[WARNING] retry (request error): %s\n", err.Error())
			time.Sleep(time.Second)
			continue
		}

		buf := make([]byte, 32*1024)
		var offset int64 = 0

		for {
			n, err := resp.Body.Read(buf)

			if n > 0 {
				written, err := file.Write(buf[:n])
				if err != nil {
					d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] error while writing into file: %s\n", file.Name())
				}

				offset += int64(n)
				atomic.AddInt64(downloaded, int64(written))
			}

			if err == io.EOF {
				resp.Body.Close()
				return nil
			}

			if err != nil {
				d.logger.Printf(logger.LOG_LEVEL_WARN, "[WARNING] connection error: %s\n", err.Error())
				resp.Body.Close()
				time.Sleep(time.Second)
				break // keluar loop read → retry request
			}
		}
	}

	return errors.New("Download failed after retries")
}
func (d *Downloader) downloadChunk(url string, start, end int64, file *os.File, downloaded *int64) error {
	maxRetry := 5
	currentStart := start

	for attempt := 0; attempt < maxRetry; attempt++ {

		req, _ := httpclient.NewDefaultWebRequest(url)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", currentStart, end))

		resp, err := d.client.Do(req)
		if err != nil {
			d.logger.Printf(logger.LOG_LEVEL_WARN, "[WARNING] retry (request error): %s\n", err)
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return errors.New("range not supported")
		}

		buf := make([]byte, 32*1024)
		offset := currentStart

		for {
			n, err := resp.Body.Read(buf)

			if n > 0 {
				written, err := file.WriteAt(buf[:n], offset)
				if err != nil {
					d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] error while writing buffer into file: %s\n", err.Error())
				}

				offset += int64(n)
				currentStart = offset
				atomic.AddInt64(downloaded, int64(written))
			}

			if err == io.EOF {
				resp.Body.Close()
				return nil
			}

			if err != nil {
				d.logger.Printf(logger.LOG_LEVEL_WARN, "connection error %s, retrying from: %d", err.Error(), currentStart)
				resp.Body.Close()
				time.Sleep(time.Second)
				break // keluar loop read → retry request
			}
		}
	}
	return errors.New("chunk failed after retries")
}

func (d *Downloader) getHeaderResponse(url string) (*http.Response, error) {
	req, err := httpclient.NewDefaultWebRequest(url)
	req.Method = "HEAD"
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	return resp, nil
}

func (d *Downloader) downloadDirectUrl(url string, config *config.Config) error {
	resp, err := d.getHeaderResponse(url)
	if err != nil {
		return errors.New("error while requesting header response: " + err.Error())
	}

	contentLength := resp.ContentLength

	_, err = os.Stat(config.OutFile)
	if os.IsExist(err) {
		config.OutFile += time.DateOnly
	}

	f, err := os.OpenFile(config.OutFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return errors.New("error while opening file" + err.Error())
	}
	defer f.Close()

	var downloaded int64

	var supportRanges bool

	if len(resp.Header["Accept-Ranges"]) > 0 {
		if resp.Header["Accept-Ranges"][0] == "bytes" {
			supportRanges = true
		}
	}

	if !supportRanges {
		d.logger.Printf(logger.LOG_LEVEL_WARN, "Server does not support paralel request, downloading all file in 1 connection")
		done := make(chan bool, 1)

		go progress.ShowProgress(contentLength, &downloaded, done)
		timeStart := time.Now()

		err = d.downloadAll(url, f, &downloaded)
		if err != nil {
			return err
		}
		done <- true

		d.logger.Printf(logger.LOG_LEVEL_INFO, "[Info] Downloaded: %s in %s\n", config.OutFile, time.Since(timeStart))

	} else {
		d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] Downloading %s in %d connection\n", f.Name(), d.config.Gorountines)
		var wg sync.WaitGroup
		timeStart := time.Now()
		done := make(chan bool, 1)

		go progress.ShowProgress(contentLength, &downloaded, done)

		for i := 0; i < d.config.Gorountines; i++ {
			wg.Add(1)
			chunk := contentLength / int64(d.config.Gorountines)
			start := chunk * int64(i)
			end := start + chunk - 1

			if i == d.config.Gorountines-1 {
				end = contentLength
			}

			go func(start, end int64) {
				err = d.downloadChunk(
					url,
					start,
					end,
					f,
					&downloaded,
				)

				defer wg.Done()
			}(start, end)

			if err != nil {
				return err
			}
		}
		wg.Wait()
		done <- true

		d.logger.Printf(logger.LOG_LEVEL_INFO, "[Info] Downloaded %s in %s\n", f.Name(), time.Since(timeStart))
	}

	return nil
}

func (d *Downloader) executeParalelHttpDownload(url, filename string, size int64) error {
	cl := size
	threads := d.calcThreads(cl)
	threads = d.config.Gorountines

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}

	f.Truncate(cl)
	chunk := cl / int64(threads)

	var downloaded int64 = 0
	d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] Downloading: %s\n", filename)
	timeStart := time.Now()

	var wg sync.WaitGroup

	done := make(chan bool, 1)
	go progress.ShowProgress(cl, &downloaded, done)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		start := int64(i) * chunk
		end := start + chunk - 1

		if i == threads-1 {
			end = cl - 1
		}

		go func(start, end int64) {
			defer wg.Done()
			d.downloadChunk(url, start, end, f, &downloaded)
		}(start, end)
	}

	wg.Wait()
	done <- true
	close(done)
	defer f.Close()

	fmt.Println()
	d.logger.Printf(logger.LOG_LEVEL_INFO, "[Info] Downloaded %s, size: %d, in: %s\n", filename, cl, time.Since(timeStart))

	return nil
}

func (d *Downloader) DownloadItem(downloadItem *core.DownloadItem) error {
	for _, media := range downloadItem.Media {
		err := d.executeParalelHttpDownload(media.Format.URL, media.FileName, media.Size)
		if err != nil {
			return nil
		}
	}

	for _, pp := range d.postprocessors {
		if pp.Support(downloadItem) {
			err := pp.Process(downloadItem)
			return err
		}
	}

	return nil
}

func (d *Downloader) StartDownload(url string, config *config.Config) error {
	infoExtractor := extractor.NewInfoExtractor(config)
	downloadItem, err := infoExtractor.Start()
	d.logger.SetFlags(0)

	if err != nil {
		if err.Error() == extractor.ErrExtractorNotFound {

			d.logger.Printf(logger.LOG_LEVEL_WARN, "[WARNING] Unknown url: trying to download file directly")
			return d.downloadDirectUrl(url, config)
		}

		return err
	}
	if downloadItem.IsPlaylist {
		var totalItem int = len(*downloadItem.Entries)
		d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] Downloading playlist url, list videos: %d\n", totalItem)
		for i, item := range *downloadItem.Entries {
			d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] Downloading playlist [%d/%d]\n", i+1, totalItem)
			err := d.DownloadItem(&item)
			if err != nil {
				d.logger.Printf(logger.LOG_LEVEL_INFO, "[Downloader] Error while downloading %s, skipping download...\n", item.OutputFile)
			}
		}
	} else {
		return d.DownloadItem(downloadItem)
	}

	return nil
}

func (d *Downloader) calcThreads(fileSize int64) int {
	const chunkSize = 2 * 1024 * 1024 // 2MB

	threads := int(fileSize / chunkSize)

	if threads < 1 {
		threads = 1
	} else if threads > 16 {
		threads = 16
	}

	return threads
}
