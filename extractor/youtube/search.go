package youtube

import (
	"strings"
	"fmt"
	"regexp"
	"log"
)

func extractJSON(s string, start int) (string, error) {
		var count int
		var inString bool
		var escape bool

		for i := start; i < len(s); i++ {
				c := s[i]

				if escape {
						escape = false
						continue
				}

				if c == '\\' {
						escape = true
						continue
				}

				if c == '"' {
						inString = !inString
						continue
				}

				if !inString {
						if c == '\r' || c == '\n' {
								fmt.Println("Index :", i)
						}
						if c == '{' {
								count++
						} else if c == '}' {
								count--
								if count == 0 {
										return s[start : i+1], nil
								}
						}
				}
		}

		return "", fmt.Errorf("error: invalid JSON")
}

func getVisitorData(html string) string {
		re := regexp.MustCompile(`"VISITOR_DATA":"([^"]+)"`)
		match := re.FindStringSubmatch(html)
		if len(match) > 1 {
				return match[1]
		}

		return "";
}

func getPlayerUrl(html string) string {
		re := regexp.MustCompile(`"jsUrl":"([^"]+)"`)
		match := re.FindStringSubmatch(html)
		if len(match) > 1 {
				//log.Printf("playerUrl found: %s\n", match[1])
				return match[1];
		}

		return "";
}

func getSts(baseJs string) string {
		re := regexp.MustCompile(`signatureTimestamp:(\d+)|sts:(\d+)`)
		match := re.FindStringSubmatch(baseJs)
		if len(match) > 1 {
				//log.Printf("signatureTimestamp found: %s\n", match[1])
				return match[1];
		}

		return "";

}

func getApiKey(html string) string {
		re := regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
		match := re.FindStringSubmatch(html)

		if len(match) > 1 {
				log.Printf("Api key found\n")
				return match[1]
		}

		return match[0]
}

func pickBestAudio(formats []Formats) *Formats {
		var best *Formats

		for i := range formats {
				f := &formats[i]

				if !strings.Contains(f.MimeType, "audio/mp4") ||
				!strings.Contains(f.MimeType, "mp4a") {
						continue
				}

				if best == nil || f.Bitrate > best.Bitrate {
						best = f
				}
		}

		return best
}

func pickBestVideo(formats []Formats) *Formats {
		var best *Formats

		for i := range formats {
				f := &formats[i]

				if !strings.Contains(f.MimeType, "video/mp4") ||
				!strings.Contains(f.MimeType, "avc1") {
						continue
				}

				// target 1080p atau di bawahnya
				if f.Height <= 1080 {
						if best == nil || f.Height > best.Height {
								best = f
						}
				}
		}

		return best
}
