package progress

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

func getWidth() int {
	ws := &winsize{}
	// Berikan fallback jika ioctl gagal
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if err != 0 || ws.Col == 0 {
		return 80 // Standar lebar terminal
	}
	return int(ws.Col)
}

func ShowProgress(total int64, downloaded *int64) {
	var lastBytes int64 = 0
	var lastTime = time.Now()
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	
	width := getWidth()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			width = getWidth()
		case <-ticker.C:
			now := time.Now()
			done := atomic.LoadInt64(downloaded)

			if total <= 0 { continue } // Hindari pembagian nol

			percent := float64(done) / float64(total) * 100
			if percent > 100 { percent = 100 }

			elapsed := now.Sub(lastTime).Seconds()
			delta := done - lastBytes
			speed := float64(delta) / elapsed

			lastBytes = done
			lastTime = now

			eta := -1.0
			if speed > 0 {
				eta = float64(total-done) / speed
			}

			// Ganti strings.Builder lokal agar tidak ada isu concurrency 
			// jika fungsi ini dipanggil berkali-kali
			var sb strings.Builder
			
			// Siapkan info teks terlebih dahulu untuk menghitung sisa bar
			infoText := fmt.Sprintf(" %5.1f%% | %s/s | %s/%s", 
				percent, formatBytes(speed), formatBytes(float64(done)), formatBytes(float64(total)))
			etaText := fmt.Sprintf("ETA: %s ", formatTime(eta))
			
			// Hitung sisa ruang untuk BAR
			// 3 adalah untuk karakter "[" "]" dan "\r"
			barLen := width - len(infoText) - len(etaText) - 3
			if barLen < 10 { barLen = 10 } // Minimal bar

			sb.WriteString("\r\033[K") // \r ke awal, \033[K hapus sisa baris ke kanan
			sb.WriteString(etaText)
			sb.WriteString("[")
			
			filled := int(float64(barLen) * (percent / 100))
			for j := 0; j < barLen; j++ {
				if j < filled {
					sb.WriteString("█") // Gunakan blok penuh agar lebih modern
				} else {
					sb.WriteString("-")
				}
			}
			sb.WriteString("]")
			sb.WriteString(infoText)

			fmt.Print(sb.String())

			if done >= total {
				fmt.Println("\nDownload Selesai!")
				return
			}
		}
	}
}

func formatBytes(b float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIdx := 0
	for b >= 1024 && unitIdx < len(units)-1 {
		b /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.1f %s", b, units[unitIdx])
}

func formatTime(seconds float64) string {
	if seconds < 0 || seconds > 86400 { return "--:--" }
	d := time.Duration(seconds) * time.Second
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 { return fmt.Sprintf("%02d:%02d:%02d", h, m, s) }
	return fmt.Sprintf("%02d:%02d", m, s)
}

