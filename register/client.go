package register

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/xid"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

type RegisterClient struct {
	Client    *clientv3.Client
	Endpoints []string
}

func (c *RegisterClient) Init() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	c.Client = cli
	return nil
}

func RegisterFromFile(configPath string, client *RegisterClient) error {
	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	data := make(map[string]interface{})
	err = yaml.Unmarshal(raw, &data)
	if err != nil {
		return err
	}
	prefix := data["prefix"].(string)
	serviceList := data["services"].(map[string]interface{})
	for _, rawService := range serviceList {
		content := rawService.(map[string]interface{})
		service := Service{}
		service.Id = fmt.Sprintf("%s-%s", prefix, xid.New().String())
		err = mapstructure.Decode(content, &service)
		if err != nil {
			return err
		}
		err = service.Init(client)
		if err != nil {
			return err
		}
		err = service.Register()
		if err != nil {
			return err
		}
	}
	return nil
}
