package xconf

import (
	"bytes"
	"context"
	"encoding/json"
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
		Batter []Batter `json:"batter"`
	} `json:"batters"`
}

func TestCurd(t *testing.T) {
	xconf := New(&Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
	})
	f := File{
		Group:   "xconf",
		Name:    "xconftest-test.json",
		Content: jsonExample,
	}

	if err := xconf.Put(context.TODO(), f); err != nil {
		t.Fatal(err)
	}

	xconf.Watch(context.TODO(), f.Group, f.Name, func(file File) error {
		t.Log(file)
		return nil
	})

	content, err := xconf.Get(context.TODO(), f.Group, f.Name)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(content, f.Content) != 0 {
		t.Errorf("compare file content failed")
	}

	if err = xconf.Delete(context.TODO(), f.Group, f.Name); err != nil {
		t.Errorf("delete file failed")
	}
}

func TestWatch(t *testing.T) {
	xconf := New(&Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
	})
	f := File{
		Group:   "xconf",
		Name:    "xconftest-test.json",
		Content: jsonExample,
		Meta:    &Metadata{},
	}

	if err := xconf.Put(context.TODO(), f); err != nil {
		t.Fatal(err)
	}

	ch := make(chan File)

	xconf.Watch(context.TODO(), f.Group, f.Name, func(file File) error {
		ch <- file
		return nil
	})

	// update
	example := new(Example)
	if err := json.Unmarshal(f.Content, example); err != nil {
		t.Fatal(err)
	}
	example.Name = "Cake2"
	f.Content, _ = json.Marshal(example)
	xconf.Put(context.TODO(), f)
	newfile := <-ch
	if bytes.Compare(newfile.Content, f.Content) != 0 {
		t.Errorf("compare file content failed")
	}

	// update with gray
}
