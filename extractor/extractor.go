package extractor

import (
	"errors"
	"fmt"
	"godl/config"
	"godl/core"
	"godl/extractor/youtube"
)

type Extractor interface {
	Match (url string) bool
	Extract(url string) (*core.DownloadItem, error)
	InitConfig(cfg *config.Config)
}

type InfoExtractor struct {
	cfg    				*config.ExtractorConfig
	config 				*config.Config
}

func NewInfoExtractor(config *config.Config) InfoExtractor {
	return InfoExtractor{
		cfg: 			config.ExtractorConfig,
		config: 		config,
	}
}

func (ie *InfoExtractor) FindExtractor() (Extractor, error) {
	var extractors = []Extractor{
		&youtube.YoutubeExtractor{},
	}

	for _, extractor := range extractors {
		if extractor.Match(ie.config.Url) {
			fmt.Printf("returning youtube extractor\n")
			extractor.InitConfig(ie.config)
			return extractor, nil
		}
	}

	return nil, errors.New(ErrExtractorNotFound)
}

func (ie *InfoExtractor) Start() (*core.DownloadItem, error) {
	extractor, err := ie.FindExtractor()
	if err != nil {
		return nil, err
	}

	extractor.InitConfig(ie.config)
	fmt.Printf("initializing config\n")
	
	return extractor.Extract(ie.config.Url)
}

