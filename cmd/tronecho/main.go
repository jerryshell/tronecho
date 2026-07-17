package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jerryshell/tronecho/internal/config"
	"github.com/jerryshell/tronecho/internal/service"
	"github.com/jerryshell/tronecho/internal/store"
	"github.com/jerryshell/tronecho/internal/tron"
	"github.com/nats-io/nats.go"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "addr" {
		os.Exit(addrSubcommand(os.Args[2:]))
		return
	}

	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	if cfg.Chain.ConfirmationDepth < 19 {
		logger.Warn("confirmation_depth < 19: accepting reorg risk",
			"depth", cfg.Chain.ConfirmationDepth)
	}

	// --- store ---
	st, err := store.Open(cfg.Store.BadgerPath, cfg.Store.VlogGCInterval, logger)
	if err != nil {
		logger.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// --- matcher ---
	matcher, err := service.NewMatcher(st)
	if err != nil {
		logger.Error("new matcher", "err", err)
		os.Exit(1)
	}
	logger.Info("matcher loaded", "addresses", matcher.Count())

	// --- tron client ---
	client, err := tron.NewClient(
		cfg.Chain.RPCUrls, cfg.Chain.APIKey,
		cfg.Chain.RPS, cfg.Chain.RequestTimeout,
		cfg.Chain.NodeFailThreshold, logger,
	)
	if err != nil {
		logger.Error("new tron client", "err", err)
		os.Exit(1)
	}

	// --- emitter ---
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	em, err := service.NewEmitter(ctx,
		cfg.NATS.URL, cfg.NATS.Stream(),
		cfg.NATS.EventSubject(), cfg.NATS.AlertSubject(), logger,
	)
	if err != nil {
		logger.Error("new emitter", "err", err)
		os.Exit(1)
	}
	defer em.Drain()

	// --- resolver ---
	resolver := service.NewResolver(client, st, *cfg.Chain.ResolveAssets, logger)

	// --- worker ---
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	worker := service.NewWorker(ctx,
		&service.ChainConfig{
			StartBlock:        cfg.Chain.StartBlock,
			PollInterval:      cfg.Chain.PollInterval,
			CatchupBatch:      cfg.Chain.CatchupBatch,
			FailedMaxAttempts: cfg.Chain.FailedMaxAttempts,
		},
		client, st, matcher, resolver, em, logger,
	)

	// --- addr API ---
	api := newAddrAPI(em.Conn(), matcher, worker, cfg.NATS.APISubjectPrefix(), logger)
	if err := api.start(); err != nil {
		logger.Error("addr api start", "err", err)
		os.Exit(1)
	}
	defer api.drain()

	// --- graceful shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutting down...")
		worker.Stop()
		cancel()
	}()

	logger.Info("tronecho started",
		"rpc", client.ActiveNode(),
		"stream", cfg.NATS.Stream(),
		"event", cfg.NATS.EventSubject(),
		"addresses", matcher.Count(),
	)

	if err := worker.Run(); err != nil {
		logger.Error("worker exited", "err", err)
		os.Exit(1)
	}
}

// --- addr API (NATS request-reply) ---

type addrAPI struct {
	nc      *nats.Conn
	matcher *service.Matcher
	worker  *service.Worker
	prefix  string
	logger  *slog.Logger
	subs    []*nats.Subscription
}

func newAddrAPI(nc *nats.Conn, matcher *service.Matcher, worker *service.Worker, prefix string, logger *slog.Logger) *addrAPI {
	return &addrAPI{nc: nc, matcher: matcher, worker: worker, prefix: prefix, logger: logger}
}

func (a *addrAPI) start() error {
	handlers := map[string]func(*nats.Msg){
		".addr.v1.add":        a.handleAdd,
		".addr.v1.batchAdd":   a.handleBatchAdd,
		".addr.v1.remove":     a.handleRemove,
		".addr.v1.setEnabled": a.handleSetEnabled,
		".addr.v1.get":        a.handleGet,
		".addr.v1.list":       a.handleList,
		".status.v1.get":      a.handleStatus,
	}
	for suffix, handler := range handlers {
		subj := a.prefix + suffix
		sub, err := a.nc.Subscribe(subj, handler)
		if err != nil {
			return fmt.Errorf("subscribe %s: %w", subj, err)
		}
		a.subs = append(a.subs, sub)
		a.logger.Info("subscribed", "subject", subj)
	}
	return nil
}

