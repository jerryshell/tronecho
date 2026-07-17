package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jerryshell/tronecho/internal/store"
	"github.com/jerryshell/tronecho/internal/tron"
	"golang.org/x/sync/errgroup"
)

// Stats 是 status 探针的数据源。
type Stats struct {
	ChainHeight   uint64
	ProcessedHeight uint64
	Lag           uint64
	FailedBlocks  int
	Addresses     int
	AssetsCached  int
	ActiveNode    string
	StartedAt     time.Time
}

type Worker struct {
	cfg      *ChainConfig
	client   *tron.Client
	store    *store.Store
	matcher  *Matcher
	resolver *Resolver
	em       *Emitter
	logger   *slog.Logger

	chainHeight   uint64
	processedHeight uint64
	failedCount   int
	startedAt     time.Time
	mu            sync.RWMutex // guards chain/processed/failed (status reader)

	ctx    context.Context
	cancel context.CancelFunc
}

// ChainConfig 从 config 解析后的链配置（避免 import 循环）。
type ChainConfig struct {
	StartBlock        *uint64
	PollInterval      time.Duration
	CatchupBatch      int
	FailedMaxAttempts int
}

func NewWorker(
	ctx context.Context,
	cfg *ChainConfig,
	client *tron.Client,
	st *store.Store,
	matcher *Matcher,
	resolver *Resolver,
	em *Emitter,
	logger *slog.Logger,
) *Worker {
	ctx, cancel := context.WithCancel(ctx)
	return &Worker{
		cfg:       cfg,
		client:    client,
		store:     st,
		matcher:   matcher,
		resolver:  resolver,
		em:        em,
		logger:    logger,
		startedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (w *Worker) Stop() { w.cancel() }

// Stats 返回当前状态快照（线程安全，供 status handler 读取）。
func (w *Worker) Stats() Stats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return Stats{
		ChainHeight:   w.chainHeight,
		ProcessedHeight: w.processedHeight,
		Lag:           w.chainHeight - w.processedHeight,
		FailedBlocks:  w.failedCount,
		Addresses:     w.matcher.Count(),
		AssetsCached:  w.resolver.AssetsCached(),
		ActiveNode:    w.client.ActiveNode(),
		StartedAt:     w.startedAt,
	}
}

// Run 主循环：实时 + 启动补块 + 失败重试。
func (w *Worker) Run() error {
	// 确定起始高度
	saved, err := w.store.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("get latest block: %w", err)
	}
	var start uint64
	if w.cfg.StartBlock != nil {
		start = *w.cfg.StartBlock
	}
	if saved > start {
		start = saved
	}
	if w.cfg.StartBlock == nil && saved == 0 {
		// 未配置 start_block 且无历史进度 — 从最新区块开始
		latest, err := w.client.SolidityLatestBlock(w.ctx)
		if err != nil {
			return fmt.Errorf("get latest block for initial start: %w", err)
		}
		start = latest
		w.logger.Info("no start_block configured, starting from latest block", "start", start)
	}
	w.processedHeight = start
	w.logger.Info("worker starting", "start", start)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("worker stopped")
			return nil
		case <-ticker.C:
			if err := w.tick(); err != nil {
				w.logger.Error("tick error", "err", err)
			}
		}
	}
}

func (w *Worker) tick() error {
	ctx := w.ctx

	// 1. 获取链上固化高度
	chainHeight, err := w.client.SolidityLatestBlock(ctx)
	if err != nil {
		w.logger.Error("solidity latest block", "err", err)
		w.mu.Lock()
		w.rpcFailure()
		w.mu.Unlock()
		return err
	}
	w.mu.Lock()
	w.chainHeight = chainHeight
	w.mu.Unlock()

	// 2. lag 计算
	lag := chainHeight - w.processedHeight
	w.logger.Info("tick", "chain", chainHeight, "processed", w.processedHeight, "lag", lag)

	if w.processedHeight >= chainHeight {
		return nil // 等新块
	}

	// 3. 批量追赶（lag > 0 时始终使用，最大化吞吐）
	w.catchup(ctx, chainHeight)

	// 4. 重试失败块
	w.retryFailed(ctx)
	return nil
}

