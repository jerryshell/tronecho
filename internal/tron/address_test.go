package tron

import "testing"

func TestHexToBase58(t *testing.T) {
	// 真实 Nile 地址（从 fixture block_mixed.json 中的 to_address 提取）
	hex41 := "416089190d5842b7539f790e9689316a0bba9af4e1"
	got := HexToBase58(hex41)
	if len(got) < 30 || got[0] != 'T' {
		t.Errorf("HexToBase58(%s) = %q, want T... base58", hex41, got)
	}
	// round-trip
	hexBack, err := Base58ToHex(got)
	if err != nil {
		t.Fatalf("Base58ToHex(%s): %v", got, err)
	}
	if hexBack != hex41 {
		t.Errorf("round-trip failed: %s -> %s -> %s", hex41, got, hexBack)
	}
}

func TestNormalizeAddress(t *testing.T) {
	// base58 pass-through — 从 fixture 中提取的真实地址
	hex41 := "416089190d5842b7539f790e9689316a0bba9af4e1"
	b58 := HexToBase58(hex41)
	addr, err := NormalizeAddress(b58)
	if err != nil {
		t.Fatal(err)
	}
	if addr != b58 {
		t.Errorf("pass-through failed: %s -> %s", b58, addr)
	}

	// hex 41 prefix
	addr2, err := NormalizeAddress(hex41)
	if err != nil {
		t.Fatal(err)
	}
	if addr2 != b58 {
		t.Errorf("hex->base58 failed: got %q, want %q", addr2, b58)
	}

	// empty
	_, err = NormalizeAddress("")
	if err == nil {
		t.Error("expected error for empty address")
	}

	// invalid
	_, err = NormalizeAddress("not_an_address")
	if err == nil {
		t.Error("expected error for garbage address")
	}
}

func TestBase58CheckEncode(t *testing.T) {
	// round-trip via HexToBase58 / Base58ToHex
	hexStr := "41a0b0c0d0e0f00102030405060708090a0b0c0d0e"
	b58 := HexToBase58(hexStr)
	if len(b58) < 20 {
		t.Errorf("encoded too short: %q", b58)
	}
	hexBack, err := Base58ToHex(b58)
	if err != nil {
		t.Fatalf("Base58ToHex(%s): %v", b58, err)
	}
	if hexBack != hexStr {
		t.Errorf("round-trip failed: %s -> %s -> %s", hexStr, b58, hexBack)
	}
}
