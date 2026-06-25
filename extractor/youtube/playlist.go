package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"godl/httpclient"
	"io"
	"net/http"
	"strings"
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

func TestGetPlaylistEntryFromApi(url string) {

		match := UrlIsPlaylist.FindStringSubmatch(url)

		Id := match[1]

		payload, _ := json.Marshal(Payload{
				Context: Context{
						Client: Client {
								ClientName: "ANDROID_VR",
								ClientVersion: "1.65.10",
								DeviceMake: "Oculus",
								DeviceModel: "Quest 3",
								AndroidSdkVersion: 32,
								UserAgent:  "com.google.android.apps.youtube.vr.oculus/1.65.10 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
								Hl: "en",
								OsName: "Android",
								OsVersion: "12L",
								TimeZone: "UTC",
								Utcoffsetminutes: 0,	
						},
				},
				BrowseId: "VL" + Id,
				RacyCheckOk: true,
				ContentCheckOk: true,
		})

			client := http.DefaultClient
			apiUrl := "https://www.youtube.com/youtubei/v1/browse"
			req, _ := http.NewRequest("POST", apiUrl, bytes.NewReader(payload))

			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			fmt.Println("BODY::===========\n", string(body))
}

func (yt *YoutubeExtractor) GetListVideoFromPlaylist(playlistUrl string) (*PlaylistVideoListRenderer, error) {
		var PlaylistRenderer PlaylistVideoListRenderer

		match := UrlIsPlaylist.FindStringSubmatch(playlistUrl)
		playlistID := match[1]
		payload := DEFAULT_PAYLOAD[strings.ToLower(DEFAULT_YT_CLIENT)]
		payload.PlaybackContext = nil
		payload.BrowseId = "VL" + playlistID

		bytePayload, err := json.Marshal(payload)
		if err != nil {
				return nil, errors.New("json marshal error: " + err.Error())
		}
		apiUrl := "https://www.youtube.com/youtubei/v1/browse"

		req, err := httpclient.NewRequest("POST", apiUrl, bytes.NewReader(bytePayload))
		if err != nil {
				return nil, errors.New("error while generating request for url: " + apiUrl + " error: " + err.Error())
		}

		resp, err := yt.client.Do(req)
		if err != nil {
				return nil, errors.New("error while requesting to playlistUrl: " + apiUrl + " error: " + err.Error())
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
				return nil, errors.New("error while reading from connection " + playlistUrl + " error: " + err.Error())
		}

		idx := bytes.Index(data, []byte(YT_PLAYLIST_VIDEO_RENDERER))
		if idx == -1 {
				return nil, errors.New("error: " + YT_PLAYLIST_VIDEO_RENDERER + " not found in JSON api")
		}

		// idx2 := bytes.Index(data[idx + len(YT_PLAYLIST_VIDEO_RENDERER):], []byte(YT_PLAYLIST_VIDEO_RENDERER))
		// if idx == -1 {
		// 		//Mungkinkah YT_PLAYLIST_VIDEO_RENDERER hanya ada 1 atau selalu 2?????, kalau hanya satu skip error ini
		// } else {
		// 		idx += idx2
		// }

		start := bytes.Index(data[idx:], []byte("{"))
		if start == -1 {
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
