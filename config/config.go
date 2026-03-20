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
}


func ParseArgs() Config {
	threads := flag.Int("threads", 4, "Jumlah koneksi")
	output := flag.String("o", "[godl] videoPlayback.mp4", "Output file")
	debug := flag.Bool("debug", false, "Debug traffic")

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
	}
}

