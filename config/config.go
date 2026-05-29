package config

import (
	"flag"
	"fmt"
	"os"
)

type DownloaderConfig struct {
		MaxRetries 						int
		Gorountines						int
}

type ExtractorConfig struct {
		MaxRetries 						int
		PrintTraffic					bool
		EmbedSubtitles				bool
		EmbedThumbnail				bool
}

type Config struct {
		DownloaderCfg  				*DownloaderConfig
		ExtractorConfig				*ExtractorConfig
		Url 									string
		Debug 								bool
		Quiet									bool
		OutFile 							string
		Directory							string
}

func ParseArgs() Config {
		var cfg = Config {
				DownloaderCfg: &DownloaderConfig{},
				ExtractorConfig: &ExtractorConfig{},
		}

		currentDir, _ := os.Getwd()

		//General Config
		flag.StringVar(&cfg.OutFile, "o", "[godl]videoplayback.mp4", "Name output file")
		flag.StringVar(&cfg.Directory, "d", currentDir, "Specified the directory for downloaded file")

		//Downloader Config
		flag.IntVar(&cfg.DownloaderCfg.Gorountines, "N", 4, "Total Gorountines to download url (if server suppport range request)")
		flag.IntVar(&cfg.DownloaderCfg.MaxRetries, "R", 5, "Total retries downloader to retry if there is an error")

		//Extractor Config
		flag.IntVar(&cfg.ExtractorConfig.MaxRetries, "extractor-retries", 3, "Total retries extractor do if connnecton error")
		flag.BoolVar(&cfg.ExtractorConfig.PrintTraffic, "print-traffic", false, "Print all traffic extractor use")
		flag.BoolVar(&cfg.ExtractorConfig.EmbedThumbnail, "embed-thumbnail", false, "Embed Thumbnail into video")
		flag.BoolVar(&cfg.ExtractorConfig.EmbedSubtitles, "embed-subs", false, "Write soft subtitles into file")

		flag.Parse()


		args := flag.Args()
		if len(args) < 1 {
				fmt.Println("Usage: godl [options] <URL>")
				flag.PrintDefaults()
				return Config{}
		}

		cfg.Url = args[0]
		return cfg
}

