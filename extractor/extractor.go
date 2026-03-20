package extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"net/http/httputil"
	"time"
	"goDownloader/config"
)

var tr = http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 20,
	MaxConnsPerHost:     20,
	IdleConnTimeout:     30 * time.Second,
}


type LogTransport struct {
	Base http.RoundTripper
	Debug bool
}

func (l *LogTransport)RoundTrip(r *http.Request) (*http.Response, error){
	log.Printf("SENDING REQUEST:\n%s\n", dumpRequest(r))

	resp, err := l.Base.RoundTrip(r)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return resp, err
	}
	log.Printf("RECEIVING HEADERS:\n%s\n", dumpResponseHeader(resp))

	return resp, err
}

func NewClient(debug bool) *http.Client {
	if debug {
		return &http.Client{
			Transport: &LogTransport{
				Debug: true,
			},
		}
	} else {
		return &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout: 1 * time.Second,
				
			},
		}
	}
}




type Target struct {
	FileName string
	FileSize int64
	Url 	string
}

type YtMetaData struct {
	PlayerResponse 	PlayerResponse
	VisitorData	string
	PlayerUrl 	string
	Cookies		[]*http.Cookie
	SignatureTimeStamp int
	InnertubeApiKey	string
	ApiUrl 		string
}

type Payload struct {
	Context 	Context			`json:"context"`
	VideoId 	string			`json:"videoId"`
	PlaybackContext *PlaybackContext		`json:"playbackContext,omitempty"`
	ContentCheckOk	bool 			`json:"contentCheckOk"`
	RacyCheckOk	bool			`json:"racyCheckOk"`
}

type Context struct {
	Client Client `json:"client"`
}

type Client struct {
	ClientName 	string	`json:"clientName"` 
	ClientVersion	string	`json:"clientVersion"`
	DeviceMake 	string	`json:"deviceMake,omitempty"`
	DeviceModel 	string	`json:"deviceModel,omitempty"`
	AndroidSdkVersion int	`json:"androidSdkVersion,omitempty"`
	UserAgent 	string	`json:"userAgent"`
	OsName 		string	`json:"osName,omitempty"`
	OsVersion 	string	`json:"osVersion,omitempty"`
	Hl 		string	`json:"hl"`
	TimeZone 	string	`json:"timeZone"`
	Utcoffsetminutes int	`json:"utcOffsetMinutes"`
}

type PlaybackContext struct {
	ContentPlaybackContext *ContentPlaybackContext `json:"contentPlaybackContext,omitempty"`

}

type ContentPlaybackContext struct {
		Html5Preference string `json:"html5Preference,omitempty"`
		SignatureTimeStamp int `json:"signatureTimestamp,omitempty"`
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
	} `json:"streamingData"`
}

type VideoDetails struct {
	Title 		string	`json:"title"`
	Author 		string	`json:"author"`
	VideoId 	string	`json:"videoId"`
	LengthSeconds 	string	`json:"lengthSeconds"`
	IsPrivate	bool	`json:"isPrivate"`
}

type Formats struct {
	Fps 			int	`json:"fps"`
	ProjectionType 		string	`json:"projectionType"`
	ApproxDurationMs 	string	`json:"approxDurationMs"`
	AudioSampleRate		string	`json:"audioSampleRate"`
	Itag			int	`json:"itag"`
	Bitrate			int	`json:"bitrate"`
	AverageBitrate 		int	`json:"averageBitrate"`
	QualityOrdinal 		string	`json:"qualityOrdinal"`
	AudioQuality		string	`json:"audioQuality"`
	Url			string	`json:"url"`
	SignatureCipher 	string	`json:"signatureCipher"`
	LastModified		string 	`json:"lastModified"`
	Quality 		string 	`json:"quality"`
	ContentLength 		string 	`json:"contentLength"`
	QualityLabel		string 	`json:"qualityLabel"`
	AudioChannels		int 	`json:"audioChannels"`
	MimeType 		string 	`json:"mimeType"`
	Width 			int 	`json:"width"`
	Height 			int 	`json:"height"`
}


var client *http.Client

