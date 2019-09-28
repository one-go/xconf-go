package xconf

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/etcd-io/etcd/clientv3"
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
	Meta    *Metadata
}

type Metadata struct {
	Version    int64    `json:"-"`
	CreateTime int64    `json:"createTime"`
	UpdateTime int64    `json:"updateTime"`
	Gray       []string `json:"gray"`
}

type Xconf struct {
	id     string
	prefix string
	client *clientv3.Client
}

type Options struct {
	ID        string   `json:"id"`
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
	if opt.ID == "" {
		opt.ID, _ = os.Hostname()
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
		id:     opt.ID,
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
		keyName := c.Key(group, name)
		metaName := keyName + ".metadata"
		ch := c.client.Watch(ctx, metaName)
		for wresp := range ch {
			for _, ev := range wresp.Events {
				meta := new(Metadata)
				if err := json.Unmarshal(ev.Kv.Value, meta); err != nil {
					// TODO: error
					continue
				}
				// check in gray list?
				if len(meta.Gray) > 0 && SliceIndex(len(meta.Gray), func(i int) bool { return meta.Gray[i] == c.id }) == -1 {
					continue
				}
				res, err := c.client.Get(ctx, keyName)
				if err != nil {
					// TODO: error
					continue
				}
				meta.Version = res.Kvs[0].Version
				h(File{group, name, res.Kvs[0].Value, meta})
			}
		}
	}()
	return nil
}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}
