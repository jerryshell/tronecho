package tron

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client 是 Tron HTTP API 客户端：限流 + 有序多节点 fallback。
type Client struct {
	nodes     []string
	apiKey    string
	http      *http.Client
	limiter   *rate.Limiter
	threshold int
	logger    *slog.Logger

	mu        sync.Mutex
	active    int
	failCount int
}

func NewClient(nodes []string, apiKey string, rps float64, timeout time.Duration, failThreshold int, logger *slog.Logger) (*Client, error) {
	if len(nodes) == 0 {
		return nil, errors.New("tron: no rpc nodes configured")
	}
	if failThreshold <= 0 {
		failThreshold = 5
	}
	return &Client{
		nodes:     nodes,
		apiKey:    apiKey,
		http:      &http.Client{Timeout: timeout},
		limiter:   rate.NewLimiter(rate.Limit(rps), max(1, int(rps))),
		threshold: failThreshold,
		logger:    logger,
	}, nil
}

// ActiveNode 返回当前节点 URL（用于 status 探针）。
func (c *Client) ActiveNode() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nodes[c.active]
}

func (c *Client) nodeCount() int { return len(c.nodes) }

// post 向当前节点发 POST；失败计入计数，超阈值切换下一节点后重试，全部失败返回错误。
func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < c.nodeCount(); attempt++ {
		node := c.ActiveNode()
		data, err := c.doOnce(ctx, node, path, payload)
		if err == nil {
			c.mu.Lock()
			c.failCount = 0
			c.mu.Unlock()
			return data, nil
		}
		lastErr = err
		c.logger.Warn("tron rpc failed", "node", node, "path", path, "err", err)
		c.registerFailure()
	}
	return nil, fmt.Errorf("tron rpc %s: all %d node(s) failed: %w", path, c.nodeCount(), lastErr)
}

func (c *Client) registerFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCount++
	if c.failCount >= c.threshold && c.nodeCount() > 1 {
		old := c.active
		c.active = (c.active + 1) % c.nodeCount()
		c.failCount = 0
		c.logger.Warn("tron rpc node switched",
			"from", c.nodes[old], "to", c.nodes[c.active])
	}
}

func (c *Client) doOnce(ctx context.Context, node, path string, payload []byte) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, node+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// --- 业务接口（固化数据全走 walletsolidity） ---

// SolidityLatestBlock 返回最新固化块高度。
func (c *Client) SolidityLatestBlock(ctx context.Context) (uint64, error) {
	data, err := c.post(ctx, "/walletsolidity/getnowblock", map[string]any{})
	if err != nil {
		return 0, err
	}
	var blk Block
	if err := json.Unmarshal(data, &blk); err != nil {
		return 0, fmt.Errorf("unmarshal nowblock: %w", err)
	}
	return uint64(blk.BlockHeader.RawData.Number), nil
}

// GetBlock 返回固化区块（含交易明细）。
func (c *Client) GetBlock(ctx context.Context, num uint64) (*Block, error) {
	data, err := c.post(ctx, "/walletsolidity/getblock", map[string]any{
		"id_or_num": strconv.FormatUint(num, 10),
		"detail":    true,
	})
	if err != nil {
		return nil, err
	}
	var blk Block
	if err := json.Unmarshal(data, &blk); err != nil {
		return nil, fmt.Errorf("unmarshal block %d: %w", num, err)
	}
	if blk.BlockID == "" {
		return nil, fmt.Errorf("block %d not found", num)
	}
	return &blk, nil
}

// GetTxnInfos 返回固化区块的全部收据；空块返回空切片。
func (c *Client) GetTxnInfos(ctx context.Context, num uint64) ([]*TxnInfo, error) {
	data, err := c.post(ctx, "/walletsolidity/gettransactioninfobyblocknum", map[string]any{
		"num": num,
	})
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	// 节点可能对空块返回错误对象而非空数组
	if trimmed[0] == '{' {
		var apiErr struct {
			Error string `json:"Error"`
		}
		if json.Unmarshal(trimmed, &apiErr) == nil && apiErr.Error != "" {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected txninfo response for block %d: %s", num, truncate(string(trimmed), 200))
	}
	var infos []*TxnInfo
	if err := json.Unmarshal(trimmed, &infos); err != nil {
		return nil, fmt.Errorf("unmarshal txninfo %d: %w", num, err)
	}
	return infos, nil
}

// TriggerConstant 调用合约只读方法，返回 ABI 编码的 hex 结果（第一个 return value）。
// 元数据查询走 fullnode（symbol/decimals 不可变，读最新状态无妨）。
func (c *Client) TriggerConstant(ctx context.Context, contractHex41, selector string) (string, error) {
	data, err := c.post(ctx, "/wallet/triggerconstantcontract", map[string]any{
		"owner_address":     "410000000000000000000000000000000000000000",
		"contract_address":  contractHex41,
		"function_selector": selector,
		"parameter":         "",
	})
	if err != nil {
		return "", err
	}
	var res TriggerConstantResult
	if err := json.Unmarshal(data, &res); err != nil {
		return "", fmt.Errorf("unmarshal trigger result: %w", err)
	}
	if res.Result.Code != "" {
		msg := res.Result.Message
		if raw, derr := hex.DecodeString(msg); derr == nil {
			msg = string(raw)
		}
		return "", fmt.Errorf("trigger %s on %s: %s (%s)", selector, contractHex41, res.Result.Code, msg)
	}
	if len(res.ConstantResult) == 0 || res.ConstantResult[0] == "" {
		return "", errors.New("empty constant_result")
	}
	return res.ConstantResult[0], nil
}

// GetAssetIssue 查询 TRC10 资产信息。
func (c *Client) GetAssetIssue(ctx context.Context, assetID string) (*AssetIssue, error) {
	data, err := c.post(ctx, "/wallet/getassetissuebyid", map[string]any{"value": assetID})
	if err != nil {
		return nil, err
	}
	var issue AssetIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("unmarshal asset issue: %w", err)
	}
	if issue.ID == "" {
		return nil, fmt.Errorf("asset %s not found", assetID)
	}
	return &issue, nil
}
