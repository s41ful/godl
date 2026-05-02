package main

import (
	"fmt"
	"godl/config"
	"godl/downloader"
	"os"
)

func main(){
	fmt.Printf("args: %v\n", os.Args)

	var configs config.Config

	configs = config.ParseArgs()
	if configs.Url == "" {
		return
	}

	fmt.Printf("[Info] configs: %+v, ", configs)
	fmt.Printf("downloader configs: %+v, ", configs.DownloaderCfg)
	fmt.Printf("extractor configs: %v\n", configs.ExtractorConfig)

	dl := downloader.NewDownloader(&configs)
	err := dl.StartDownload(configs.Url, &configs)
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}
}
