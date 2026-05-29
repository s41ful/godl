package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"godl/httpclient"
	"io"
)

type PlaylistVideoRenderer struct {
	VideoID							string 						`json:"videoID"`
	Thumbnail           			struct {
		Thumbnails					[]Thumbnail 				`json:"thumbnails"`
	} `json:"thumbnail"`
}

type PlaylistVideoListRenderer struct {
	Contents 						[]struct {
		PlaylistVideoListRenderer	PlaylistVideoRenderer		`json:"playlistVideoRenderer"`
	}	`json:"contents"`
	PlaylistId 						string 						`json:"playlistId"`
}

func (yt *YoutubeExtractor) GetListVideoFromPlaylist(playlistUrl string) (*PlaylistVideoListRenderer, error) {
	var PlaylistRenderer PlaylistVideoListRenderer

	req, err := httpclient.NewDefaultWebRequest(playlistUrl)
	if err != nil {
		return nil, fmt.Errorf("error while generating request for url: %s, error: %s\n", playlistUrl, err.Error())
	}

	resp, err := yt.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while requesting to playlistUrl: %s, error: %s\n", playlistUrl, err.Error())
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading from connection %s, error: %s\n", playlistUrl, err.Error())
	}

	idx := bytes.Index(data, []byte(YT_PLAYLIST_VIDEO_RENDERER))
	if idx == -1 {
		return nil, fmt.Errorf("error %s not found in html\n", YT_PLAYLIST_VIDEO_RENDERER)
	}

	idx2 := bytes.Index(data[idx + len(YT_PLAYLIST_VIDEO_RENDERER):], []byte(YT_PLAYLIST_VIDEO_RENDERER))
	if idx == -1 {
		//Mungkinkah YT_PLAYLIST_VIDEO_RENDERER hanya ada 1 atau selalu 2?????, kalau hanya satu skip error ini
	} else {
		idx += idx2
	}

	start := bytes.Index(data[idx:], []byte("{"))
	if idx == -1 {
		return nil, fmt.Errorf("error { not found in start %s\n", YT_PLAYLIST_VIDEO_RENDERER)
	}

	idx += start

	jsonStr, err := extractJSON(string(data), idx)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonStr), &PlaylistRenderer)
	if err != nil {
		return nil, fmt.Errorf("error cannot Unmarshal json data into PlaylistVideoListRenderer, error: %s\n", err.Error())
	}

	return &PlaylistRenderer, nil
}
