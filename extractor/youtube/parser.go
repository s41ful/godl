package youtube

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type HlsSegmentUrl string


type HlsFormat struct {
	bandwith			int
	codecs        [2]string
	width 				int
	height				int
	subtitles			string
	frameRate     int
	url           string
}

func ParseHlsPlaylist(hlsPlaylist string) []HlsFormat {
	var fHls []HlsFormat

	list := strings.Split(hlsPlaylist, "\n")

	for i, el := range list {
		if strings.Contains(el, "#EXT-X-STREAM-INF:") {
			el = strings.TrimLeft(el, "#EXT-X-STREAM-INF:")
			var format = map[string]string{}
			parts := strings.Split(el, ",")

			for i, el := range parts {
				part := strings.Split(el, "=") 

				if string(part[0][len(part[0]) - 1]) == "\"" {
					continue
				}

				if string(part[1][0]) == "\"" {
					part[1] += "," + parts[i + 1]
					//fmt.Printf("parsing %s\n", part[0] + part[1])
					format[part[0]] = part[1]
					continue
				} 

				

				//fmt.Printf("parsing %s\n", el)
				format[part[0]] = part[1]
			}

			//fmt.Println("Atoi: ", format["BANDWITH"])
			bandwith, _ := strconv.Atoi(format["BANDWIDTH"])

			res := strings.Split(format["RESOLUTION"], "x")
			width, _ := strconv.Atoi(res[0])
			height, _ := strconv.Atoi(res[1])

			codecs := strings.Split(format["CODECS"], ",")
			frameRate, _ := strconv.Atoi(format["FRAME-RATE"])

			fHls = append(fHls, HlsFormat{
				bandwith: 		bandwith,
				width: 				width,
				height: 			height,
				codecs: 			[2]string{codecs[0], codecs[1]},
				frameRate: 		frameRate,
				subtitles: 		format["SUBTITLES"],
				url: 					list[i + 1],
			})

		}
	}


	return fHls
}

func GetSegmentsFromMediaPlaylist(mediaPlaylist string) []HlsSegmentUrl {
	var segments []HlsSegmentUrl
	list := strings.Split(mediaPlaylist, "\n")

	for _, el := range list {
		if !strings.Contains(el, "#EXT") {
			segments = append(segments, HlsSegmentUrl(el))
		}

	}

	return segments
}



func ParseRangeFromURL(u string) (int64, int64, error) {
	var re = regexp.MustCompile(`/begin/(\d+)/len/(\d+)/`)
	m := re.FindStringSubmatch(u)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("range not found")
	}

	begin, _ := strconv.ParseInt(m[1], 10, 64)
	length, _ := strconv.ParseInt(m[2], 10, 64)

	return begin, length, nil
}
