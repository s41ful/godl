package main

import (
	"fmt"
	"godl/config"
	"godl/downloader"
	"log"
	"os"
)

/*
Config :
	-v :  Verbose
	-o <OutFile>> : outputFile
	--print-traffic :  Print  traffic
	--debug :    Debug = truee


*/


func main(){
	log.Printf("args: %v\n", os.Args)

	var configs config.Config

	configs = config.ParseArgs()
	if configs.Url == "" {
		return
	}

	fmt.Printf("configs: %+v\n", configs)
	fmt.Printf("downloader configs: %v\n", configs.DownloaderCfg)
	fmt.Printf("extractor configs: %v\n", configs.ExtractorConfig)

	dl := downloader.NewDownloader(&configs)
	err := dl.StartDownload(configs.Url, &configs)
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}
}
