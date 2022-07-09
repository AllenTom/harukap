package thumbnail

import (
	"bytes"
	"context"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io"
)

type LocalThumbnailProcess struct {
}

func (p *LocalThumbnailProcess) loadImageFromByte(input io.ReadCloser) (image.Image, error) {
	thumbnailImage, _, err := image.Decode(input)
	if err != nil {
		return nil, err
	}
	return thumbnailImage, nil
}
func (p *LocalThumbnailProcess) Resize(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error) {
	thumbnailImage, err := p.loadImageFromByte(input)
	if err != nil {
		return nil, err
	}
	width := thumbnailImage.Bounds().Dx()
	height := thumbnailImage.Bounds().Dy()
	toWidth, toHeight := option.GetSize(width, height)
	// make thumbnail
	resizeImage := resize.Thumbnail(uint(toWidth), uint(toHeight), thumbnailImage, resize.Lanczos3)

	// mkdir
	outputImage := bytes.NewBuffer(nil)
	if err != nil {
		return nil, err
	}
	// save result
	err = jpeg.Encode(outputImage, resizeImage, nil)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(outputImage), nil
}