func (w *Worker) processNext(ctx context.Context, target uint64) {
	n := w.processedHeight + 1
	if n > target {
		return
	}
	if ok := w.processBlock(ctx, n); ok {
		w.store.SetLatestBlock(n)
		w.processedHeight = n
	}
}

// catchup 并发拉取落后的区块（受 catchup_batch 约束）。
func (w *Worker) catchup(ctx context.Context, target uint64) {
	batch := w.cfg.CatchupBatch
	if batch <= 0 {
		batch = 20
	}
	start := w.processedHeight + 1
	end := start + uint64(batch) - 1
	if end > target {
		end = target
	}
	w.logger.Info("catchup", "start", start, "end", end, "count", end-start+1)

	// errgroup 限并发，逐块处理；失败块记 failed 但不阻塞进度
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(batch)
	for n := start; n <= end; n++ {
		n := n
		g.Go(func() error {
			w.processBlock(gctx, n)
			return nil
		})
	}
	g.Wait()
	// 进度推进到本批末尾（失败块已记 failed，下次重试）
	w.store.SetLatestBlock(end)
	w.processedHeight = end
}

func (w *Worker) processBlock(ctx context.Context, n uint64) bool {
	blk, err := w.client.GetBlock(ctx, n)
	if err != nil {
		w.logger.Error("get block", "block", n, "err", err)
		w.store.MarkFailed(n, err.Error())
		w.mu.Lock()
		w.failedCount++
		w.mu.Unlock()
		return false
	}
	infos, err := w.client.GetTxnInfos(ctx, n)
	if err != nil {
		w.logger.Error("get txn infos", "block", n, "err", err)
		w.store.MarkFailed(n, err.Error())
		w.mu.Lock()
		w.failedCount++
		w.mu.Unlock()
		return false
	}

	transfers := tron.ParseBlock(blk, infos)
	for i := range transfers {
		tr := &transfers[i]
		if !w.matcher.Exists(tr.To) {
			continue
		}
		// 填充 label
		if meta, ok := w.matcher.Get(tr.To); ok {
			tr.Label = meta.Label
		}
		// 解析资产元数据（仅命中时）
		if m := w.resolver.Resolve(ctx, tr.Asset); m != nil {
			tr.Symbol = m.Symbol
			tr.Decimals = m.Decimals
		}
		if err := w.em.PublishTransfer(ctx, tr); err != nil {
			w.logger.Error("publish transfer", "id", tr.ID, "err", err)
			w.store.MarkFailed(n, err.Error())
			w.mu.Lock()
			w.failedCount++
			w.mu.Unlock()
			return false
		}
		w.logger.Info("transfer published", "id", tr.ID, "to", tr.To, "asset", tr.Asset)
	}
	return true
}

func (w *Worker) retryFailed(ctx context.Context) {
	failed, err := w.store.ListFailed(50) // 每 tick 最多重试 50 个
	if err != nil || len(failed) == 0 {
		return
	}
	for _, fb := range failed {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if ok := w.processBlock(ctx, fb.Number); ok {
			w.store.RemoveFailed(fb.Number)
			w.mu.Lock()
			if w.failedCount > 0 {
				w.failedCount--
			}
			w.mu.Unlock()
			w.logger.Info("failed block recovered", "block", fb.Number)
		} else if fb.Attempts+1 >= w.cfg.FailedMaxAttempts {
			// 超限告警后丢弃
			w.em.PublishAlert(ctx, Alert{
				Type:      AlertFailedBlockDropped,
				Block:     fb.Number,
				Attempts:  fb.Attempts + 1,
				LastError: fb.LastError,
			})
			w.store.RemoveFailed(fb.Number)
			w.logger.Warn("failed block dropped after max attempts", "block", fb.Number, "attempts", fb.Attempts+1)
		}
	}
}

// rpcFailure 告警：连续失败 tick 数（由外层调用累计）。
func (w *Worker) rpcFailure() {
	// 简化：在 tick 中失败时直接记数，status 探针展示 failedBlocks
}
