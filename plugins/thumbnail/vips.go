package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

type VipsThumbnailEngine struct {
	Target string
}

func (e *VipsThumbnailEngine) Resize(ctx context.Context, input io.ReadCloser, option ThumbnailOption) (io.ReadCloser, error) {
	read, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(read))
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile("./img_tmp.jpg", read, 0644)
	if err != nil {
		return nil, err
	}
	defer os.Remove("./img_tmp.jpg")
	tempFilePath := "./output_tmp." + format
	cmd := exec.Command(e.Target, fmt.Sprintf("--size=%dx", option.MaxWidth), "./img_tmp.jpg", "-o", tempFilePath)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFilePath)
	buf := bytes.NewBuffer(data)
	return ioutil.NopCloser(buf), nil
}

func NewVipsThumbnailEngine(target string) *VipsThumbnailEngine {
	return &VipsThumbnailEngine{Target: target}
}
