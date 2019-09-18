package main

import (
	"context"
	"log"

	"github.com/one-go/xconf-go"
)

func main() {
	xcli := xconf.New(&xconf.Options{
		Endpoints: []string{"test.riodev.oa.com:2379"},
		Username:  "",
		Password:  "",
	})
	group := "xconf"
	name := "xconftest-test.json"

	xcli.Watch(context.TODO(), group, name, func(file xconf.File) error {
		log.Printf("content: %s", string(file.Content))
		return nil
	})

	content, _ := xcli.Get(context.TODO(), group, name)
	log.Printf("content: %s", string(content))
}
