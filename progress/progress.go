package progress

import (
	"fmt"
	"strings"
	"time"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB",
		float64(b)/float64(div),
		"KMGTPE"[exp],
	)
}

func formatETA(download, total, speed int64) string {
	if speed == 0 {
		return "ETA:--"
	}
	if (total-download) / speed > 60 {
		return fmt.Sprintf(
			"ETA:%d%d%s%d%s", 0,(total - download) / speed / 60,":" ,(total - download) / speed % 60 , "s",
		)
	} else {
		return fmt.Sprintf(
			"ETA:%d%d:%d",0,0, (total - download) / speed,
		)
	}

}

func renderProgress(downloaded, total, speed int64, percent float64) {
	barWidth := 34

	filled := int(percent / 100 * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("#", filled) + strings.Repeat(" ", empty)

	fmt.Printf("\r[%s] %6.2f%% %s/%s %s/s %s",
		bar,
		percent,
		formatBytes(downloaded),
		formatBytes(total),
		formatBytes(speed),
		formatETA(downloaded, total, speed),
	)
}

func MonitorProgress(totalByte int, Progress chan int64, totalDetik *int)  {
	ticker := time.NewTicker(1 * time.Second)

	var downloaded int64
	var lastDownloaded int64
	for {
		select {
		case n := <-Progress:
			downloaded += n

		case <-ticker.C:
			*totalDetik += 1
			speed := downloaded - lastDownloaded
			lastDownloaded = downloaded
			var percent float64 = float64(downloaded) / float64(totalByte) * 100

			renderProgress(downloaded, int64(totalByte), speed, percent)
			if downloaded >= int64(totalByte) || percent >= 99.6 {
				fmt.Println("Berhasil mendownload dalam waktu :", totalDetik, " Detik")
				return 

			}
		}
	}
}
