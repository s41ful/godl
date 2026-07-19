package youtube

import (
	"errors"
	"godl/core"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (yt *YoutubeExtractor) getUrlFromVideoID(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}

func (yt *YoutubeExtractor) GetUrlType(url string) (UrlType, error) {
	match := UrlIsPlaylist.FindStringSubmatch(url)
	if len(match) > 1 {
		return PLAYLIST_URL, nil
	}
	
	match = UrlIsVideo.FindStringSubmatch(url)
	if len(match) > 1 {
		return VIDEO_URL, nil
	}

	return UNKNOWN_MEDIA_URL, errors.New(ErrInvalidYoutubeUrl)
}

func (yt *YoutubeExtractor) extractJSON(s string, start int) (string, error) {
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

func (yt *YoutubeExtractor) getVisitorData(html *string) string {
		re := regexp.MustCompile(`"VISITOR_DATA":"([^"]+)"`)
		match := re.FindStringSubmatch(*html)
		if len(match) > 1 {
				return match[1]
		}

		return "";
}

func (yt *YoutubeExtractor) getPlayerUrl(html *string) string {
		re := regexp.MustCompile(`"jsUrl":"([^"]+)"`)
		match := re.FindStringSubmatch(*html)
		if len(match) > 1 {
				return match[1];
		}

		return "";
}

func (yt *YoutubeExtractor) getSts(baseJs *string) string {
		re := regexp.MustCompile(`signatureTimestamp:(\d+)|sts:(\d+)`)
		match := re.FindStringSubmatch(*baseJs)
		if len(match) > 1 {
				return match[1];
		}

		return "";

}

func (yt *YoutubeExtractor) getApiKey(html *string) string {
		re := regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
		match := re.FindStringSubmatch(*html)

		if len(match) > 1 {
				return match[1]
		}

		return ""
}

func (yt *YoutubeExtractor) pickSingleFormats(playerResponse *PlayerResponse) (*core.DownloadItem, error) {
	if len(playerResponse.StreamingData.Formats) == 0 {
		return nil, errors.New("playerResponse streamingData does not contains any url or formats")
	}

	contentLength, err := strconv.Atoi(playerResponse.StreamingData.Formats[0].ContentLength)

	var mediaInfo = []core.MediaInfo{
		{ ID:       playerResponse.VideoDetails.VideoId,
			Tittle:   playerResponse.VideoDetails.Title,
			FileName: playerResponse.VideoDetails.Title + ".mp4",
			Size:     int64(contentLength), 
			Format: core.Format{
				URL:      playerResponse.StreamingData.Formats[0].Url,
				Type:     "Video+Audio",
				HasAudio: true,
			},
		},

	}

	downloadItem := core.DownloadItem{
		IsPlaylist: false,
		OutputFile: playerResponse.VideoDetails.Title + ".mp4",
		OutputPath: yt.configs.Directory,

		Entries: nil,
		Media: mediaInfo,
	} 

	return &downloadItem, err
}

func (yt *YoutubeExtractor) pickBestAudio(formats []Formats) *Formats {
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

func (yt *YoutubeExtractor) pickBestVideo(formats []Formats) *Formats {
		var best *Formats

		for i := range formats {
				f := &formats[i]

				if !strings.Contains(f.MimeType, "video/mp4") ||
				!strings.Contains(f.MimeType, "avc1") {
						continue
				}

				if f.Height <= 1080 {
						if best == nil || f.Height > best.Height {
								best = f
						}
				}
		}

		return best
}

func (yt *YoutubeExtractor) pickBestThumbnail(thumbnails []Thumbnail) Thumbnail {
		var bthumbnail Thumbnail	

		for _, el := range thumbnails {
			if el.Height >= bthumbnail.Height {
					bthumbnail = el
			}
		}

		return bthumbnail
}

func (yt *YoutubeExtractor) checkFFmpeg() bool {
		cmd := exec.Command("ffmpeg", "-h")
		err := cmd.Run()

		return err == nil
}
