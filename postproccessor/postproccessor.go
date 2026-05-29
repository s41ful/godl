package postproccessor

import (
	"fmt"
	"godl/core"
	"os"
	"os/exec"
	"path/filepath"
)

type Postprocessor interface {
	Support(item *core.DownloadItem) bool
	Process(item *core.DownloadItem) error
}

type FFmpegMergePP struct {}

func (pp *FFmpegMergePP) Support(downloadedItem *core.DownloadItem) bool {
	var haveAudio bool
	var haveVideo bool
	for _, item := range downloadedItem.Media {
		if item.Format.HasAudio {
			haveAudio = true
		} else if item.Format.HasVideo {
			haveVideo = true
		}
	}

	return haveAudio && haveVideo
}

func (pp *FFmpegMergePP) Process(downloadedItem *core.DownloadItem) error {
	var outputFile string = downloadedItem.OutputFile
	var err error

	if outputFile == "[godl]videoplayback.mp4" {
		outputFile = downloadedItem.Media[0].Tittle + ".mp4"
	}
	var audioPath string
	var videoPath string

	for _, item := range downloadedItem.Media {
		if item.Format.HasAudio {
			audioPath = item.FileName
		} else if item.Format.HasVideo {
			videoPath = item.FileName
		}
	}

	fmt.Printf("[Downloader] Merging files with ffmpeg: %s+%s -> %s\n", audioPath, videoPath, outputFile)

	cmd := exec.Command(
		"ffmpeg",
		"-i", audioPath,
		"-i", videoPath,
		"-c:v", "copy",
		outputFile,
	)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg: error while merging file: %s\n", err)
		return err
	}

	fmt.Printf("[Info] Removing audio & video files\n")

	for _, media := range downloadedItem.Media {
		err = os.Remove(media.FileName)
		if err != nil {
			fmt.Printf("[Info] error while removing file: %s, err: %s\n", media.FileName, err.Error())	
		}
	}

	currentDir, err := os.Getwd()
	if currentDir != downloadedItem.OutputPath {
		fmt.Printf("[Downloader] moving %s to -> %s\n", outputFile, filepath.Join(downloadedItem.OutputFile, outputFile))
		err = os.Rename(outputFile, filepath.Join(downloadedItem.OutputFile, outputFile))
		if err != nil {
			fmt.Printf("error while moving file: %s\n", err.Error())
			return err
		}
	}

	return nil
}

func GetALlPP() []Postprocessor {
	return []Postprocessor{
		&FFmpegMergePP{},
	}
}
