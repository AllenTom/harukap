package util

import (
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

func GuessImageFormat(r io.Reader) (format string, err error) {
	_, format, err = image.DecodeConfig(r)
	return
}
