package thumbnail

import (
	"context"
	"io"
)

type ThumbnailProcess interface {
	Resize(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error)
}
