package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"godl/config"
	"godl/core"
	"godl/httpclient"
	"godl/logger"
)

const (
	DEFAULT_YT_CLIENT          = "ANDROID_VR"
	YT_INITIAL_PLAYER_RESPONSE = "ytInitialPlayerResponse"
	YT_PLAYLIST_VIDEO_RENDERER = "playlistVideoListRenderer"
)

const (		
	UNKNOWN_MEDIA_URL = iota
	VIDEO_URL 
	PLAYLIST_URL
)

var (
	UrlIsYouTube  = regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?(?:youtube\.com|youtu\.be)\b`)
	UrlIsVideo    = regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?(?:youtube\.com|youtu\.be)\/(?:watch\?v=|embed\/|v\/|shorts\/)?([a-zA-Z0-9_-]{11})`)
	UrlIsPlaylist = regexp.MustCompile(`^(?:https?:\/\/)?(?:www\.)?(?:m\.)?youtube\.com\/.*[?&]list=([a-zA-Z0-9_-]+)`)
)

type DownloadItem struct {
	FileName string
	FileSize int64
	Url      string
}
type UrlType int

type YtMetaData struct {
	PlayerResponse     PlayerResponse
	VisitorData        string
	PlayerUrl          string
	Cookies            []*http.Cookie
	SignatureTimeStamp int
	InnertubeApiKey    string
	ApiUrl             string
}

type Payload struct {
	Context         Context          `json:"context"`
	VideoId         string           `json:"videoId,omitempty"`
	BrowseId				string					 `json:"browseId,omitempty"`
	PlaybackContext *PlaybackContext `json:"playbackContext,omitempty"`
	Continuation    string					 `json:"continuation,omitempty"`
	ContentCheckOk  bool             `json:"contentCheckOk,omitempty"`
	RacyCheckOk     bool             `json:"racyCheckOk,omitempty"`
	Params					string   				 `json:"params,omitempty"`
}

type Context struct {
	Client Client `json:"client"`
}

type Client struct {
	ClientName        string `json:"clientName"`
	ClientVersion     string `json:"clientVersion"`
	DeviceMake        string `json:"deviceMake,omitempty"`
	DeviceModel       string `json:"deviceModel,omitempty"`
	AndroidSdkVersion int    `json:"androidSdkVersion,omitempty"`
	UserAgent         string `json:"userAgent"`
	OsName            string `json:"osName,omitempty"`
	OsVersion         string `json:"osVersion,omitempty"`
	Hl                string `json:"hl"`
	TimeZone          string `json:"timeZone"`
	Utcoffsetminutes  int    `json:"utcOffsetMinutes"`
}

type PlaybackContext struct {
	ContentPlaybackContext *ContentPlaybackContext `json:"contentPlaybackContext,omitempty"`
}

type ContentPlaybackContext struct {
	Html5Preference    string `json:"html5Preference,omitempty"`
	SignatureTimeStamp int    `json:"signatureTimestamp,omitempty"`
}

type PlayerResponse struct {
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`

	VideoDetails VideoDetails `json:"videoDetails"`

	StreamingData *struct {
		Formats         []Formats `json:"formats"`
		AdaptiveFormats []Formats `json:"adaptiveFormats"`
		HlsManifestUrl  string    `json:"hlsManifestUrl"`
	} `json:"streamingData"`
}

type VideoDetails struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	VideoId       string `json:"videoId"`
	LengthSeconds string `json:"lengthSeconds"`
	IsPrivate     bool   `json:"isPrivate"`
	Thumbnail     struct {
		Thumbnails []Thumbnail `json:"thumbnails"`
	} `json:"thumbnail"`
}

type Formats struct {
	Fps              int    `json:"fps"`
	ProjectionType   string `json:"projectionType"`
	ApproxDurationMs string `json:"approxDurationMs"`
	AudioSampleRate  string `json:"audioSampleRate"`
	Itag             int    `json:"itag"`
	Bitrate          int    `json:"bitrate"`
	AverageBitrate   int    `json:"averageBitrate"`
	QualityOrdinal   string `json:"qualityOrdinal"`
	AudioQuality     string `json:"audioQuality"`
	Url              string `json:"url"`
	SignatureCipher  string `json:"signatureCipher"`
	LastModified     string `json:"lastModified"`
	Quality          string `json:"quality"`
	ContentLength    string `json:"contentLength"`
	QualityLabel     string `json:"qualityLabel"`
	AudioChannels    int    `json:"audioChannels"`
	MimeType         string `json:"mimeType"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
}