func ExtractUrl(url string, config config.Config) ([]Target, error) {
	client = NewClient(config.Debug)
	webPageMetadata, err := ExtractWebPage(url)
	if err != nil {
		return nil, fmt.Errorf("error: error while extracting web page, %s", err)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	respApi, err := CallApi(&webPageMetadata)
	if err != nil {
		fmt.Println(err)
		return nil,fmt.Errorf("error: error while calling api, %s", err)

	}

	if respApi.PlayabilityStatus.Status != "OK" {
		log.Fatalf("[Error] %s", respApi.PlayabilityStatus.Reason )
		return nil, fmt.Errorf("error: error api response != OK")
	}

	if respApi.StreamingData == nil {
		fmt.Println("streamingData nil")
		return nil, fmt.Errorf("error: could not get streamingData")
	}

	bestAudio := pickBestAudio(respApi.StreamingData.AdaptiveFormats)
	audioSize, err := strconv.ParseInt(bestAudio.ContentLength, 10, 64)
	if err != nil {
		return nil , err
	}
	bestVideo := pickBestVideo(respApi.StreamingData.AdaptiveFormats)
	videoSize, err := strconv.ParseInt(bestVideo.ContentLength, 10, 64)
	if err != nil {
		return nil, err
	}


	fmt.Printf("Best Audio Found: itag: %s, height: %s, mimeType: %s\n",bestAudio.Itag, bestAudio.Height, bestAudio.MimeType)
	fmt.Printf("Best Video Found: Url: %s, itag: %s, height: %s, mimeType: %s\n",  bestVideo.Itag, bestVideo.Height, bestVideo.MimeType)

	return []Target{
		Target{
			FileName: webPageMetadata.PlayerResponse.VideoDetails.Title + ".mp4a",
			Url: bestAudio.Url,
			FileSize: audioSize,
		},
		Target{
			FileName: webPageMetadata.PlayerResponse.VideoDetails.Title + ".mp4",
			Url: bestVideo.Url,
			FileSize: videoSize,
		},
	}, nil
}

func ExtractWebPage(url string) (YtMetaData, error) {
	target := "ytInitialPlayerResponse"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return YtMetaData{}, err
	}
	log.Println("Extracting Web Page")

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	req.Header.Set("Accept-Language", "en-us,en;q=0.5")

	log.Printf("[Extracting Web Page] sending request\n")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return YtMetaData{}, err
	}

	log.Printf("[Extracting Web Page] receiving response\n")
	
	
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

	req, err = http.NewRequest("GET", PlayerUrl, nil)
	if err != nil {
		fmt.Println(err)
		return YtMetaData{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0;     Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")                                          
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")                                                                                      
	req.Header.Set("Accept-Language", "en-us,en;q=0.5")        

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
		SignatureTimeStamp: sts,
		VisitorData: VisitorData,
		Cookies: cookies,
		PlayerUrl: PlayerUrl,
		PlayerResponse: ytPlayer,
		ApiUrl: apiUrl,
		InnertubeApiKey: apiKey,
	}, nil

}

func extractJSON(s string, start int) (string, error) {
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
				fmt.Println("Index :", i)
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

	return "", fmt.Errorf("error: invalid JSON")
}

func getVisitorData(html string) string {
	re := regexp.MustCompile(`"VISITOR_DATA":"([^"]+)"`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		return match[1]
	}

	return match[0];
}

func getPlayerUrl(html string) string {
	re := regexp.MustCompile(`"jsUrl":"([^"]+)"`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		return match[1];
	}

	return match[0];
}

func getSts(baseJs string) string {
	re := regexp.MustCompile(`signatureTimestamp:(\d+)|sts:(\d+)`)
	match := re.FindStringSubmatch(baseJs)
	if len(match) > 1 {
		return match[1];
	}

	return match[0];
}

func getApiKey(html string) string {
	re := regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	match := re.FindStringSubmatch(html)

	if len(match) > 1 {
		return match[1]
	}

	return match[0]
}


