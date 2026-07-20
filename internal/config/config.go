package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultNodeFailThreshold = 5
	defaultConfirmationDepth = 19
	defaultPollInterval      = 3 * time.Second
	defaultCatchupBatch      = 20
	defaultRequestTimeout    = 20 * time.Second
	defaultMaxRetries        = 3
	defaultFailedMaxAttempts = 10
	defaultResolveAssets     = true
	defaultVlogGCInterval    = 10 * time.Minute
	defaultRPS               = 10.0
	defaultBadgerPath        = "./data"
	defaultNATSPrefix        = "tronecho"
)

type Config struct {
	Chain ChainConfig `yaml:"chain"`
	Store StoreConfig `yaml:"store"`
	NATS  NATSConfig  `yaml:"nats"`
}

type ChainConfig struct {
	RPCUrls           []string      `yaml:"rpc_urls"`
	APIKey            string        `yaml:"api_key"`
	StartBlock        *uint64       `yaml:"start_block"`
	RPS               float64       `yaml:"rps"`
	ResolveAssets     *bool         `yaml:"resolve_assets"`
	NodeFailThreshold int           `yaml:"node_fail_threshold"`
	ConfirmationDepth int           `yaml:"confirmation_depth"`
	PollInterval      time.Duration `yaml:"poll_interval"`
	CatchupBatch      int           `yaml:"catchup_batch"`
	RequestTimeout    time.Duration `yaml:"request_timeout"`
	MaxRetries        int           `yaml:"max_retries"`
	FailedMaxAttempts int           `yaml:"failed_max_attempts"`
}

type StoreConfig struct {
	BadgerPath     string        `yaml:"badger_path"`
	VlogGCInterval time.Duration `yaml:"vlog_gc_interval"`
}

type NATSConfig struct {
	URL    string `yaml:"url"`
	Prefix string `yaml:"prefix"`
}

func (n *NATSConfig) Stream() string           { return n.Prefix }
func (n *NATSConfig) EventSubject() string     { return n.Prefix + ".event.transfer" }
func (n *NATSConfig) APISubjectPrefix() string { return n.Prefix }
func (n *NATSConfig) AlertSubject() string     { return n.Prefix + ".alert" }

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	expanded := os.ExpandEnv(string(raw))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)

	if len(cfg.Chain.RPCUrls) == 0 {
		return nil, fmt.Errorf("chain.rpc_urls is required")
	}
	if cfg.NATS.URL == "" {
		return nil, fmt.Errorf("nats.url is required")
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	c := &cfg.Chain

	c.NodeFailThreshold = intWithDefault(c.NodeFailThreshold, "TRONECHO_NODE_FAIL_THRESHOLD", defaultNodeFailThreshold)
	c.ConfirmationDepth = intWithDefault(c.ConfirmationDepth, "TRONECHO_CONFIRMATION_DEPTH", defaultConfirmationDepth)
	c.PollInterval = durationWithDefault(c.PollInterval, "TRONECHO_POLL_INTERVAL", defaultPollInterval)
	c.CatchupBatch = intWithDefault(c.CatchupBatch, "TRONECHO_CATCHUP_BATCH", defaultCatchupBatch)
	c.RequestTimeout = durationWithDefault(c.RequestTimeout, "TRONECHO_REQUEST_TIMEOUT", defaultRequestTimeout)
	c.MaxRetries = intWithDefault(c.MaxRetries, "TRONECHO_MAX_RETRIES", defaultMaxRetries)
	c.FailedMaxAttempts = intWithDefault(c.FailedMaxAttempts, "TRONECHO_FAILED_MAX_ATTEMPTS", defaultFailedMaxAttempts)

	if c.RPS <= 0 {
		c.RPS = floatWithDefault(0, "TRONECHO_RPS", defaultRPS)
	}

	if c.APIKey == "" {
		c.APIKey = os.Getenv("TRON_API_KEY")
	}

	if c.ResolveAssets == nil {
		v := defaultResolveAssets
		if s := os.Getenv("TRONECHO_RESOLVE_ASSETS"); s != "" {
			v = s == "true" || s == "1"
		}
		c.ResolveAssets = &v
	}

	if cfg.Store.BadgerPath == "" {
		cfg.Store.BadgerPath = defaultBadgerPath
	}
	cfg.Store.VlogGCInterval = durationWithDefault(cfg.Store.VlogGCInterval, "TRONECHO_VLOG_GC_INTERVAL", defaultVlogGCInterval)

	if cfg.NATS.Prefix == "" {
		cfg.NATS.Prefix = defaultNATSPrefix
	}
}

func intWithDefault(val int, envKey string, fallback int) int {
	if val != 0 {
		return val
	}
	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func floatWithDefault(val float64, envKey string, fallback float64) float64 {
	if val != 0 {
		return val
	}
	if v := os.Getenv(envKey); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func durationWithDefault(val time.Duration, envKey string, fallback time.Duration) time.Duration {
	if val != 0 {
		return val
	}
	if v := os.Getenv(envKey); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
