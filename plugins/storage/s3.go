package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/allentom/harukap"
	util "github.com/allentom/harukap/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3ClientConfig struct {
	Id       string `json:"id"`
	Secret   string `json:"secret"`
	Region   string `json:"region"`
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
	Password string `json:"password"`
}
type S3Client struct {
	Session    *session.Session
	Service    *s3.S3
	ConfigName string
	Config     *S3ClientConfig
}

func (c *S3Client) Copy(ctx context.Context, bucket, key, destBucket, destKey string) error {
	_, err := c.Service.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(destBucket),
		Key:        aws.String(destKey),
		CopySource: aws.String(fmt.Sprintf("%s/%s", bucket, key)),
	})
	return err
}

func (c *S3Client) IsExist(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.Service.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}
	return true, nil
}

func (c *S3Client) Delete(ctx context.Context, bucket, key string) error {
	_, err := c.Service.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *S3Client) OnInit(e *harukap.HarukaAppEngine) error {
	if c.ConfigName == "" {
		c.ConfigName = "s3"
	}
	baseKeyPath := fmt.Sprintf("storage.%s", c.ConfigName)
	if c.Config == nil {
		c.Config = &S3ClientConfig{
			Id:       e.ConfigProvider.Manager.GetString(baseKeyPath + ".id"),
			Secret:   e.ConfigProvider.Manager.GetString(baseKeyPath + ".secret"),
			Region:   e.ConfigProvider.Manager.GetString(baseKeyPath + ".region"),
			Token:    e.ConfigProvider.Manager.GetString(baseKeyPath + ".token"),
			Endpoint: e.ConfigProvider.Manager.GetString(baseKeyPath + ".endpoint"),
			Password: e.ConfigProvider.Manager.GetString(baseKeyPath + ".password"),
		}
	}
	logger := e.LoggerPlugin.Logger.NewScope("S3Storage")
	logger.WithFields(map[string]interface{}{
		"name":     c.ConfigName,
		"region":   c.Config.Region,
		"endpoint": c.Config.Endpoint,
		"id":       util.MaskKeepHeadTail(c.Config.Id, 2, 2),
		"secret":   util.MaskKeepHeadTail(c.Config.Secret, 2, 2),
		"token":    util.MaskKeepHeadTail(c.Config.Token, 1, 2),
		"password": util.MaskKeepHeadTail(c.Config.Password, 1, 2),
	}).Info("s3 storage config")
	return c.Init()
}

func (c *S3Client) Init() error {
	if c.Session == nil {
		c.Session = session.Must(session.NewSession(
			&aws.Config{
				Credentials:      credentials.NewStaticCredentials(c.Config.Id, c.Config.Secret, c.Config.Token),
				Endpoint:         aws.String(c.Config.Endpoint),
				S3ForcePathStyle: aws.Bool(true),
				Region:           aws.String(c.Config.Region),
			}))
	}
	if c.Service == nil {
		c.Service = s3.New(c.Session)
	}
	return nil
}

func (c *S3Client) Upload(ctx context.Context, body io.Reader, bucket string, key string) error {
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	// test string body
	_, err = c.Service.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   aws.ReadSeekCloser(bytes.NewReader(buf)),
		ContentMD5: func() *string {
			h := md5.New()
			h.Write(buf)
			return aws.String(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		}(),
	})
	if err != nil {
		return err
	}
	return nil
}
func (c *S3Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	output, err := c.Service.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}
