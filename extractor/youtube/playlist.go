package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"godl/httpclient"
	"godl/logger"
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
		return nil, errors.New("error while generating request for url: " + playlistUrl + "error: " + err.Error())
	}

	resp, err := yt.client.Do(req)
	if err != nil {
		return nil, errors.New("error while requesting to playlistUrl: " + playlistUrl + "error: " + err.Error())
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("error while reading from connection " + playlistUrl + "error: " + err.Error())
	}

	idx := bytes.Index(data, []byte(YT_PLAYLIST_VIDEO_RENDERER))
	if idx == -1 {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, string(data))
		return nil, errors.New("error: " + YT_PLAYLIST_VIDEO_RENDERER + " not found in html")
	}

	idx2 := bytes.Index(data[idx + len(YT_PLAYLIST_VIDEO_RENDERER):], []byte(YT_PLAYLIST_VIDEO_RENDERER))
	if idx == -1 {
		//Mungkinkah YT_PLAYLIST_VIDEO_RENDERER hanya ada 1 atau selalu 2?????, kalau hanya satu skip error ini
	} else {
		idx += idx2
	}

	start := bytes.Index(data[idx:], []byte("{"))
	if idx == -1 {
		return nil, errors.New("error { not found in start " + YT_PLAYLIST_VIDEO_RENDERER)
	}

	idx += start

	jsonStr, err := extractJSON(string(data), idx)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonStr), &PlaylistRenderer)
	if err != nil {
		return nil, errors.New("error cannot Unmarshal json data into PlaylistVideoListRenderer, error: " + err.Error())
	}

	return &PlaylistRenderer, nil
}
