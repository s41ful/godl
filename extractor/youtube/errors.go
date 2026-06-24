package youtube

const (
	ErrInvalidYoutubeUrl               = "error invalid youtube url"
	ErrYtInitialPlayerResponseNotFound = "could not get YtInitialPlaylerResponse in watch URL"
	ErrInvalidJSON					   = "error invalid JSON"
	ErrPlaylerUrlNotFound			   = "error could not find js player URL in watch HTML"
	ErrVisitorDataNotFound 			   = "error could not find VISITOR_DATA in watch HTML"
	ErrSignatureTimeStampNotFound 	   = "error could not find signatureTimestamp in baseJS"
)
