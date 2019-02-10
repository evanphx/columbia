package loader

import (
	"encoding/base64"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/blake2b"

	"github.com/evanphx/columbia/exec"
	"github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/wasm"
	"github.com/evanphx/columbia/wasm/validate"
	hclog "github.com/hashicorp/go-hclog"
	lru "github.com/hashicorp/golang-lru"
)

type LoaderCache struct {
	mu sync.RWMutex

	cache *lru.ARCCache
}

func NewLoaderCache() *LoaderCache {
	cache, err := lru.NewARC(100)
	if err != nil {
		panic(err)
	}

	return &LoaderCache{cache: cache}
}

func (l *LoaderCache) Lookup(key string) (*exec.PreparedModule, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	val, ok := l.cache.Get(key)
	if !ok {
		return nil, false
	}

	return val.(*exec.PreparedModule), true
}

func (l *LoaderCache) Set(key string, m *exec.PreparedModule) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cache.Add(key, m)
}

func NewLoader(cache *LoaderCache) *Loader {
	return &Loader{
		L:     hclog.L(),
		cache: cache,
	}
}

type Loader struct {
	L     hclog.Logger
	cache *LoaderCache
	env   *wasm.Module
}

func (l *Loader) LoadFile(path string, env *wasm.Module) (*Module, error) {
	l.env = env

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return l.Load(f, env)
}

func (l *Loader) Load(r io.ReadSeeker, env *wasm.Module) (*Module, error) {
	var cacheKey string

	if l.cache != nil {
		log.L.Debug("calculating module cache key")

		h, err := blake2b.New256(nil)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(h, r)
		if err != nil {
			return nil, err
		}

		cacheKey = base64.URLEncoding.EncodeToString(h.Sum(nil))

		log.L.Debug("looking for cached module", "key", cacheKey)

		_, err = r.Seek(0, os.SEEK_SET)
		if err != nil {
			return nil, err
		}

		if mod, ok := l.cache.Lookup(cacheKey); ok {
			return &Module{l, mod}, nil
		}
	}

	l.env = env

	m, err := wasm.ReadModule(r, l.importer)
	if err != nil {
		return nil, err
	}

	err = validate.VerifyModule(m)
	if err != nil {
		return nil, err
	}

	pm, err := exec.PrepareModule(m)
	if err != nil {
		return nil, err
	}

	if l.cache != nil {
		log.L.Debug("cached module", "key", cacheKey)
		l.cache.Set(cacheKey, pm)
	}

	return &Module{l, pm}, nil
}

func (l *Loader) importer(name string) (*wasm.Module, error) {
	if name == "env" {
		return l.env, nil
	}

	return nil, nil
}
