package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Key 布局，全部带 v1/ 前缀：
const (
	latestBlockKey   = "v1/tron/latest_block"
	failedPrefix     = "v1/tron/failed/"
	addrPrefix       = "v1/addr/"
	assetTRC20Prefix = "v1/asset/trc20/"
	assetTRC10Prefix = "v1/asset/trc10/"
)

var ErrNotFound = errors.New("store: key not found")

type Store struct {
	db     *badger.DB
	logger *slog.Logger
	done   chan struct{}
}

func Open(path string, vlogGCInterval time.Duration, logger *slog.Logger) (*Store, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}
	s := &Store{db: db, logger: logger, done: make(chan struct{})}
	if vlogGCInterval > 0 {
		go s.vlogGCLoop(vlogGCInterval)
	}
	return s, nil
}

func (s *Store) Close() error {
	close(s.done)
	return s.db.Close()
}

// vlogGCLoop 定期回收 value log，防止磁盘膨胀；ErrNoRewrite 属正常。
func (s *Store) vlogGCLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			for {
				err := s.db.RunValueLogGC(0.5)
				if err != nil {
					if err != badger.ErrNoRewrite {
						s.logger.Warn("badger vlog GC", "err", err)
					}
					break
				}
			}
		}
	}
}

// --- 进度 ---

func (s *Store) GetLatestBlock() (uint64, error) {
	var n uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(latestBlockKey))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			parsed, err := strconv.ParseUint(string(v), 10, 64)
			if err != nil {
				return err
			}
			n = parsed
			return nil
		})
	})
	if err == badger.ErrKeyNotFound {
		return 0, nil // 冷启动
	}
	return n, err
}

func (s *Store) SetLatestBlock(n uint64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(latestBlockKey), []byte(strconv.FormatUint(n, 10)))
	})
}

// --- 失败块 ---

// FailedBlock 是 v1/tron/failed/<n> 的值。
type FailedBlock struct {
	Number        uint64 `json:"-"`
	Attempts      int    `json:"attempts"`
	LastError     string `json:"last_error"`
	LastAttemptAt int64  `json:"last_attempt_at"`
}

func failedKey(n uint64) []byte {
	k := make([]byte, len(failedPrefix)+8)
	copy(k, failedPrefix)
	binary.BigEndian.PutUint64(k[len(failedPrefix):], n)
	return k
}

// MarkFailed 记录或累加一次失败，返回累计次数。
func (s *Store) MarkFailed(n uint64, errMsg string) (int, error) {
	attempts := 1
	err := s.db.Update(func(txn *badger.Txn) error {
		key := failedKey(n)
		if item, err := txn.Get(key); err == nil {
			var fb FailedBlock
			if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &fb) }); err == nil {
				attempts = fb.Attempts + 1
			}
		}
		data, _ := json.Marshal(FailedBlock{
			Attempts:      attempts,
			LastError:     errMsg,
			LastAttemptAt: time.Now().Unix(),
		})
		return txn.Set(key, data)
	})
	return attempts, err
}

// ListFailed 按块号升序返回失败块（二进制大端 key 保证数值序），最多 limit 条。
func (s *Store) ListFailed(limit int) ([]FailedBlock, error) {
	var out []FailedBlock
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(failedPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var fb FailedBlock
			if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &fb) }); err != nil {
				return err
			}
			fb.Number = binary.BigEndian.Uint64(item.Key()[len(failedPrefix):])
			out = append(out, fb)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
		return nil
	})
	return out, err
}

func (s *Store) RemoveFailed(n uint64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(failedKey(n))
	})
}

// --- 地址 ---

type AddrMeta struct {
	Label     string `json:"label"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Store) PutAddr(addr string, meta AddrMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(addrPrefix+addr), data)
	})
}

func (s *Store) GetAddr(addr string) (*AddrMeta, error) {
	var meta AddrMeta
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(addrPrefix + addr))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error { return json.Unmarshal(v, &meta) })
	})
	if err == badger.ErrKeyNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Store) DeleteAddr(addr string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(addrPrefix + addr))
	})
}

// LoadAllAddrs 启动时全量加载地址到内存。
func (s *Store) LoadAllAddrs() (map[string]AddrMeta, error) {
	out := make(map[string]AddrMeta)
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(addrPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var meta AddrMeta
			if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &meta) }); err != nil {
				return err
			}
			out[string(item.Key()[len(addrPrefix):])] = meta
		}
		return nil
	})
	return out, err
}

// ListAddrs 按地址字典序分页（cursor 为上一页最后一个地址，不含）。
func (s *Store) ListAddrs(cursor string, limit int) (items map[string]AddrMeta, nextCursor string, err error) {
	items = make(map[string]AddrMeta)
	if limit <= 0 {
		limit = 100
	}
	err = s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(addrPrefix)
		seek := []byte(addrPrefix + cursor)
		for it.Seek(seek); it.ValidForPrefix(prefix); it.Next() {
			addr := string(it.Item().Key()[len(addrPrefix):])
			if addr == cursor {
				continue // cursor 本身已在上一页返回
			}
			if len(items) >= limit {
				nextCursor = addr
				return nil
			}
			var meta AddrMeta
			if err := it.Item().Value(func(v []byte) error { return json.Unmarshal(v, &meta) }); err != nil {
				return err
			}
			items[addr] = meta
		}
		return nil
	})
	return items, nextCursor, err
}

// --- 资产元数据 ---

type AssetMeta struct {
	Symbol     string `json:"symbol,omitempty"`
	Decimals   int    `json:"decimals,omitempty"`
	Unresolved bool   `json:"unresolved,omitempty"`
}

// GetAsset 按缓存 key（如 "v1/asset/trc20/T..."）查询，未命中返回 (nil, nil)。
func (s *Store) GetAsset(cacheKey string) (*AssetMeta, error) {
	var meta AssetMeta
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error { return json.Unmarshal(v, &meta) })
	})
	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Store) PutAsset(cacheKey string, meta AssetMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(cacheKey), data)
	})
}

// CountAssets 统计已缓存资产数（status 探针用）。
func (s *Store) CountAssets() (int, error) {
	count := 0
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for _, prefix := range []string{assetTRC20Prefix, assetTRC10Prefix} {
			p := []byte(prefix)
			for it.Seek(p); it.ValidForPrefix(p); it.Next() {
				count++
			}
		}
		return nil
	})
	return count, err
}