func CallApi(ytData *YtMetaData)(PlayerResponse, error){
	req, err := MakeApiRequest(ytData, "ANDROID_VR")
	if err != nil {
		return PlayerResponse{}, err
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("[Call Api] sending request\n")

	resp, err := client.Do(req)
	if err != nil {
		return PlayerResponse{}, fmt.Errorf("[Error]: cannot do request", err)
	}


	defer resp.Body.Close()
	respApi, err := io.ReadAll(resp.Body)
	if err != nil {
		return PlayerResponse{}, err
	}
	log.Printf("[Call Api] Downloading JSON Api")
	playerResponse := PlayerResponse{}

	if resp.StatusCode == 400 {
		fmt.Println(string(respApi))
	}

	err = json.Unmarshal(respApi, &playerResponse)
	if err != nil {
		return PlayerResponse{}, err
	}

	return playerResponse, nil
}

func NewPayload(clientName, vidioId string, signatureTimestamp int)Payload {
	switch clientName {
	case "ANDROID_VR", "android_vr":
		return Payload{
			Context: Context{
				Client: Client {
					ClientName: "ANDROID_VR",
					ClientVersion: "1.71.26",
					DeviceMake: "Oculus",
					DeviceModel: "Quest 3",
					AndroidSdkVersion: 32,
					UserAgent:  "com.google.android.apps.youtube.vr.oculus/1.71.26 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
					Hl: "en",
					OsName: "Android",
					OsVersion: "12L",
					TimeZone: "UTC",
					Utcoffsetminutes: 0,	
				},
			},
			VideoId: vidioId,
			RacyCheckOk: true,
			ContentCheckOk: true,

			PlaybackContext: &PlaybackContext{
				ContentPlaybackContext: &ContentPlaybackContext{
					Html5Preference: "HTML5_PREF_WANTS",
					SignatureTimeStamp: signatureTimestamp,
				},
			},
		}

	case "WEB":
		return Payload{
			Context: Context{
				Client: Client{
					ClientName: "WEB",
					ClientVersion: "2.20260114.08.00",
					UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.5 Safari/605.1.15,gzip(gfe)",
					Hl: "en",
					TimeZone: "UTC",
					Utcoffsetminutes: 0,
				},
			},
			VideoId: vidioId,
			PlaybackContext: &PlaybackContext{
				ContentPlaybackContext: &ContentPlaybackContext{
					Html5Preference: "HTML5_PREF_WANTS",
					SignatureTimeStamp: signatureTimestamp,
				},
			},
			ContentCheckOk: true,
			RacyCheckOk: true,
		}
	case "ANDROID":
		return Payload{
			Context: Context{
				Client: Client{
					ClientName:    "ANDROID",
					ClientVersion: "17.31.35",
					UserAgent:     "com.google.android.youtube/17.31.35 (Linux; U; Android 11)",
					Hl:            "en",
					TimeZone:      "UTC",
					Utcoffsetminutes: 0,
				},
			},
			VideoId: vidioId,
			ContentCheckOk: true,
			RacyCheckOk: true,
		}
	default:
		return Payload{}
	}
}

func MakeApiRequest(ytData *YtMetaData, clientName string) (*http.Request, error) {
	switch clientName {
	case "ANDROID_VR":
		payload := NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)
		androidVrPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest("POST", ytData.ApiUrl, bytes.NewReader(androidVrPayload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "com.google.android.apps.youtube.vr.oculus/1.71.26" )
		for i := range ytData.Cookies {
			req.AddCookie(ytData.Cookies[i])
		}
		req.Header.Add("X-Youtube-Client-Name", "28")
		req.Header.Add("X-Youtube-Client-Version", "1.71.26")
		req.Header.Add("X-Goog-Visitor-Id", ytData.VisitorData)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,applicatio    n/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Add("Origin", "https://www.youtube.com")
		req.Header.Set("Content-Type", "application/json")
		return req, nil

	case "ANDROID":
		payload := NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", ytData.ApiUrl, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}


		req.Header.Set("User-Agent", "com.google.android.youtube/17.31.35")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Youtube-Client-Name", "3") // ANDROID = 3
		req.Header.Set("X-Youtube-Client-Version", "17.31.35")

		return req, nil
	case "WEB":
		payload := NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", ytData.ApiUrl, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		for i := range ytData.Cookies {
			req.AddCookie(ytData.Cookies[i])
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.5 Safari/605.1.15")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Youtube-Client-Name", "1") // ANDROID = 3
		req.Header.Set("X-Youtube-Client-Version", "2")
		req.Header.Set("X-Goog-Visitor-Id", ytData.VisitorData)
		req.Header.Set("Origin", "https://www.youtube.com")

		return req, nil
	default:
		return nil, fmt.Errorf("error: unknown clientName")
	}
}

func pickBestAudio(formats []Formats) *Formats {
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

func pickBestVideo(formats []Formats) *Formats {
	var best *Formats

	for i := range formats {
		f := &formats[i]

		if !strings.Contains(f.MimeType, "video/mp4") ||
		!strings.Contains(f.MimeType, "avc1") {
			continue
		}

		// target 1080p atau di bawahnya
		if f.Height <= 1080 {
			if best == nil || f.Height > best.Height {
				best = f
			}
		}
	}

	return best
}

func dumpRequest(req *http.Request) string {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return "dump error:" + err.Error()
	}
	return string(dump)
}


func dumpResponseHeader(resp *http.Response) string {
	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		return "dump error: " + err.Error() 
	}

	return string(dump)
}
		
