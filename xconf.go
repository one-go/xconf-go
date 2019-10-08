package xconf

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/etcd-io/etcd/clientv3"
)

var (
	// ErrNotFound is the error returned by file not exists
	ErrNotFound = errors.New("not found")
)

const (
	DefaultName      = "xconf"
	DefaultNamespace = "x"
)

type File struct {
	Group   string
	Name    string
	Content []byte
	Version int64
	Meta    Metadata
}

type Metadata struct {
	CreateTime int64  `json:"createTime"`
	UpdateTime int64  `json:"updateTime"`
	Gray       string `json:"gray"`
}

type Xconf struct {
	id       string
	prefix   string
	cacheDir string
	client   *clientv3.Client
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
	if opt.CacheDir == "" {
		opt.CacheDir = "/var/tmp"
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
		prefix:   "/" + DefaultName + "/" + opt.Namespace + "/",
		client:   cli,
		id:       opt.ID,
		cacheDir: filepath.Join(opt.CacheDir, DefaultName, opt.Namespace),
	}
	return xconf
}

type OnChange func(*File) error

func (c *Xconf) GetConfig(ctx context.Context, group, name string) ([]byte, error) {
	data, err := c.ReadCache(group, name)
	if err == nil {
		return data, err
	}
	if data, _, err = c.Get(ctx, c.Key(group, name)); err != nil {
		return nil, err
	}
	if err = c.WriteCache(group, name, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Xconf) Get(ctx context.Context, key string) ([]byte, int64, error) {
	res, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, 0, err
	}
	if res.Count == 0 {
		return nil, 0, ErrNotFound
	}
	return res.Kvs[0].Value, res.Kvs[0].Version, nil
}

func (c *Xconf) CreateFile(ctx context.Context, f *File) error {
	name := c.Key(f.Group, f.Name)
	if f.Meta.CreateTime == 0 {
		f.Meta.CreateTime = time.Now().Unix()
	}
	meta, err := json.Marshal(&f.Meta)
	if err != nil {
		return err
	}
	if _, err := c.client.Put(ctx, name, string(f.Content)); err != nil {
		return err
	}
	if _, err = c.client.Put(ctx, c.Meta(name), string(meta)); err != nil {
		return err
	}
	return nil
}

func (c *Xconf) UpdateFile(ctx context.Context, f *File) error {
	name := c.Key(f.Group, f.Name)
	meta, err := json.Marshal(&f.Meta)
	if err != nil {
		return err
	}
	if _, err := c.client.Put(ctx, name, string(f.Content)); err != nil {
		return err
	}
	if _, err = c.client.Put(ctx, c.Meta(name), string(meta)); err != nil {
		return err
	}
	return nil
}

func (c *Xconf) DeleteFile(ctx context.Context, group, name string) error {
	fname := c.Key(group, name)
	if _, err := c.client.Delete(ctx, fname); err != nil {
		return err
	}
	if _, err := c.client.Delete(ctx, c.Meta(fname)); err != nil {
		return err
	}
	return nil
}

func (c *Xconf) ReadCache(group, name string) ([]byte, error) {
	fname := filepath.Join(c.cacheDir, group, name)
	return ioutil.ReadFile(fname)
}

func (c *Xconf) WriteCache(group, name string, content []byte) error {
	groupDir := filepath.Join(c.cacheDir, group)
	if err := os.MkdirAll(groupDir, os.ModePerm); err != nil {
		return err
	}
	fname := filepath.Join(groupDir, name)
	return ioutil.WriteFile(fname, content, 0666)
}

func (c *Xconf) Key(group, name string) string {
	return c.prefix + group + "/" + name
}

func (c *Xconf) Meta(key string) string {
	return key + ".metadata"
}

func (c *Xconf) Watch(ctx context.Context, group, name string, h OnChange) error {
	go func() {
		keyName := c.Key(group, name)
		metaName := c.Meta(keyName)
		ch := c.client.Watch(ctx, metaName)
		file := &File{
			Group: group,
			Name:  name,
		}
		for wresp := range ch {
			for _, ev := range wresp.Events {
				meta := Metadata{}
				if err := json.Unmarshal(ev.Kv.Value, &meta); err != nil {
					// TODO: error
					continue
				}
				// check in gray list?
				if meta.Gray != "" && !c.CheckGray(meta.Gray) {
					// not in gray
					continue
				}
				res, err := c.client.Get(ctx, keyName)
				if err != nil {
					// TODO: error
					continue
				}
				file.Meta = meta
				file.Content = res.Kvs[0].Value
				file.Version = res.Kvs[0].Version
				c.WriteCache(group, name, file.Content)

				h(file)
			}
		}
	}()
	return nil
}

func (c *Xconf) CheckGray(gray string) bool {
	return strings.Index(gray, c.id) != -1
}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}
