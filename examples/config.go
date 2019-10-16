package main

import (
	"context"
	"encoding/json"

	"github.com/one-go/xconf-go"
)

var (
	xconfClient    *xconf.Xconf
	xconfConfigs   [2]*Config
	xconfConfigIdx int
)

func XconfLoad(content []byte) error {
	c := new(Config)
	idx := (xconfConfigIdx + 1) % len(xconfConfigs)
	if err := json.Unmarshal(content, c); err != nil {
		return err
	}
	xconfConfigs[idx] = c
	xconfConfigIdx = idx
	return nil
}

func Xconf() *Config {
	return xconfConfigs[xconfConfigIdx]
}

func XconfInit(opt *xconf.Options, group, name string) error {
	xconfClient = xconf.New(opt)
	xconfClient.Watch(context.TODO(), group, name, func(file *xconf.File) error {
		if err := XconfLoad(file.Content); err != nil {
		}
		return nil
	})

	content, _ := xconfClient.GetConfig(context.TODO(), group, name)
	return XconfLoad(content)
}

// Config which
type Config struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	PPU     float32 `json:"ppu"`
	Batters struct {
		Batter []string `json:"batter"`
	} `json:"batters"`
}
