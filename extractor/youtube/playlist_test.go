package youtube_test

import (
	// "godl/config"
	"godl/extractor/youtube"
	"testing"
)

var playlistUrls = []string{
	"https://www.youtube.com/playlist?list=PL2Fq-K0QdOQj7wkuI06Ni974PI5qiNXhL",
	"https://www.youtube.com/playlist?list=PLCinWjGoPhnuNtSHz0UfvFE26d4w0fF4h",
	"https://www.youtube.com/playlist?list=PLCinWjGoPhntSoiSRhJTjJH2WlcwMeeO_",
}

// func TestGetListVideo(t *testing.T) {
// 	yte := youtube.NewYoutubeExtractor(&config.Config{
// 		DownloaderCfg: &config.DownloaderConfig{},
// 		ExtractorConfig: &config.ExtractorConfig{},
// 	})	
//
// 	for _, el := range playlistUrls {
// 		listVideoRenderer, err := yte.GetListVideoFromPlaylist(el)
// 		if err != nil {
// 			t.Logf("error cannot extract playlist url: %s, error: %s\n", el, err.Error())
// 		} else {
// 			t.Logf("extracting playlist url success: PlaylistVideoListRenderer: %+v\n", listVideoRenderer)
// 		}
// 	}
// }
func TestGetListVideo(t *testing.T) {
	youtube.TestGetPlaylistEntryFromApi(playlistUrls[0])
}
