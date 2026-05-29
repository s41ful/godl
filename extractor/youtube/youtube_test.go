package youtube_test

import (
	"fmt"
	"godl/extractor/youtube"
	"testing"
)

func isYoutubeUrl(url string) bool {
	return youtube.UrlIsYouTube.MatchString(url)
}

func isVideoUrl(url string) (bool, string) {
	match := youtube.UrlIsVideo.FindStringSubmatch(url)

	if len(match) >= 1 {
		return true, match[1]
	} else {
		return false, ""
	}
}

func isPlaylistUrl(url string) (bool, string) {
	match := youtube.UrlIsPlaylist.FindStringSubmatch(url)
	
	if len(match) >= 1 {
		return true, match[1]
	} else {
		return false, ""
	}
}

func chekUrlType(url string) {
	if isYoutubeUrl(url) {
		isPlaylistYt, playlistID := isPlaylistUrl(url)
		if isPlaylistYt {
			fmt.Printf("%s is youtube playlist url, playlistID: %s\n", url, playlistID)
			return
		}
		isVideoUrl, videoId := isVideoUrl(url)
		if isVideoUrl {
			fmt.Printf("%s is youtube video url, videoID: %s\n", url, videoId)
			return
		} 
	} else {
		fmt.Printf("%s is not a youtube url\n", url)
	}
}

func TestUrls(t *testing.T) {
	testUrls := []string{
		"https://youtube.com",
		"https://youtube.co",
		"https://youtube",
		"https://www.youtube.com/watch?v=eys5TpLWdgQ",
		"https://youtu.be/eys5TpLWdgQ?si=-Xwnd3aGlsnH9k97",
		"https://youtu.be/-6onQirbtUI?si=6g_n-CaJaGpQXYwX",
		"https://www.youtube.com/watch?v=ckpotxktdvY&list=PL6sXFl6SgjL6hxkws-EmQGXmUy8HODwY5",
	}

	for _, el := range testUrls {
		chekUrlType(el)
	}
}
