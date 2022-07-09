package thumbnail

import (
	"context"
	"fmt"
	"github.com/allentom/harukap"
	"io"
)

type ThumbnailOption struct {
	MaxWidth  int    `hsource:"query" hname:"maxWidth"`
	MaxHeight int    `hsource:"query" hname:"maxHeight"`
	Mode      string `hsource:"query" hname:"mode"`
}
type Engine struct {
	Process   map[string]ThumbnailProcess
	UseEngine string
}

func NewEngine() *Engine {
	return &Engine{
		Process: map[string]ThumbnailProcess{},
	}
}

func (t *Engine) Resize(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error) {
	defaultEngine := t.Process[t.UseEngine]
	if defaultEngine == nil {
		return nil, fmt.Errorf("no default engine")
	}
	return defaultEngine.Resize(ctx, input, option)
}

func (t *Engine) OnInit(e *harukap.HarukaAppEngine) error {
	configManager := e.ConfigProvider.Manager
	rawThumbnails := configManager.GetStringMap("thumbnails")
	for name := range rawThumbnails {
		storageType := configManager.GetString(fmt.Sprintf("thumbnails.%s.type", name))
		if storageType == "" {
			continue
		}
		switch storageType {
		case "thumbnailservice":
			thumbnailServicePlugin := &ThumbnailServicePlugin{
				Prefix: name,
			}
			err := thumbnailServicePlugin.OnInit(e)
			if err != nil {
				return err
			}
			t.Process[name] = thumbnailServicePlugin
		case "local":
			local := &LocalThumbnailProcess{}
			t.Process[name] = local
		case "vips":
			target := configManager.GetString(fmt.Sprintf("thumbnails.%s.target", name))
			vips := &VipsThumbnailEngine{
				Target: target,
			}
			t.Process[name] = vips
		}
	}
	t.UseEngine = configManager.GetString("thumbnails.default")
	return nil
}