type Thumbnail struct {
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type YoutubeExtractor struct {
	client  *http.Client
	configs *config.Config
	config  config.ExtractorConfig
	logger 	*logger.Logger
	hasFFmpeg bool
}

func NewYoutubeExtractor(config *config.Config) *YoutubeExtractor {
	return &YoutubeExtractor{
		client:  httpclient.NewClient(config.Debug, config.DownloaderCfg.MaxRetries),
		config:  *config.ExtractorConfig,
		configs: config,
		logger:  config.Logger,
	}
}

func (yt *YoutubeExtractor) InitConfig(cfg *config.Config) {
	yt.client 	= httpclient.NewClient(cfg.Debug, cfg.ExtractorConfig.MaxRetries)
	yt.config 	= *cfg.ExtractorConfig
	yt.configs 	= cfg
	yt.logger 	= cfg.Logger
}

func (yt *YoutubeExtractor) Extract(url string) (*core.DownloadItem, error) {
	urlType, err := yt.GetUrlType(url)
	if err != nil {
		return nil, err
	}

	yt.hasFFmpeg = yt.checkFFmpeg()
	yt.logger.SetFlags(0)

	if !yt.hasFFmpeg {
		yt.logger.Println(logger.LOG_LEVEL_INFO, "[warning] " + WarnFFmpegNotInstalled)
	}

	switch urlType {
	case PLAYLIST_URL:
		matchID := UrlIsPlaylist.FindStringSubmatch(url)
		yt.logger.Printf(logger.LOG_LEVEL_DEBUG, "[youtube] extracting playlist: (%s)\n", matchID[1])
		return yt.ExtractPlaylist(url)
	case VIDEO_URL:
		matchID := UrlIsVideo.FindStringSubmatch(url)
		yt.logger.Printf(logger.LOG_LEVEL_DEBUG, "[youtube] extracting video: (%s)\n", matchID[1])
		return yt.ExtractVideoUrl(url)
	}

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
	var logLevelBefore = yt.logger.GetLogLevel()

	for i, item := range playlist.Contents {
		url := yt.getUrlFromVideoID(item.PlaylistVideoListRenderer.VideoID)
		yt.logger.SetLogLevel(logger.LOG_LEVEL_INFO)
		fmt.Printf("\r\033[K[youtube] extracting playlist items: [%d/%d]", i+1, len(playlist.Contents))

		yt.logger.SetLogLevel(logger.LOG_LEVEL_NONE)

		item, err := yt.ExtractVideoUrl(url)
		if err != nil {
			yt.logger.Printf(logger.LOG_LEVEL_INFO, "error while extracting url: %s, %s, skipping item: %d\n", url, err.Error(), i+1)
			continue
		}

		itemList = append(itemList, *item)
	}

	yt.logger.SetLogLevel(logLevelBefore)

	fmt.Println()

	return &core.DownloadItem{
		IsPlaylist: true,

		Entries: &itemList,
	}, nil
}

func (yt *YoutubeExtractor) ExtractVideoUrl(url string) (*core.DownloadItem, error) {
	webPageMetadata, err := yt.ExtractWebPage(url)
	if err != nil {
		return nil, errors.New("error while extracting web page, " + err.Error())
	}

	if !yt.hasFFmpeg {
		return yt.fallbackToLowest(webPageMetadata)
	}

	respApi, err := yt.CallApi(webPageMetadata, DEFAULT_YT_CLIENT)
	if err != nil {
		return nil, errors.New("error while calling api, " + err.Error())
	}

	if respApi.PlayabilityStatus.Status != "OK" {
		yt.logger.Printf(logger.LOG_LEVEL_INFO, "[error] %s\n", respApi.PlayabilityStatus.Reason)
		return nil, errors.New("api response != OK")
	}

	if respApi.StreamingData == nil {
		return nil, errors.New("could not get streamingData")
	}
	if strings.Contains(respApi.VideoDetails.Title, "/") {
		respApi.VideoDetails.Title = strings.ReplaceAll(respApi.VideoDetails.Title, "/", "-")
	}

	bestAudio := yt.pickBestAudio(respApi.StreamingData.AdaptiveFormats)
	audioFileName := respApi.VideoDetails.Title + ".f" + strconv.Itoa(bestAudio.Itag) + ".mp4a"
	audioSize, err := strconv.ParseInt(bestAudio.ContentLength, 10, 64)
	if err != nil {
		return nil, err
	}

	// TODO: add support another video and audio codecs
	bestVideo := yt.pickBestVideo(respApi.StreamingData.AdaptiveFormats)
	videoFileName := respApi.VideoDetails.Title + ".f" + strconv.Itoa(bestVideo.Itag) + ".mp4"
	videoSize, err := strconv.ParseInt(bestVideo.ContentLength, 10, 64)
	if err != nil {
		return nil, err
	}

	yt.logger.Printf(logger.LOG_LEVEL_INFO, "[youtube] getting format %d+%d\n", bestAudio.Itag, bestVideo.Itag)

	var mediaInfo = []core.MediaInfo{
		{
			ID:       strconv.Itoa(bestAudio.Itag),
			Tittle:   respApi.VideoDetails.Title,
			FileName: audioFileName,
			Size:     audioSize,

			Format: core.Format{
				URL:      bestAudio.Url,
				Type:     "Audio",
				HasAudio: true,
			},
		},
		{
			ID:       strconv.Itoa(bestVideo.Itag),
			FileName: videoFileName,
			Size:     videoSize,

			Format: core.Format{
				URL:      bestVideo.Url,
				Type:     "Video",
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
		Media:   mediaInfo,
	}

	return &downloadItem, nil
}

func (yt *YoutubeExtractor) ExtractWebPage(url string) (*YtMetaData, error) {
	req, err := httpclient.NewDefaultWebRequest(url)
	if err != nil {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, err)
		return nil, err
	}
	yt.logger.Println(logger.LOG_LEVEL_INFO, "[youtube] downloading web page")

	resp, err := yt.client.Do(req)
	if err != nil {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, err)
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, err)
		return nil, err
	}
	cookies := resp.Cookies()

	defer resp.Body.Close()

	html := string(data)

	idx := strings.Index(html, YT_INITIAL_PLAYER_RESPONSE)
	if idx == -1 {
		return nil, errors.New(ErrYtInitialPlayerResponseNotFound)
	}

	start := strings.Index(html[idx:], "{")
	idx += start

	jsonStr, err := yt.extractJSON(html, idx)
	if err != nil {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, err)
		return nil, err
	}

	//var streamingData map[string]any
	jsonStr = strings.ReplaceAll(jsonStr, "\r", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
	ytPlayer := PlayerResponse{}

	err = json.Unmarshal([]byte(jsonStr), &ytPlayer)
	if err != nil {
		yt.logger.Println(logger.LOG_LEVEL_DEBUG, "error while marshalling jsonStr")
		return nil, err
	}

	VisitorData := yt.getVisitorData(&html)
	if VisitorData == "" {
		return nil, errors.New(ErrVisitorDataNotFound)
	}
	jsUrl := yt.getPlayerUrl(&html)
	if jsUrl == "" {
		return nil, errors.New(ErrPlaylerUrlNotFound)
	}
	PlayerUrl := "https://www.youtube.com" + jsUrl

	req, err = httpclient.NewDefaultWebRequest(PlayerUrl)
	if err != nil {
		return nil, err
	}

	resp, err = yt.client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	jsString := string(data)
	timeStamp := yt.getSts(&jsString)
	if timeStamp == "" {
		return nil, errors.New(ErrSignatureTimeStampNotFound)
	}

	sts, err := strconv.Atoi(timeStamp)
	if err != nil {
		return nil, err
	}

	apiKey := yt.getApiKey(&html)
	apiUrl := "https://www.youtube.com/youtubei/v1/player?prettyPrint=false&key=" + apiKey
	if apiKey == "" {
		yt.logger.Printf(logger.LOG_LEVEL_WARN, "[warning] api key does not found in HTML")
		apiUrl = "https://www.youtube.com/youtubei/v1/player?prettyPrint=false"
	}

	return &YtMetaData{
		SignatureTimeStamp: sts,
		VisitorData:        VisitorData,
		Cookies:            cookies,
		PlayerUrl:          PlayerUrl,
		PlayerResponse:     ytPlayer,
		ApiUrl:             apiUrl,
		InnertubeApiKey:    apiKey,
	}, nil
}

