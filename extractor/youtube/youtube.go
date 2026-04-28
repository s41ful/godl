package youtube

import (
	"encoding/json"
	"fmt"
	"godl/config"
	"godl/httpclient"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Target struct {
		FileName 			string
		FileSize 			int64
		Url 					string
}

type ExtractedUrl interface{}

type AudioAndVideo struct {
	Audio 			Target
	Video 			Target
}

type VideoSingleUrl Target

type YtMetaData struct {
		PlayerResponse 			PlayerResponse
		VisitorData					string
		PlayerUrl 					string
		Cookies							[]*http.Cookie
		SignatureTimeStamp 	int
		InnertubeApiKey			string
		ApiUrl 							string
}

type Payload struct {
		Context 				Context							`json:"context"`
		VideoId 				string							`json:"videoId"`
		PlaybackContext *PlaybackContext		`json:"playbackContext,omitempty"`
		ContentCheckOk	bool 								`json:"contentCheckOk"`
		RacyCheckOk			bool								`json:"racyCheckOk"`
}

type Context struct {
		Client Client `json:"client"`
}

type Client struct {
		ClientName 					string		`json:"clientName"` 
		ClientVersion				string		`json:"clientVersion"`
		DeviceMake 					string		`json:"deviceMake,omitempty"`
		DeviceModel 				string		`json:"deviceModel,omitempty"`
		AndroidSdkVersion 	int				`json:"androidSdkVersion,omitempty"`
		UserAgent 					string		`json:"userAgent"`
		OsName 							string		`json:"osName,omitempty"`
		OsVersion 					string		`json:"osVersion,omitempty"`
		Hl 									string		`json:"hl"`
		TimeZone 						string		`json:"timeZone"`
		Utcoffsetminutes 		int				`json:"utcOffsetMinutes"`
}

type PlaybackContext struct {
		ContentPlaybackContext *ContentPlaybackContext `json:"contentPlaybackContext,omitempty"`

}

type ContentPlaybackContext struct {
		Html5Preference 		string 		`json:"html5Preference,omitempty"`
		SignatureTimeStamp	int 			`json:"signatureTimestamp,omitempty"`
}


type PlayerResponse struct {
		PlayabilityStatus struct {
				Status 					string 		`json:"status"`
				Reason 					string 		`json:"reason"`
		} `json:"playabilityStatus"`

		VideoDetails 		VideoDetails 	`json:"videoDetails"`

		StreamingData *struct {
				Formats         []Formats `json:"formats"`
				AdaptiveFormats []Formats `json:"adaptiveFormats"`
				HlsManifestUrl	string		`json:"hlsManifestUrl"`
		} `json:"streamingData"`
}

type VideoDetails struct {
		Title 							string		`json:"title"`
		Author 							string		`json:"author"`
		VideoId 						string		`json:"videoId"`
		LengthSeconds 			string		`json:"lengthSeconds"`
		IsPrivate						bool			`json:"isPrivate"`
}

type Formats struct {
		Fps 								int				`json:"fps"`
		ProjectionType 			string		`json:"projectionType"`
		ApproxDurationMs 		string		`json:"approxDurationMs"`
		AudioSampleRate			string		`json:"audioSampleRate"`
		Itag								int				`json:"itag"`
		Bitrate							int				`json:"bitrate"`
		AverageBitrate 			int				`json:"averageBitrate"`
		QualityOrdinal 			string		`json:"qualityOrdinal"`
		AudioQuality				string		`json:"audioQuality"`
		Url									string		`json:"url"`
		SignatureCipher 		string		`json:"signatureCipher"`
		LastModified				string 		`json:"lastModified"`
		Quality 						string 		`json:"quality"`
		ContentLength 			string 		`json:"contentLength"`
		QualityLabel				string 		`json:"qualityLabel"`
		AudioChannels				int 			`json:"audioChannels"`
		MimeType 						string 		`json:"mimeType"`
		Width 							int 			`json:"width"`
		Height 							int 			`json:"height"`
}

type YoutubeExtractor struct {
	configs        		*config.Config 
	config  					config.ExtractorConfig
}

const (
	DEFAULT_YT_CLIENT = "ANDROID_VR"
)

var client = httpclient.ExtracrorClient

func NewYoutubeExtractor(cfg *config.Config) *YoutubeExtractor {
	return &YoutubeExtractor{
		configs: 			cfg,
		config: 			*cfg.ExtractorConfig,	

	}
}

func (yt *YoutubeExtractor) ExtractUrl(url string) (ExtractedUrl, error) {
		client = httpclient.NewClient(yt.config.PrintTraffic)
		webPageMetadata, err := yt.ExtractWebPage(url)
		if err != nil {
				return nil, fmt.Errorf("error: error while extracting web page, %s", err)
		}


		respApi, err := yt.CallApi(&webPageMetadata, DEFAULT_YT_CLIENT)
		if err != nil {
				fmt.Printf("error: %s\n", err.Error())

				return nil,fmt.Errorf("error: error while calling api, %s", err)
		}

		if respApi.PlayabilityStatus.Status != "OK" {
				fmt.Printf("[Error] %s", respApi.PlayabilityStatus.Reason )
				return nil, fmt.Errorf("error: error api response != OK")
		}

		if respApi.StreamingData == nil {
				fmt.Println("streamingData nil")
				return nil, fmt.Errorf("error: could not get streamingData")
		}

		bestAudio := pickBestAudio(respApi.StreamingData.AdaptiveFormats)
		audioFileName := respApi.VideoDetails.Title + ".f" + strconv.Itoa(bestAudio.Itag) + ".mp4a" 
		audioSize, err := strconv.ParseInt(bestAudio.ContentLength, 10, 64)
		if err != nil {
				return nil , err
		}

		bestVideo := pickBestVideo(respApi.StreamingData.AdaptiveFormats)
		videoFileName := respApi.VideoDetails.Title + ".f" + strconv.Itoa(bestVideo.Itag) + ".mp4"
		videoSize, err := strconv.ParseInt(bestVideo.ContentLength, 10, 64)
		if err != nil {
				return nil, err
		}


		fmt.Printf("[Extractor] getting downloading format %d+%d\n",bestAudio.Itag, bestVideo.Itag)

		return AudioAndVideo{
			Audio: Target{
				FileName: 		audioFileName,
				FileSize:			audioSize,
				Url: 					bestAudio.Url,	
			},
			Video: Target{
				FileName: 		videoFileName,
				FileSize: 		videoSize,
				Url: 					bestAudio.Url,	
			},

		}, nil
}

func (yt *YoutubeExtractor) ExtractWebPage(url string) (YtMetaData, error) {
		target := "ytInitialPlayerResponse"
		//req, err := http.NewRequest("GET", url, nil)
		req, err := httpclient.NewDefaultWebRequest(url)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}
		fmt.Printf("[Extractor] Downloading web page\n")

		resp, err := client.Do(req)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}


		data, err := io.ReadAll(resp.Body)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}
		cookies := resp.Cookies()

		defer resp.Body.Close()


		html := string(data)

		idx := strings.Index(html, target)
		start := strings.Index(html[idx:], "{")
		idx += start

		jsonStr, err := extractJSON(html, idx)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}
		//var streamingData map[string]any
		jsonStr = strings.ReplaceAll(jsonStr, "\r", "")
		jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
		ytPlayer := PlayerResponse{}

		err = json.Unmarshal([]byte(jsonStr), &ytPlayer)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}

		VisitorData := getVisitorData(html)
		PlayerUrl := "https://www.youtube.com" + getPlayerUrl(html)

		req, err = httpclient.NewDefaultWebRequest(PlayerUrl)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}

		resp,err = client.Do(req)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}
		data , err = io.ReadAll(resp.Body)
		if err != nil {
				fmt.Println(err)
				return YtMetaData{}, err
		}
		sts, err := strconv.Atoi(getSts(string(data)))
		if err != nil {
				return YtMetaData{}, err
		}

		apiKey := getApiKey(html)

		apiUrl := "https://www.youtube.com/youtubei/v1/player?prettyPrint=false&key=" + apiKey 
		//fmt.Printf("Sts: %s\n", sts)

		return YtMetaData{
				SignatureTimeStamp:		 sts,
				VisitorData: VisitorData,
				Cookies: cookies,
				PlayerUrl: PlayerUrl,
				PlayerResponse: ytPlayer,
				ApiUrl: apiUrl,
				InnertubeApiKey: apiKey,
		}, nil
}
