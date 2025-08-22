package storage

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/allentom/harukap"
	"github.com/spf13/afero"
)

type LocalStorageConfig struct {
	Path string `json:"path"`
}
type LocalStorage struct {
	fs         afero.Fs
	Config     *LocalStorageConfig
	ConfigName string
}

func (l *LocalStorage) IsExist(ctx context.Context, bucket, key string) (bool, error) {
	_, err := l.fs.Stat(filepath.Join(bucket, key))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (l *LocalStorage) Copy(ctx context.Context, bucket, key, destBucket, destKey string) error {
	srcPath := filepath.Join(bucket, key)
	destPath := filepath.Join(destBucket, destKey)
	err := l.fs.MkdirAll(filepath.Dir(destPath), 0755)
	if err != nil {
		return err
	}
	destFile, err := l.fs.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()
	srcFile, err := l.fs.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func (l *LocalStorage) Delete(ctx context.Context, bucket, key string) error {
	storePath := filepath.Join(bucket, key)
	return l.fs.Remove(storePath)
}

func (l *LocalStorage) OnInit(e *harukap.HarukaAppEngine) error {
	if l.ConfigName == "" {
		l.ConfigName = "local"
	}
	baseKeyPath := fmt.Sprintf("storage.%s", l.ConfigName)
	if l.Config == nil {
		l.Config = &LocalStorageConfig{
			Path: e.ConfigProvider.Manager.GetString(baseKeyPath + ".path"),
		}
	}
	logger := e.LoggerPlugin.Logger.NewScope("LocalStorage")
	logger.WithFields(map[string]interface{}{
		"name": l.ConfigName,
		"path": l.Config.Path,
	}).Info("local storage config")
	return l.Init()
}

func (l *LocalStorage) Upload(ctx context.Context, body io.Reader, bucket string, key string) error {
	storePath := filepath.Join(bucket, key)
	err := l.fs.MkdirAll(filepath.Dir(storePath), 0755)
	if err != nil {
		return err
	}
	file, err := l.fs.Create(filepath.Join(bucket, key))
	if err != nil {
		return err
	}

	defer file.Close()
	raw, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	_, err = file.Write(raw)
	if err != nil {
		return err
	}
	return nil
}

func (l *LocalStorage) Init() error {
	l.fs = afero.NewBasePathFs(afero.NewOsFs(), l.Config.Path)
	return nil
}

func (l *LocalStorage) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	file, err := l.fs.Open(filepath.Join(bucket, key))
	if err != nil {
		return nil, err
	}
	return file, nil
}