func (a *addrAPI) drain() {
	for _, sub := range a.subs {
		sub.Drain()
	}
}

func (a *addrAPI) handleAdd(m *nats.Msg) {
	var req struct {
		Address string `json:"address"`
		Label   string `json:"label"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	addr, err := tron.NormalizeAddress(req.Address)
	if err != nil {
		respond(m, errReply("INVALID_ADDRESS", err.Error()))
		return
	}
	if err := a.matcher.Add(addr, req.Label); err != nil {
		respond(m, errReply("INTERNAL", err.Error()))
		return
	}
	respond(m, okReply(nil))
	a.logger.Info("addr added", "address", addr, "label", req.Label)
}

func (a *addrAPI) handleBatchAdd(m *nats.Msg) {
	var req struct {
		Items []struct {
			Address string `json:"address"`
			Label   string `json:"label"`
		} `json:"items"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	if len(req.Items) > 1000 {
		respond(m, errReply("PAYLOAD_TOO_LARGE", "max 1000 items"))
		return
	}
	type result struct {
		Address string `json:"address"`
		OK      bool   `json:"ok"`
		Code    string `json:"code,omitempty"`
	}
	results := make([]result, len(req.Items))
	for i, item := range req.Items {
		addr, err := tron.NormalizeAddress(item.Address)
		if err != nil {
			results[i] = result{Address: item.Address, OK: false, Code: "INVALID_ADDRESS"}
			continue
		}
		if err := a.matcher.Add(addr, item.Label); err != nil {
			results[i] = result{Address: addr, OK: false, Code: "INTERNAL"}
			continue
		}
		results[i] = result{Address: addr, OK: true}
	}
	respond(m, okReply(map[string]any{"results": results}))
}

func (a *addrAPI) handleRemove(m *nats.Msg) {
	var req struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	addr, err := tron.NormalizeAddress(req.Address)
	if err != nil {
		respond(m, errReply("INVALID_ADDRESS", err.Error()))
		return
	}
	if err := a.matcher.Remove(addr); err != nil {
		respond(m, errReply("INTERNAL", err.Error()))
		return
	}
	respond(m, okReply(nil))
	a.logger.Info("addr removed", "address", addr)
}

func (a *addrAPI) handleSetEnabled(m *nats.Msg) {
	var req struct {
		Address string `json:"address"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	addr, err := tron.NormalizeAddress(req.Address)
	if err != nil {
		respond(m, errReply("INVALID_ADDRESS", err.Error()))
		return
	}
	ok, err := a.matcher.SetEnabled(addr, req.Enabled)
	if err != nil {
		respond(m, errReply("INTERNAL", err.Error()))
		return
	}
	if !ok {
		respond(m, errReply("NOT_FOUND", "address not registered"))
		return
	}
	respond(m, okReply(nil))
	a.logger.Info("addr setEnabled", "address", addr, "enabled", req.Enabled)
}

func (a *addrAPI) handleGet(m *nats.Msg) {
	var req struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	addr, err := tron.NormalizeAddress(req.Address)
	if err != nil {
		respond(m, errReply("INVALID_ADDRESS", err.Error()))
		return
	}
	meta, ok := a.matcher.Get(addr)
	if !ok {
		respond(m, okReply(map[string]any{"found": false}))
		return
	}
	respond(m, okReply(map[string]any{
		"found":      true,
		"label":      meta.Label,
		"enabled":    meta.Enabled,
		"created_at": meta.CreatedAt,
	}))
}

type addrEntry struct {
	Address   string `json:"address"`
	Label     string `json:"label"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

func paginateAddrs(all map[string]store.AddrMeta, cursor string, limit int) ([]addrEntry, string) {
	addrs := make([]string, 0, len(all))
	for addr := range all {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)

	if limit <= 0 {
		limit = 100
	}

	items := make([]addrEntry, 0, limit)
	nextCursor := ""
	for _, addr := range addrs {
		if addr <= cursor {
			continue
		}
		if len(items) >= limit {
			nextCursor = items[len(items)-1].Address
			break
		}
		meta := all[addr]
		items = append(items, addrEntry{
			Address:   addr,
			Label:     meta.Label,
			Enabled:   meta.Enabled,
			CreatedAt: meta.CreatedAt,
		})
	}
	return items, nextCursor
}

func (a *addrAPI) handleList(m *nats.Msg) {
	var req struct {
		Cursor string `json:"cursor"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		respond(m, errReply("BAD_REQUEST", err.Error()))
		return
	}
	items, nextCursor := paginateAddrs(a.matcher.All(), req.Cursor, req.Limit)
	respond(m, okReply(map[string]any{"items": items, "next_cursor": nextCursor}))
}

func (a *addrAPI) handleStatus(m *nats.Msg) {
	st := a.worker.Stats()
	respond(m, okReply(map[string]any{
		"chainHeight":   st.ChainHeight,
		"processedHeight": st.ProcessedHeight,
		"lag":           st.Lag,
		"failedBlocks":  st.FailedBlocks,
		"addresses":     st.Addresses,
		"assetsCached":  st.AssetsCached,
		"activeNode":    st.ActiveNode,
		"startedAt":     st.StartedAt.Unix(),
		"uptimeSec":     int(time.Since(st.StartedAt).Seconds()),
	}))
}

// --- envelope helpers ---

type envelope struct {
	OK    bool    `json:"ok"`
	Data  any     `json:"data,omitempty"`
	Error *apiErr `json:"error,omitempty"`
}

type apiErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func okReply(data any) []byte {
	b, _ := json.Marshal(envelope{OK: true, Data: data})
	return b
}

func errReply(code, msg string) []byte {
	b, _ := json.Marshal(envelope{OK: false, Error: &apiErr{Code: code, Message: msg}})
	return b
}

func respond(m *nats.Msg, data []byte) {
	m.Respond(data)
}

// --- addr subcommand ---

func addrSubcommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: tronecho addr import <file> | tronecho addr dump")
		return 1
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load("config.yaml")
	if err != nil {
		cfg = &config.Config{Store: config.StoreConfig{BadgerPath: "./data"}}
	}

	st, err := store.Open(cfg.Store.BadgerPath, 0, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		return 1
	}
	defer st.Close()

	switch args[0] {
	case "import":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: tronecho addr import <file>")
			return 1
		}
		return addrImport(st, args[1], logger)
	case "dump":
		return addrDump(st, logger)
	default:
		fmt.Fprintf(os.Stderr, "unknown addr subcommand: %s\n", args[0])
		return 1
	}
}

func addrImport(st *store.Store, path string, logger *slog.Logger) int {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file: %v\n", err)
		return 1
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	imported := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		addr := strings.TrimSpace(parts[0])
		label := ""
		if len(parts) > 1 {
			label = strings.TrimSpace(parts[1])
		}
		addr, err := tron.NormalizeAddress(addr)
		if err != nil {
			logger.Warn("skip invalid address", "raw", parts[0], "err", err)
			continue
		}
		if err := st.PutAddr(addr, store.AddrMeta{
			Label: label, Enabled: true, CreatedAt: time.Now().Unix(),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "put addr %s: %v\n", addr, err)
			return 1
		}
		imported++
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		return 1
	}
	logger.Info("import done", "count", imported)
	return 0
}

func addrDump(st *store.Store, logger *slog.Logger) int {
	addrs, err := st.LoadAllAddrs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load addrs: %v\n", err)
		return 1
	}
	for addr, meta := range addrs {
		enabled := "false"
		if meta.Enabled {
			enabled = "true"
		}
		fmt.Printf("%s,%s,%s\n", addr, enabled, meta.Label)
	}
	logger.Info("dump done", "count", len(addrs))
	return 0
}
