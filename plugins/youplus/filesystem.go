package youplus

import (
	"bytes"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/afero"
	"io/fs"
	"os"
	"time"
)

type FileSystemClient struct {
	client  *resty.Client
	baseUrl string
	Auth    string
}

func (c *FileSystemClient) Create(name string) (afero.File, error) {
	file := NewFile(c, name)
	_, err := c.client.R().SetQueryParam("path", name).
		SetResult(&file).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/create")
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *FileSystemClient) Mkdir(name string, perm os.FileMode) error {
	_, err := c.client.R().SetQueryParam("path", name).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/mkdir")
	if err != nil {
		return err
	}
	return nil
}

func (c *FileSystemClient) MkdirAll(path string, perm os.FileMode) error {
	_, err := c.client.R().SetQueryParam("path", path).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/mkdirall")
	if err != nil {
		return err
	}
	return nil
}

func (c *FileSystemClient) Open(name string) (afero.File, error) {
	file := NewFile(c, name)
	_, err := c.client.R().SetQueryParam("path", name).
		SetResult(&file).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/open")
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *FileSystemClient) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file := NewFile(c, name)
	_, err := c.client.R().SetQueryParam("path", name).
		SetResult(&file).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/open")
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (c *FileSystemClient) Remove(name string) error {
	_, err := c.client.R().SetQueryParam("path", name).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/remove")
	if err != nil {
		return err
	}
	return nil
}

func (c *FileSystemClient) RemoveAll(path string) error {
	_, err := c.client.R().SetQueryParam("path", path).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/removeall")
	if err != nil {
		return err
	}
	return nil
}

func (c *FileSystemClient) Rename(oldname, newname string) error {
	_, err := c.client.R().SetQueryParam("path", newname).
		SetQueryParam("source", oldname).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/rename")
	if err != nil {
		return err
	}
	return nil
}

func (c *FileSystemClient) Stat(name string) (os.FileInfo, error) {
	file := NewFile(c, name)
	_, err := c.client.R().SetQueryParam("path", name).
		SetResult(&file).
		SetHeader("Authorization", "Bearer "+c.Auth).
		Get(c.baseUrl + "/fs/open")
	if err != nil {
		return nil, err
	}
	return file.Info, nil
}

func (c *FileSystemClient) Name() string {
	return "youplus"
}

func (c *FileSystemClient) Chmod(name string, mode os.FileMode) error {
	return nil
}

func (c *FileSystemClient) Chown(name string, uid, gid int) error {
	return nil
}

func (c *FileSystemClient) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return nil
}

type File struct {
	fs     *FileSystemClient
	Info   *FileInfo `json:"info"`
	path   string
	offset int64
	whence int
}

func NewFile(fs *FileSystemClient, path string) *File {
	return &File{
		fs:   fs,
		path: path,
	}
}
func (f *File) Close() error {
	return nil
}

func (f *File) Read(p []byte) (n int, err error) {
	resp, err := f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("off", fmt.Sprintf("%d", f.offset)).
		SetQueryParam("whence", fmt.Sprintf("%d", len(p))).
		Get(f.fs.baseUrl + "/fs/file/read")
	if err != nil {
		return 0, err
	}
	buf := bytes.NewBuffer(resp.Body())
	return buf.Read(p)
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	resp, err := f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("off", fmt.Sprintf("%d", f.offset)).
		SetQueryParam("whence", fmt.Sprintf("%d", f.whence)).
		Get(f.fs.baseUrl + "/fs/file/read")
	if err != nil {
		return 0, err
	}
	buf := bytes.NewBuffer(resp.Body())
	return buf.Read(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	f.offset = offset
	f.whence = whence
	return offset, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	_, err = f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("off", fmt.Sprintf("%d", f.offset)).
		SetQueryParam("whence", fmt.Sprintf("%d", f.whence)).
		SetBody(p).
		Post(f.fs.baseUrl + "/fs/file/write")
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	_, err = f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("off", fmt.Sprintf("%d", off)).
		SetQueryParam("whence", fmt.Sprintf("%d", f.whence)).
		SetBody(p).
		Post(f.fs.baseUrl + "/fs/file/write")
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (f *File) Name() string {
	return f.Info.FileName
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	var files []*FileInfo
	_, err := f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("count", fmt.Sprintf("%d", count)).
		SetResult(&files).
		Get(f.fs.baseUrl + "/fs/file/readdir")
	if err != nil {
		return nil, err
	}
	result := make([]os.FileInfo, len(files))
	for i, file := range files {
		result[i] = file
	}
	return result, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	var files []*FileInfo
	_, err := f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("count", fmt.Sprintf("%d", n)).
		SetResult(&files).
		Get(f.fs.baseUrl + "/fs/file/readdir")
	if err != nil {
		return nil, err
	}
	result := make([]string, len(files))
	for i, file := range files {
		result[i] = file.Name()
	}
	return result, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return f.Info, nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	_, err := f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("size", fmt.Sprintf("%d", size)).
		Get(f.fs.baseUrl + "/fs/file/truncate")
	if err != nil {
		return err
	}
	return nil
}

func (f *File) WriteString(s string) (ret int, err error) {
	_, err = f.fs.client.R().SetQueryParam("path", f.path).
		SetHeader("Authorization", "Bearer "+f.fs.Auth).
		SetQueryParam("off", fmt.Sprintf("%d", f.offset)).
		SetQueryParam("whence", fmt.Sprintf("%d", f.whence)).
		SetBody([]byte(s)).
		Post(f.fs.baseUrl + "/fs/file/write")
	if err != nil {
		return 0, err
	}
	return len(s), nil
}

type FileInfo struct {
	FileName    string `json:"name"`
	FileSize    int64  `json:"size"`
	FileMode    uint32 `json:"mode"`
	FileModTime string `json:"modTime"`
	FileIsDir   bool   `json:"isDir"`
}

func (i *FileInfo) Name() string {
	return i.FileName
}

func (i *FileInfo) Size() int64 {
	return i.FileSize
}

func (i *FileInfo) Mode() fs.FileMode {
	return fs.FileMode(i.FileMode)
}

func (i *FileInfo) ModTime() time.Time {
	modTile, _ := time.Parse("2006-01-02 15:04:05", i.FileModTime)
	return modTile
}

func (i *FileInfo) IsDir() bool {
	return i.FileIsDir
}

func (i *FileInfo) Sys() any {
	return nil
}
