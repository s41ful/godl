package youtube

import (
	"encoding/json"
	"fmt"
	"godl/config"
	"godl/core"
	"godl/httpclient"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	DEFAULT_YT_CLIENT 				= 		"ANDROID_VR"
	YT_INITIAL_PLAYER_RESPONSE 		=		"ytInitialPlayerResponse"
	YT_PLAYLIST_VIDEO_RENDERER 		=		"playlistVideoRenderer"
	VIDEO_URL						= 		0
	PLAYLIST_URL					=       1
)

var (
	UrlIsYouTube 	= 	regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?(?:youtube\.com|youtu\.be)\b`)
	UrlIsVideo   	= 	regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?(?:youtube\.com|youtu\.be)\/(?:watch\?v=|embed\/|v\/|shorts\/)?([a-zA-Z0-9_-]{11})`)
	UrlIsPlaylist 	= 	regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?youtube\.com\/.*[?&]list=([a-zA-Z0-9_-]+)`)
)

type TargetTask interface {
	GetItems()     			[]DownloadItem
	OnComplete()			error
}

type DownloadItem struct {
	FileName 				string
	FileSize 				int64
	Url 					string
}

type AudioAndVideo struct {
	outDirectori			string
	outputFile  			string
	Audio 					DownloadItem
	Video 					DownloadItem
}

func (av *AudioAndVideo) GetItems() []DownloadItem {
	return []DownloadItem {av.Audio, av.Video}
}

func (av *AudioAndVideo) OnComplete() error {
	var outputFile string = av.outputFile
	var err error

	if outputFile == "[godl]videoplayback.mp4" {
		part := strings.Split(av.Audio.FileName, ".")
		outputFile = part[0] + ".mp4"
	}

	fmt.Printf("[Downloader] Merging files with ffmpeg: %s+%s -> %s\n", av.Audio.FileName, av.Video.FileName, outputFile)

	cmd := exec.Command(
		"ffmpeg",
		"-i", av.Audio.FileName,
		"-i", av.Video.FileName,
		"-c:v", "copy",
		outputFile,
	)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg: error while merging file: %s\n", err)
		return err
	}

	fmt.Printf("[Info] Removing audio & video files\n")
	items := av.GetItems()
	for _, t := range items {
		err = os.Remove(t.FileName)
		if err != nil {
			fmt.Printf("[Info] error while removing file: %s, err: %s\n", t.FileName, err.Error())	
		}
	}

	currentDir, err := os.Getwd()
	if currentDir != av.outDirectori {
		fmt.Printf("[Downloader] moving %s to -> %s\n", outputFile, filepath.Join(av.outDirectori, outputFile))
		err = os.Rename(outputFile, filepath.Join(av.outDirectori, outputFile))
		if err != nil {
			fmt.Printf("error while moving file: %s\n", err.Error())
			return err
		}
	}

	return nil
}

type VideoSingleUrl DownloadItem

type UrlType int

func (v *VideoSingleUrl) GetItems() []DownloadItem {
	return []DownloadItem {
		{
			FileName: 	v.FileName,
			FileSize: 	v.FileSize,
			Url:	  	v.Url,
		},
	}
}

func (v *VideoSingleUrl) OnComplete() error {
	return nil
}

type YtMetaData struct {
	PlayerResponse 					PlayerResponse
	VisitorData						string
	PlayerUrl 						string
	Cookies							[]*http.Cookie
	SignatureTimeStamp 				int
	InnertubeApiKey					string
	ApiUrl 							string
}

type Payload struct {
	Context 						Context					`json:"context"`
	VideoId 						string					`json:"videoId"`
	PlaybackContext 				*PlaybackContext		`json:"playbackContext,omitempty"`
	ContentCheckOk					bool 					`json:"contentCheckOk"`
	RacyCheckOk						bool					`json:"racyCheckOk"`
}

type Context struct {
	Client 							Client 					`json:"client"`
}

type Client struct {
	ClientName 						string					`json:"clientName"` 
	ClientVersion					string					`json:"clientVersion"`
	DeviceMake 						string					`json:"deviceMake,omitempty"`
	DeviceModel 					string					`json:"deviceModel,omitempty"`
	AndroidSdkVersion 				int						`json:"androidSdkVersion,omitempty"`
	UserAgent 						string					`json:"userAgent"`
	OsName 							string					`json:"osName,omitempty"`
	OsVersion 						string					`json:"osVersion,omitempty"`
	Hl 								string					`json:"hl"`
	TimeZone 						string					`json:"timeZone"`
	Utcoffsetminutes 				int						`json:"utcOffsetMinutes"`
}

type PlaybackContext struct {
	ContentPlaybackContext 			*ContentPlaybackContext `json:"contentPlaybackContext,omitempty"`

}

type ContentPlaybackContext struct {
	Html5Preference 				string 					`json:"html5Preference,omitempty"`
	SignatureTimeStamp				int 					`json:"signatureTimestamp,omitempty"`
}


type PlayerResponse struct {
	PlayabilityStatus struct {
		Status 						string 					`json:"status"`
		Reason 						string 					`json:"reason"`
	} `json:"playabilityStatus"`

	VideoDetails 					VideoDetails 			`json:"videoDetails"`

	StreamingData *struct {
		Formats         			[]Formats 				`json:"formats"`
		AdaptiveFormats 			[]Formats 				`json:"adaptiveFormats"`
		HlsManifestUrl				string					`json:"hlsManifestUrl"`
	} `json:"streamingData"`
}

type VideoDetails struct {
	Title 							string					`json:"title"`
	Author 							string					`json:"author"`
	VideoId 						string					`json:"videoId"`
	LengthSeconds 					string					`json:"lengthSeconds"`
	IsPrivate						bool					`json:"isPrivate"`
	Thumbnail           			struct {
		Thumbnails      		[]Thumbnail  				`json:"thumbnails"`
	} `json:"thumbnail"`
}

type Formats struct {
	Fps 							int						`json:"fps"`
	ProjectionType 					string					`json:"projectionType"`
	ApproxDurationMs 				string					`json:"approxDurationMs"`
	AudioSampleRate					string					`json:"audioSampleRate"`
	Itag							int						`json:"itag"`
	Bitrate							int						`json:"bitrate"`
	AverageBitrate 					int						`json:"averageBitrate"`
	QualityOrdinal 					string					`json:"qualityOrdinal"`
	AudioQuality					string					`json:"audioQuality"`
	Url								string					`json:"url"`
	SignatureCipher 				string					`json:"signatureCipher"`
	LastModified					string 					`json:"lastModified"`
	Quality 						string 					`json:"quality"`
	ContentLength 					string 					`json:"contentLength"`
	QualityLabel					string 					`json:"qualityLabel"`
	AudioChannels					int 					`json:"audioChannels"`
	MimeType 						string 					`json:"mimeType"`
	Width 							int 					`json:"width"`
	Height 							int 					`json:"height"`
}

type Thumbnail struct {
	Url 							string 					`json:"url"`
	Width 							int 					`json:"width"`
	Height 							int 					`json:"height"`
}

type YoutubeExtractor struct {
	client 							*http.Client
	configs        					*config.Config 
	config  						config.ExtractorConfig
}

func NewYoutubeExtractor(config *config.Config) *YoutubeExtractor {
	return &YoutubeExtractor{
		client: 				httpclient.NewClient(config.Debug, config.DownloaderCfg.MaxRetries),
		config: 				*config.ExtractorConfig,
		configs: 				config,
	}
}

func (yt *YoutubeExtractor) InitConfig(cfg *config.Config) {
	yt.client 				= 			httpclient.NewClient(cfg.Debug, cfg.ExtractorConfig.MaxRetries)
	yt.config 				= 			*cfg.ExtractorConfig
	yt.configs 				= 			cfg
}

func (yt *YoutubeExtractor) Extract(url string) (*core.DownloadItem, error) {
	urlType, err := getUrlType(url)
	if err != nil {
		return nil, err
	}

	switch urlType {
	case PLAYLIST_URL:
		fmt.Printf("[Youtube] Extracting PlaylistUrl\n")
		return yt.ExtractPlaylist(url)
	case VIDEO_URL:
		fmt.Printf("[Youtube] Extracting VideoUrl\n")
		return yt.ExtractVideoUrl(url)
	}

	fmt.Printf("Unknown Youtube Url")
	return nil, err
}

func (yt *YoutubeExtractor) Match(url string) bool {
	return UrlIsYouTube.MatchString(url)
}

func (yt *YoutubeExtractor) ExtractPlaylist(url string) (*core.DownloadItem, error) {
	playlist, err := yt.GetListVideoFromPlaylist(url)
	if err != nil {
		return nil, err
	}

	var itemList = []core.DownloadItem{}

	for i, item := range playlist.Contents {
		url := getUrlFromVideoID(item.PlaylistVideoListRenderer.VideoID)

		item, err := yt.ExtractVideoUrl(url)
		if err != nil {
			fmt.Printf("error while extracting url: %s, %s, skipping item: %d\n", url, err.Error(), i + 1)
			continue
		}

		itemList = append(itemList, *item)
	}
	return &core.DownloadItem{
		IsPlaylist: true,

		Entries: &itemList,
	}, nil
}

func (yt *YoutubeExtractor) ExtractVideoUrl(url string) (*core.DownloadItem, error) {
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
		return nil, fmt.Errorf("error: could not get streamingData") }
	if strings.Contains(respApi.VideoDetails.Title, "/") {
		respApi.VideoDetails.Title = strings.ReplaceAll(respApi.VideoDetails.Title, "/", "-")
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

	fmt.Printf("[Extractor] getting format %d+%d\n",bestAudio.Itag, bestVideo.Itag)
	fmt.Printf("getting best video+audio: %+v\n%+v\n", bestVideo, bestAudio)

	var mediaInfo = []core.MediaInfo{
		{
			ID: strconv.Itoa(bestAudio.Itag),
			Tittle: respApi.VideoDetails.Title,
			FileName: audioFileName,
			Size: audioSize,

			Format: core.Format{
				URL: bestAudio.Url,
				Type: "Audio",
				HasAudio: true,
			},
		},
		{
			ID: strconv.Itoa(bestVideo.Itag),
			FileName: videoFileName,
			Size: videoSize,

			Format: core.Format{
				URL: bestVideo.Url,
				Type: "Video",
				HasVideo: true,
			},
		},
	}

	var outputFile string 
	if outputFile == "[godl]videoplayback.mp4" {
		outputFile = respApi.VideoDetails.Title + ".mp4"
	}
	var downloadItem = core.DownloadItem{
		IsPlaylist: false,
		OutputFile: respApi.VideoDetails.Title + ".mp4",
		OutputPath: yt.configs.Directory,
		
		Entries: nil,
		Media: mediaInfo,
	}

	return &downloadItem, nil
}

func (yt *YoutubeExtractor) ExtractWebPage(url string) (YtMetaData, error) {
	req, err := httpclient.NewDefaultWebRequest(url)
	if err != nil {
		fmt.Println(err)
		return YtMetaData{}, err
	}
	fmt.Printf("[Extractor] Downloading web page\n")

	resp, err := yt.client.Do(req)
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

	//What if idx == -1??
	idx := strings.Index(html, YT_INITIAL_PLAYER_RESPONSE)
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

	resp,err = yt.client.Do(req)
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

	return YtMetaData{
		SignatureTimeStamp:		sts,
		VisitorData: 			VisitorData,
		Cookies: 				cookies,
		PlayerUrl: 				PlayerUrl,
		PlayerResponse: 		ytPlayer,
		ApiUrl: 				apiUrl,
		InnertubeApiKey: 		apiKey,
	}, nil
}
