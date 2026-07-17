package tron

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
)

// HexToBase58 将 hex 地址（41 前缀 / 0x 前缀 / 40-hex）转为 base58（T...）。
// 无法转换时原样返回。
func HexToBase58(h string) string {
	cleaned := strings.TrimPrefix(strings.TrimSpace(h), "0x")
	switch len(cleaned) {
	case 40:
		cleaned = "41" + cleaned
	case 42:
		// 应已是 41 前缀
	default:
		return h
	}
	if !strings.HasPrefix(cleaned, "41") {
		return h
	}
	raw, err := hex.DecodeString(cleaned)
	if err != nil {
		return h
	}
	// raw = 21 bytes (0x41 + 20 bytes payload)
	return base58.CheckEncode(raw[1:], raw[0])
}

// Base58ToHex 将 base58（T...）转为 41 前缀 hex。非法地址返回错误。
func Base58ToHex(addr string) (string, error) {
	payload, version, err := base58.CheckDecode(addr)
	if err != nil {
		return "", fmt.Errorf("base58check decode %q: %w", addr, err)
	}
	if version != 0x41 {
		return "", fmt.Errorf("bad version 0x%02x, want 0x41", version)
	}
	return fmt.Sprintf("41%x", payload), nil
}

// NormalizeAddress 接受 base58（T...）或 hex（41.../0x41.../40-hex），
// 统一返回 base58。链上解析出的地址一律是 base58，注册侧必须归一。
func NormalizeAddress(addr string) (string, error) {
	s := strings.TrimSpace(addr)
	if s == "" {
		return "", errors.New("empty address")
	}
	if s[0] == 'T' || s[0] == 't' {
		if _, _, err := base58.CheckDecode(s); err != nil {
			return "", fmt.Errorf("invalid tron address %q: %w", addr, err)
		}
		return s, nil
	}
	if !IsHexAddress(s) {
		return "", fmt.Errorf("invalid tron address %q", addr)
	}
	return HexToBase58(s), nil
}

// IsHexAddress 判断是否为可转换的 Tron hex 地址形式。
func IsHexAddress(s string) bool {
	c := strings.TrimPrefix(s, "0x")
	if len(c) == 40 {
		_, err := hex.DecodeString(c)
		return err == nil
	}
	if len(c) == 42 && strings.HasPrefix(c, "41") {
		_, err := hex.DecodeString(c)
		return err == nil
	}
	return false
}
