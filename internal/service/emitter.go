package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jerryshell/tronecho/internal/tron"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Emitter 是 JetStream 事件发布 + 告警 pub。
type Emitter struct {
	nc           *nats.Conn
	js           jetstream.JetStream
	eventSubject string
	alertSubject string
	logger       *slog.Logger
}

// NewEmitter 初始化 NATS 连接 + JetStream stream。
func NewEmitter(ctx context.Context, url, streamName, eventSubject, alertSubject string, logger *slog.Logger) (*Emitter, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        streamName,
		Subjects:    []string{eventSubject},
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.LimitsPolicy,
		MaxMsgSize:  32 * 1024,
		MaxBytes:    512 * 1024 * 1024,
		MaxAge:      72 * time.Hour,
		Duplicates:  24 * time.Hour,
		Description: "TronEcho transfer events (schema v1)",
	})
	if err != nil {
		nc.Close()
		return nil, err
	}
	logger.Info("JetStream stream ready", "stream", streamName, "event_subject", eventSubject)
	return &Emitter{
		nc: nc, js: js,
		eventSubject: eventSubject,
		alertSubject: alertSubject,
		logger:       logger,
	}, nil
}

func (e *Emitter) Conn() *nats.Conn { return e.nc }

// PublishTransfer 发布转账事件，幂等键 = 事件 ID。
func (e *Emitter) PublishTransfer(ctx context.Context, tr *tron.Transfer) error {
	data, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	_, err = e.js.PublishMsg(ctx, &nats.Msg{
		Subject: e.eventSubject,
		Data:    data,
		Header:  nats.Header{"Nats-Msg-Id": {tr.ID}},
	})
	return err
}

// AlertType 告警类型
type AlertType string

const (
	AlertFailedBlockDropped AlertType = "failed_block_dropped"
	AlertRPCUnavailable     AlertType = "rpc_unavailable"
)

// Alert 告警事件
type Alert struct {
	Type                AlertType `json:"type"`
	Block               uint64    `json:"block,omitempty"`
	Attempts            int       `json:"attempts,omitempty"`
	LastError           string    `json:"last_error,omitempty"`
	Since               int64     `json:"since,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures,omitempty"`
}

// PublishAlert 发布告警事件。
func (e *Emitter) PublishAlert(ctx context.Context, alert Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	return e.nc.Publish(e.alertSubject, data)
}

// Drain 等待已发布消息送达后关闭连接。
func (e *Emitter) Drain() error {
	return e.nc.Drain()
}
