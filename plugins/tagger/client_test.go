package tagger

import (
	"os"
	"testing"
)

func InitClient(t *testing.T) (*Client, error) {
	url := os.Getenv("URL")
	if url == "" {
		t.Fatal("URL env is not set")
	}
	client := NewClient(url)
	return client, nil
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
	tags, err := client.TagImage(file)
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
