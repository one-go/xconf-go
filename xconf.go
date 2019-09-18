package xconf

import (
	"context"
	"errors"

	"go.etcd.io/etcd/clientv3"
)

var (
	// ErrNotFound is the error returned by file not exists
	ErrNotFound = errors.New("not found")
)

const (
	DefaultPrefix    = "/xconf/"
	DefaultNamespace = "x"
)

type File struct {
	Group   string
	Name    string
	Content []byte
	version int64
}

type Xconf struct {
	prefix string
	client *clientv3.Client
}

type Options struct {
	Endpoints []string `json:"endpoints"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Namespace string   `json:"namespace"`
	CacheDir  string   `json:"cache-dir"`
}

func New(opt *Options) *Xconf {
	if opt.Namespace == "" {
		opt.Namespace = DefaultNamespace
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: opt.Endpoints,
		Username:  opt.Username,
		Password:  opt.Password,
	})
	if err != nil {
		// handle error!
	}
	xconf := &Xconf{
		prefix: DefaultPrefix + opt.Namespace + "/",
		client: cli,
	}

	return xconf
}

type OnChange func(File) error

func (c *Xconf) Put(ctx context.Context, file File) error {
	_, err := c.client.Put(ctx, c.Key(file.Group, file.Name), string(file.Content))
	return err
}

func (c *Xconf) Get(ctx context.Context, group, name string) ([]byte, error) {
	res, err := c.client.Get(ctx, c.Key(group, name))
	if err != nil {
		return nil, err
	}
	if res.Count == 0 {
		return nil, ErrNotFound
	}
	return res.Kvs[0].Value, nil
}

func (c *Xconf) Delete(ctx context.Context, group, name string) error {
	_, err := c.client.Delete(ctx, c.Key(group, name))
	return err
}

func (c *Xconf) Key(group, name string) string {
	return c.prefix + group + "/" + name
}

func (c *Xconf) Watch(ctx context.Context, group, name string, h OnChange) error {
	go func() {
		ch := c.client.Watch(ctx, c.Key(group, name))
		for wresp := range ch {
			for _, ev := range wresp.Events {
				h(File{group, name, ev.Kv.Value, ev.Kv.Version})
			}
		}
	}()
	return nil
}
