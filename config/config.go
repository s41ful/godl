package config

import (
	"flag"
	"fmt"
)


type Config struct{
	Url string
	Debug bool
	Quiet bool
	OutFile string
	Threads int
	DiskCache int
}


func ParseArgs() Config {
	threads := flag.Int("threads", 4, "Total connections")
	output := flag.String("o", "[godl] videoPlayback.mp4", "Output file")
	debug := flag.Bool("debug", false, "Debug traffic")
	diskCache := flag.Int("disk-cache", 16*1024*1024, "Buffer RAM before write into disk")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: godl [options] <URL>")
		flag.PrintDefaults()
		return Config{}
	}

	return Config{
		Url:     args[0],
		Threads: *threads,
		OutFile:  *output,
		Debug: *debug,
		DiskCache: *diskCache,
	}
}

