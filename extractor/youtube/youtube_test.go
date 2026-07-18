package youtube_test

import (
	"testing"

	"godl/extractor/youtube"
)


func TestUrlsType(t *testing.T) {
		yte := youtube.YoutubeExtractor{}
		testUrls := map[string]youtube.UrlType {
				"https://youtube.com": youtube.UNKNOWN_MEDIA_URL,
				"https://youtu.be/eys5TpLWdgQ?si=-Xwnd3aGlsnH9k97": youtube.VIDEO_URL,
				"https://youtu.be/-6onQirbtUI?si=6g_n-CaJaGpQXYwX": youtube.VIDEO_URL,
				"https://www.youtube.com/watch?v=ckpotxktdvY&list=PL6sXFl6SgjL6hxkws-EmQGXmUy8HODwY5": youtube.PLAYLIST_URL,
		}

		for k, v := range testUrls {
				urlType, _ := yte.GetUrlType(k)
				if !(urlType == v) {
						t.Fatal("error url type does not same as the expected, url: ", k)
				}
		}
}
