package main

import (
	"fmt"
	"godl/config"
	"godl/downloader"
)

func main() {
	var configs config.Config

	configs = config.ParseArgs()
	if configs.Url == "" {
		return
	}

	dl := downloader.NewDownloader(&configs)
	err := dl.StartDownload(configs.Url, &configs)
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}
}
