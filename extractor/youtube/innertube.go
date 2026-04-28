package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (yt *YoutubeExtractor) CallApi(ytData *YtMetaData, ytClient string)(PlayerResponse, error){
		req, err := yt.MakeApiRequest(ytData, ytClient)
		if err != nil {
				return PlayerResponse{}, err
		}

		resp, err := client.Do(req)
		if err != nil {
				return PlayerResponse{}, fmt.Errorf("[Error]: cannot do request", err)
		}

		defer resp.Body.Close()
		respApi, err := io.ReadAll(resp.Body)
		if err != nil {
				return PlayerResponse{}, err
		}

		fmt.Printf("[Call Api] Downloading %s JSON Api\n", ytClient)
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

func (yt *YoutubeExtractor) NewPayload(clientName, vidioId string, signatureTimestamp int)Payload {
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

func (yt *YoutubeExtractor) MakeApiRequest(ytData *YtMetaData, clientName string) (*http.Request, error) {
		switch clientName {
		case "ANDROID_VR":
				payload := yt.NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)
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
				payload := yt.NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)

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
				payload := yt.NewPayload(clientName, ytData.PlayerResponse.VideoDetails.VideoId, ytData.SignatureTimeStamp)

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
