package core

type DownloadItem struct {
	OutputFile string
	OutputPath string
	Media      []MediaInfo

	IsPlaylist bool
	Entries    *[]DownloadItem
}

type Format struct {
	ID   string
	URL  string
	Size int64

	Type string

	VideoCodec string
	AudioCodec string

	Width  int
	Height int

	HasVideo bool
	HasAudio bool

	IsHLS  bool
	IsDASH bool
}

type MediaInfo struct {
	ID       string
	Tittle   string
	Size     int64
	FileName string

	Format Format
}