func (yt *YoutubeExtractor) fallbackToLowest(webPageMetadata *YtMetaData) (*core.DownloadItem, error) {
	if len(webPageMetadata.PlayerResponse.StreamingData.Formats) == 0 {
		respApi, err := yt.CallApi(webPageMetadata, DEFAULT_YT_CLIENT)
		if err != nil {
			return nil, errors.New("error while calling api, " + err.Error()) 
		}

		return yt.pickSingleFormats(&respApi)
	}

	contentLength, err := strconv.Atoi(webPageMetadata.PlayerResponse.StreamingData.Formats[0].ContentLength)

	var mediaInfo = []core.MediaInfo{
		{
			ID:       webPageMetadata.PlayerResponse.VideoDetails.VideoId,
			Tittle:   webPageMetadata.PlayerResponse.VideoDetails.Title,
			// TODO: do not hardcode "mp4" instead parse mimetype in json and get the codecs
			FileName: webPageMetadata.PlayerResponse.VideoDetails.Title + ".mp4",
			Size:     int64(contentLength), 
			Format: core.Format{
				URL:      webPageMetadata.PlayerResponse.StreamingData.Formats[0].Url,
				Type:     "Video+Audio",
				HasAudio: true,
			},
		},

	}

	downloadItem := core.DownloadItem{
		IsPlaylist: false,
		OutputFile: webPageMetadata.PlayerResponse.VideoDetails.Title + ".mp4",
		OutputPath: yt.configs.Directory,

		Entries: nil,
		Media: mediaInfo,
	} 

	return &downloadItem, err
}
