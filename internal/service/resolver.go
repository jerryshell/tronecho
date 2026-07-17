package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	"github.com/jerryshell/tronecho/internal/store"
	"github.com/jerryshell/tronecho/internal/tron"
	"golang.org/x/sync/singleflight"
)

// Resolver 懒解析资产元数据（symbol/decimals），永久缓存到 Badger。
type Resolver struct {
	client  *tron.Client
	store   *store.Store
	sf      singleflight.Group
	enabled bool
	logger  *slog.Logger
}

func NewResolver(client *tron.Client, store *store.Store, enabled bool, logger *slog.Logger) *Resolver {
	return &Resolver{client: client, store: store, enabled: enabled, logger: logger}
}

// AssetsCached 统计已缓存资产数（status 探针用）。
func (r *Resolver) AssetsCached() int {
	n, err := r.store.CountAssets()
	if err != nil {
		return 0
	}
	return n
}

// Resolve 返回资产元数据。永久缓存未解析的合约；RPC 临时错误不缓存。
// 返回 nil 时表示解析失败（事件应省略 symbol/decimals 照常发布）。
func (r *Resolver) Resolve(ctx context.Context, asset string) *tron.AssetMeta {
	if !r.enabled {
		return nil
	}
	if asset == tron.AssetTRX {
		return &tron.AssetMeta{Symbol: "TRX", Decimals: 6}
	}
	cacheKey := assetCacheKey(asset)
	if cacheKey == "" {
		return nil
	}
	// 1. 查缓存
	if cached, err := r.store.GetAsset(cacheKey); err == nil && cached != nil {
		if cached.Unresolved {
			return nil
		}
		return &tron.AssetMeta{Symbol: cached.Symbol, Decimals: cached.Decimals}
	}
	// 2. singleflight 解析（补块并发时常见）
	ch := r.sf.DoChan(cacheKey, func() (any, error) {
		return r.resolve(ctx, asset)
	})
	res := <-ch
	if res.Err != nil {
		// RPC 临时错误，不缓存
		r.logger.Debug("asset resolve failed (transient)", "asset", asset, "err", res.Err)
		return nil
	}
	meta := res.Val.(*tron.AssetMeta)
	return meta
}

func (r *Resolver) resolve(ctx context.Context, asset string) (*tron.AssetMeta, error) {
	var meta *tron.AssetMeta
	var err error
	if strings.HasPrefix(asset, tron.AssetTRC20Prefix) {
		meta, err = r.resolveTRC20(ctx, strings.TrimPrefix(asset, tron.AssetTRC20Prefix))
	} else if strings.HasPrefix(asset, tron.AssetTRC10Prefix) {
		meta, err = r.resolveTRC10(ctx, strings.TrimPrefix(asset, tron.AssetTRC10Prefix))
	} else {
		return nil, errors.New("unknown asset prefix")
	}
	if err != nil {
		// RPC 临时错误，不缓存
		return nil, err
	}
	// 永久缓存（含 unresolved 标记）
	cacheKey := assetCacheKey(asset)
	if cacheKey != "" {
		cacheMeta := store.AssetMeta{Unresolved: meta == nil}
		if meta != nil {
			cacheMeta.Symbol = meta.Symbol
			cacheMeta.Decimals = meta.Decimals
		}
		if werr := r.store.PutAsset(cacheKey, cacheMeta); werr != nil {
			r.logger.Warn("asset cache write failed", "err", werr)
		}
	}
	if meta == nil {
		return nil, errors.New("unresolved")
	}
	return meta, nil
}

func (r *Resolver) resolveTRC20(ctx context.Context, b58Addr string) (*tron.AssetMeta, error) {
	hex41, err := tron.Base58ToHex(b58Addr)
	if err != nil {
		return nil, fmt.Errorf("bad trc20 address %q: %w", b58Addr, err)
	}
	sym, err := r.callSymbol(ctx, hex41)
	if err != nil {
		r.logger.Debug("symbol() failed, treating as unresolved", "addr", b58Addr, "err", err)
		return nil, nil // 非 RPC 错误：合约无 symbol → 缓存 unresolved
	}
	dec := 0
	decHex, err := r.client.TriggerConstant(ctx, hex41, "decimals()")
	if err == nil {
		if n, parseErr := parseUint8(decHex); parseErr == nil {
			dec = n
		}
	}
	return &tron.AssetMeta{Symbol: sanitizeSymbol(sym), Decimals: dec}, nil
}

// callSymbol 兼容 string 和 bytes32 两种 ABI 返回编码。
func (r *Resolver) callSymbol(ctx context.Context, hex41 string) (string, error) {
	raw, err := r.client.TriggerConstant(ctx, hex41, "symbol()")
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("empty result")
	}
	// 标准 string ABI：第一字 == 0x20（offset），第二字 == length，然后 UTF-8 bytes
	if len(data) >= 64 {
		off := bytesToUint256(data[:32])
		if off == 32 && len(data) >= 64 {
			length := bytesToUint256(data[32:64])
			if length > 0 && length <= 128 && int(length) <= len(data)-64 {
				return string(data[64 : 64+length]), nil
			}
		}
	}
	// bytes32 老式编码：32 字节，trim 尾部 0x00
	end := len(data)
	if end > 32 {
		end = 32
	}
	for end > 0 && data[end-1] == 0 {
		end--
	}
	return string(data[:end]), nil
}

func (r *Resolver) resolveTRC10(ctx context.Context, assetID string) (*tron.AssetMeta, error) {
	issue, err := r.client.GetAssetIssue(ctx, assetID)
	if err != nil {
		r.logger.Debug("getassetissuebyid failed, treating as unresolved", "id", assetID, "err", err)
		return nil, nil
	}
	sym := hexToString(issue.Name)
	if sym == "" {
		sym = hexToString(issue.Abbr)
	}
	return &tron.AssetMeta{Symbol: sanitizeSymbol(sym), Decimals: issue.Precision}, nil
}

func sanitizeSymbol(s string) string {
	// 去控制字符、零宽字符，限长 32
	var b strings.Builder
	for _, r := range s {
		if r < 32 || unicode.IsControl(r) || r == '\u200b' || r == '\u200c' || r == '\u200d' || r == '\ufeff' {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= 32 {
			break
		}
	}
	return b.String()
}

func bytesToUint256(b []byte) uint64 {
	var n uint64
	for i := 0; i < 8 && i < len(b); i++ {
		n = n<<8 | uint64(b[i])
	}
	return n
}

func parseUint8(hexStr string) (int, error) {
	h := strings.TrimPrefix(hexStr, "0x")
	if len(h) == 0 {
		return 0, errors.New("empty")
	}
	// 取最后一字节
	if len(h) > 2 {
		h = h[len(h)-2:]
	}
	n, err := strconv.ParseUint(h, 16, 8)
	return int(n), err
}

func hexToString(hexStr string) string {
	h := strings.TrimSpace(hexStr)
	if h == "" {
		return ""
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return ""
	}
	end := len(b)
	for end > 0 && b[end-1] == 0 {
		end--
	}
	return string(b[:end])
}

func assetCacheKey(asset string) string {
	if strings.HasPrefix(asset, tron.AssetTRC20Prefix) {
		return "v1/asset/trc20/" + strings.TrimPrefix(asset, tron.AssetTRC20Prefix)
	}
	if strings.HasPrefix(asset, tron.AssetTRC10Prefix) {
		return "v1/asset/trc10/" + strings.TrimPrefix(asset, tron.AssetTRC10Prefix)
	}
	return ""
}
