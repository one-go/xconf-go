package xconf

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"testing"
)

var jsonExample = []byte(`{
"id": "0001",
"type": "donut",
"name": "Cake",
"ppu": 0.55,
"batters": {
        "batter": [
                { "type": "Regular" },
                { "type": "Chocolate" },
                { "type": "Blueberry" },
                { "type": "Devil's Food" }
            ]
    }
}`)

type Batter struct {
	Type string `json:"type"`
}

type Example struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	PPU     float32 `json:"ppu"`
	Batters struct {
		Batter []struct {
			Type string `json:"type"`
		} `json:"batter"`
	} `json:"batters"`
}

func TestCurd(t *testing.T) {
	xconf := New(&Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
	})
	f := File{
		Group:   "xconf-sdk",
		Name:    "xconftest-test.json",
		Content: jsonExample,
	}

	if err := xconf.CreateFile(context.TODO(), &f); err != nil {
		t.Fatal("create config", err)
	}

	content, err := xconf.GetConfig(context.TODO(), f.Group, f.Name)
	if err != nil {
		t.Fatal("get config", err)
	}

	if bytes.Compare(content, f.Content) != 0 {
		t.Errorf("compare file content failed")
	}

	if err = xconf.DeleteFile(context.TODO(), f.Group, f.Name); err != nil {
		t.Errorf("delete file failed")
	}
}

func TestWatch(t *testing.T) {
	client1 := New(&Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
		ID:        "client1",
	})
	client2 := New(&Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
		ID:        "client2",
	})
	f := &File{
		Group:   "xconf-sdk",
		Name:    "xconftest-test2.json",
		Content: jsonExample,
		Meta:    Metadata{},
	}

	if err := client1.CreateFile(context.TODO(), f); err != nil {
		t.Fatal(err)
	}

	ch := make(chan File)

	client1.Watch(context.TODO(), f.Group, f.Name, func(file *File) error {
		ch <- *file
		log.Printf("client1 watch")
		return nil
	})
	client2.Watch(context.TODO(), f.Group, f.Name, func(file *File) error {
		ch <- *file
		log.Printf("client2 watch")
		return nil
	})

	// update with gray
	example := new(Example)
	if err := json.Unmarshal(f.Content, example); err != nil {
		t.Fatal(err)
	}
	example.Name = "Cake2"
	f.Content, _ = json.Marshal(example)
	f.Meta.Gray = "client2"

	client1.UpdateFile(context.TODO(), f)

	newfile := <-ch
	if bytes.Compare(newfile.Content, f.Content) != 0 {
		t.Errorf("compare file content failed version=%d", f.Version)
	}

	// update all
	f.Meta.Gray = ""
	client1.UpdateFile(context.TODO(), f)

	newfile = <-ch
	if bytes.Compare(newfile.Content, f.Content) != 0 {
		t.Errorf("compare file content failed version=%d", f.Version)
	}

	newfile = <-ch
	if bytes.Compare(newfile.Content, f.Content) != 0 {
		t.Errorf("compare file content failed version=%d", f.Version)
	}
}
