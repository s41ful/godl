package youtube

import (
	"errors"
	"regexp"
	"strings"
)

func getUrlFromVideoID(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}

func getUrlType(url string) (UrlType, error) {
	match := UrlIsPlaylist.FindStringSubmatch(url)
	if len(match) > 1 {
		return PLAYLIST_URL, nil
	}
	
	match = UrlIsVideo.FindStringSubmatch(url)
	if len(match) > 1 {
		return VIDEO_URL, nil
	}

	return 0, errors.New(ErrInvalidYoutubeUrl)
}

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

		return "", errors.New(ErrInvalidJSON)
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
				return match[1]
		}

		return ""
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

func pickBestThumbnail(thumbnails []Thumbnail) Thumbnail {
		var bthumbnail Thumbnail	

		for _, el := range thumbnails {
			if el.Height >= bthumbnail.Height {
					bthumbnail = el
			}
		}

		return bthumbnail
}
