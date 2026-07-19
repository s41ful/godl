# godl

A fast and lightweight media downloader written in Go.

## Features

- Easily add support for new websites
- Concurrent downloads
- HTTP Range Requests
- Cross-platform
- No dependency
- Written in Go

## Supported Extractors

| Site | Status |
|------|--------|
| YouTube | ✅ |
| Generic HTTP/S | ✅ |

## Roadmap

- [x] YouTube extractor
- [x] Generic HTTP extractor
- [x] Mp4 codecs support
- [ ] Another codecs support
- [ ] Resume interupted download
- [ ] Configuration support
- [ ] Thumbnail support
- [ ] Subtittles support

## Requirements

- Go 1.25.4 or later (only if building from source)
- FFmpeg (required for merging audio/video and media processing)

Verify FFmpeg is installed:

```bash
ffmpeg -version
```

## Installation

```bash
git clone https://github.com/s41ful/godl.git
cd godl
make build
```

## Usage

```bash
godl [options] <URL>
```
