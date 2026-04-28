package config

import (
		"flag"
		"fmt"
		"strconv"
)

const (
		MB = 1024 * 1025
		DEFAULT_DISK_CACHE = 16 * MB
)

type DownloaderConfig struct {
		MaxRetries 					int
		Gorountines					int
		DiskCache 					int64
}

type ExtractorConfig struct {
		MaxRetries 					int
		PrintTraffic				bool
		EmbedSubtitles			bool
		EmbedThumbnail			bool
}

type Config struct {
		DownloaderCfg  				*DownloaderConfig
		ExtractorConfig				*ExtractorConfig
		Url 									string
		Debug 								bool
		Quiet									bool
		OutFile 							string
}

func parseStrDiskCache(diskCache string) int64 {
		if diskCache[len(diskCache) - 2 :] == "MB" {
				dc, err := strconv.ParseInt(diskCache[ : len(diskCache) - 2], 10, 64)
				if err != nil {
						fmt.Printf("error: cannot parse %s into integer, %s\n", diskCache[: len(diskCache) - 2], err.Error())
						return DEFAULT_DISK_CACHE
				}

				return dc * MB
		}

		return DEFAULT_DISK_CACHE
}

func ParseArgs() Config {
		var cfg = Config {
				DownloaderCfg: &DownloaderConfig{},
				ExtractorConfig: &ExtractorConfig{},
		}

		//General Config
		flag.StringVar(&cfg.OutFile, "o", "[godl]videoplayback.mp4", "Name output file")

		//Downloader Config
		flag.IntVar(&cfg.DownloaderCfg.Gorountines, "N", 4, "Total Gorountines to download url (server suppport range request)")
		flag.IntVar(&cfg.DownloaderCfg.MaxRetries, "R", 5, "Total retries downloader to retry if there is an error")
		diskCache := flag.String("disk-cache", "16MB", "Buffer download in RAM before wrtiting into disk (default 16MB)")

		//Extractor Config
		flag.IntVar(&cfg.ExtractorConfig.MaxRetries, "extractor-retries", 3, "Total retries extractor do if connnecton error")
		flag.BoolVar(&cfg.ExtractorConfig.PrintTraffic, "print-traffic", false, "Print all traffic extractor use")
		flag.BoolVar(&cfg.ExtractorConfig.EmbedThumbnail, "embed-thumbnail", false, "Embed Thumbnail into video")
		flag.BoolVar(&cfg.ExtractorConfig.EmbedSubtitles, "embed-subs", false, "Write soft subtitles into file")

		flag.Parse()

		cfg.DownloaderCfg.DiskCache = parseStrDiskCache(*diskCache)

		args := flag.Args()
		if len(args) < 1 {
				fmt.Println("Usage: godl [options] <URL>")
				flag.PrintDefaults()
				return Config{}
		}

		cfg.Url = args[0]
		return cfg
}

