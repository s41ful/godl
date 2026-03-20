package main

import (
	"goDownloader/config"
	"goDownloader/downloader"
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

	log.Printf("configs: %+v\n", configs)
	
	downloader.StartDownload(configs.Url, configs)
}
