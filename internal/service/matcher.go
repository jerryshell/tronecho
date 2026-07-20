package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/jerryshell/tronecho/internal/store"
)

// Matcher 是内存精确地址匹配器：Badger 持久层，内存 map 查询层。
type Matcher struct {
	mu    sync.RWMutex
	addrs map[string]store.AddrMeta
	st    *store.Store
}

func NewMatcher(st *store.Store) (*Matcher, error) {
	all, err := st.LoadAllAddrs()
	if err != nil {
		return nil, fmt.Errorf("load addresses: %w", err)
	}
	return &Matcher{addrs: all, st: st}, nil
}

// Exists 仅命中 enabled 的地址。
func (mt *Matcher) Exists(addr string) bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	meta, ok := mt.addrs[addr]
	return ok && meta.Enabled
}

func (mt *Matcher) Get(addr string) (*store.AddrMeta, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	meta, ok := mt.addrs[addr]
	if !ok {
		return nil, false
	}
	cp := meta
	return &cp, true
}

// Add 写穿：先 Badger 后内存。重复 add 保留原 created_at（upsert）。
func (mt *Matcher) Add(addr, label string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	meta := store.AddrMeta{Label: label, Enabled: true, CreatedAt: time.Now().Unix()}
	if old, ok := mt.addrs[addr]; ok {
		meta.CreatedAt = old.CreatedAt
	}
	if err := mt.st.PutAddr(addr, meta); err != nil {
		return err
	}
	mt.addrs[addr] = meta
	return nil
}

func (mt *Matcher) Remove(addr string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	if err := mt.st.DeleteAddr(addr); err != nil {
		return err
	}
	delete(mt.addrs, addr)
	return nil
}

// SetEnabled 禁用/启用，未注册返回 false。
func (mt *Matcher) SetEnabled(addr string, enabled bool) (bool, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	meta, ok := mt.addrs[addr]
	if !ok {
		return false, nil
	}
	meta.Enabled = enabled
	if err := mt.st.PutAddr(addr, meta); err != nil {
		return false, err
	}
	mt.addrs[addr] = meta
	return true, nil
}

func (mt *Matcher) Count() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.addrs)
}

// All 返回全部地址快照（只读用途，list 分页）。
func (mt *Matcher) All() map[string]store.AddrMeta {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	r := make(map[string]store.AddrMeta, len(mt.addrs))
	for k, v := range mt.addrs {
		r[k] = v
	}
	return r
}
