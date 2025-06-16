package tagger

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func InitClient(t *testing.T) (*Client, error) {
	url := os.Getenv("URL")
	if url == "" {
		t.Fatal("URL env is not set")
	}
	config := DefaultConfig()
	config.BaseURL = url
	config.Timeout = 5 * time.Second // 测试时使用较短的超时时间
	return NewClientWithConfig(config), nil
}

func InitConsulClient(t *testing.T) (*Client, error) {
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		t.Fatal("CONSUL_ADDR env is not set")
	}
	serviceName := os.Getenv("CONSUL_SERVICE")
	if serviceName == "" {
		t.Fatal("CONSUL_SERVICE env is not set")
	}

	config := DefaultConfig()
	config.ConsulConfig = api.DefaultConfig()
	config.ConsulConfig.Address = consulAddr
	config.ServiceName = serviceName
	config.Timeout = 5 * time.Second // 测试时使用较短的超时时间
	config.EnableDebug = true        // 测试时启用调试模式

	return NewClientWithConfig(config), nil
}

func TestTagger(t *testing.T) {
	client, err := InitClient(t)
	if err != nil {
		t.Fatal(err)
	}
	imagePath := os.Getenv("TEST_IMAGE")
	if imagePath == "" {
		t.Fatal("TEST_IMAGE env is not set")
	}
	file, err := os.Open(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tags, err := client.TagImage(file, "auto", 0.7)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) == 0 {
		t.Fatal("tags is empty")
	}
}

func TestConsulTagger(t *testing.T) {
	client, err := InitConsulClient(t)
	if err != nil {
		t.Fatal(err)
	}
	imagePath := os.Getenv("TEST_IMAGE")
	if imagePath == "" {
		t.Fatal("TEST_IMAGE env is not set")
	}
	file, err := os.Open(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tags, err := client.TagImage(file, "auto", 0.7)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) == 0 {
		t.Fatal("tags is empty")
	}
}

func TestClient_GetInfo(t *testing.T) {
	client, err := InitClient(t)
	if err != nil {
		t.Fatal(err)
	}
	info, err := client.GetInfo()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Success {
		t.Fatal("info success is false")
	}
}

func TestConsulClient_GetInfo(t *testing.T) {
	client, err := InitConsulClient(t)
	if err != nil {
		t.Fatal(err)
	}
	info, err := client.GetInfo()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Success {
		t.Fatal("info success is false")
	}
}

func TestClient_SwitchModel(t *testing.T) {
	client, err := InitClient(t)
	if err != nil {
		t.Fatal(err)
	}
	err = client.SwitchModel("Deepdanbooru")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsulClient_SwitchModel(t *testing.T) {
	client, err := InitConsulClient(t)
	if err != nil {
		t.Fatal(err)
	}
	err = client.SwitchModel("Deepdanbooru")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetTaggerState(t *testing.T) {
	client, err := InitClient(t)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetTaggerState()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsulClient_GetTaggerState(t *testing.T) {
	client, err := InitConsulClient(t)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetTaggerState()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_GetServiceUrl(t *testing.T) {
	// 测试普通客户端
	client, err := InitClient(t)
	if err != nil {
		t.Fatal(err)
	}
	url, err := client.getServiceUrl()
	if err != nil {
		t.Fatal(err)
	}
	if url != os.Getenv("URL") {
		t.Fatalf("expected url %s, got %s", os.Getenv("URL"), url)
	}

	// 测试 Consul 客户端
	consulClient, err := InitConsulClient(t)
	if err != nil {
		t.Fatal(err)
	}
	url, err = consulClient.getServiceUrl()
	if err != nil {
		t.Fatal(err)
	}
	if url == "" {
		t.Fatal("consul service url is empty")
	}
}

// 测试自定义配置
func TestClient_WithCustomConfig(t *testing.T) {
	config := DefaultConfig()
	config.Timeout = 10 * time.Second
	config.RetryCount = 5
	config.RetryWaitTime = 2 * time.Second
	config.MaxRetryWaitTime = 20 * time.Second
	config.EnableDebug = true

	client := NewClientWithConfig(config)
	if client.config.Timeout != 10*time.Second {
		t.Errorf("expected timeout %v, got %v", 10*time.Second, client.config.Timeout)
	}
	if client.config.RetryCount != 5 {
		t.Errorf("expected retry count %d, got %d", 5, client.config.RetryCount)
	}
	if client.config.RetryWaitTime != 2*time.Second {
		t.Errorf("expected retry wait time %v, got %v", 2*time.Second, client.config.RetryWaitTime)
	}
	if client.config.MaxRetryWaitTime != 20*time.Second {
		t.Errorf("expected max retry wait time %v, got %v", 20*time.Second, client.config.MaxRetryWaitTime)
	}
	if !client.config.EnableDebug {
		t.Error("expected debug mode to be enabled")
	}
}
