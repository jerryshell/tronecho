package tron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadBlock(t *testing.T, name string) *Block {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	var blk Block
	if err := json.Unmarshal(data, &blk); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	return &blk
}

func loadInfos(t *testing.T, name string) []*TxnInfo {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	var infos []*TxnInfo
	if err := json.Unmarshal(data, &infos); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	return infos
}

func TestParseBlock_Mixed(t *testing.T) {
	blk := loadBlock(t, "block_mixed.json")
	infos := loadInfos(t, "info_mixed.json")
	transfers := ParseBlock(blk, infos)

	// 块 69253346：5 TRX + 1 TRC20
	if len(transfers) != 6 {
		t.Fatalf("expected 6 transfers, got %d", len(transfers))
	}

	// 统计类型
	var trc20, native int
	for _, tr := range transfers {
		if tr.Asset == AssetTRX {
			native++
		} else if len(tr.Asset) > len(AssetTRC20Prefix) && tr.Asset[:len(AssetTRC20Prefix)] == AssetTRC20Prefix {
			trc20++
		}
	}
	if trc20 != 1 {
		t.Errorf("expected 1 TRC20, got %d", trc20)
	}
	if native != 5 {
		t.Errorf("expected 5 TRX, got %d", native)
	}

	// 验证每个事件 ID 唯一
	ids := make(map[string]bool)
	for _, tr := range transfers {
		if ids[tr.ID] {
			t.Errorf("duplicate ID: %s", tr.ID)
		}
		ids[tr.ID] = true
		// ID 格式：tron:block:txHash:logIndex
		if len(tr.ID) < 10 {
			t.Errorf("ID too short: %q", tr.ID)
		}
		if tr.Chain != "tron" {
			t.Errorf("chain=%q, want tron", tr.Chain)
		}
		if tr.V != 1 {
			t.Errorf("v=%d, want 1", tr.V)
		}
		if tr.Direction != DirectionIn {
			t.Errorf("direction=%q, want %q", tr.Direction, DirectionIn)
		}
	}
}

func TestParseBlock_Empty(t *testing.T) {
	blk := loadBlock(t, "block_empty.json")
	infos := loadInfos(t, "info_empty.json")
	transfers := ParseBlock(blk, infos)
	if len(transfers) != 0 {
		t.Fatalf("expected 0 transfers for empty block, got %d", len(transfers))
	}
}

func TestParseBlock_MultiContract(t *testing.T) {
	blk := loadBlock(t, "block_multicontract.json")
	infos := loadInfos(t, "info_multicontract.json")
	transfers := ParseBlock(blk, infos)

	if len(transfers) != 2 {
		t.Fatalf("expected 2 transfers (2 contracts), got %d", len(transfers))
	}
	// logIndex 应分别为 0 和 1（contract 数组下标）
	if transfers[0].LogIndex != 0 || transfers[1].LogIndex != 1 {
		t.Errorf("logIndex mismatch: %d, %d", transfers[0].LogIndex, transfers[1].LogIndex)
	}
	// 两个 ID 不同（关键：同 tx 多 contract 不被去重吞掉）
	if transfers[0].ID == transfers[1].ID {
		t.Errorf("IDs should differ: %s", transfers[0].ID)
	}
}

func TestParseBlock_MultiSend(t *testing.T) {
	blk := loadBlock(t, "block_multisend.json")
	infos := loadInfos(t, "info_multisend.json")
	transfers := ParseBlock(blk, infos)

	// multisend fixture：1 tx，3 Transfer log + 1 Approval log（应被跳过）
	// 所以 3 个 TRC20 转账
	if len(transfers) != 3 {
		t.Fatalf("expected 3 transfers (3 Transfer logs), got %d", len(transfers))
	}
	for _, tr := range transfers {
		if tr.Asset[:len(AssetTRC20Prefix)] != AssetTRC20Prefix {
			t.Errorf("expected TRC20 asset, got %s", tr.Asset)
		}
	}
	// logIndex 应为 0, 2, 3（下标 1 是 Approval，被跳过）
	if transfers[0].LogIndex != 0 {
		t.Errorf("first transfer logIndex=%d, want 0", transfers[0].LogIndex)
	}
	if transfers[1].LogIndex != 2 {
		t.Errorf("second transfer logIndex=%d, want 2", transfers[1].LogIndex)
	}
	if transfers[2].LogIndex != 3 {
		t.Errorf("third transfer logIndex=%d, want 3", transfers[2].LogIndex)
	}
	// 所有 ID 唯一
	ids := map[string]bool{}
	for _, tr := range transfers {
		if ids[tr.ID] {
			t.Errorf("duplicate ID: %s", tr.ID)
		}
		ids[tr.ID] = true
	}
}

func TestParseBlock_FailedTx(t *testing.T) {
	blk := loadBlock(t, "block_failed.json")
	infos := loadInfos(t, "info_failed.json")
	transfers := ParseBlock(blk, infos)

	// 失败 tx 的 log 必须被忽略
	if len(transfers) != 0 {
		t.Fatalf("expected 0 transfers from failed tx, got %d", len(transfers))
	}
}

func TestTxnSuccessful(t *testing.T) {
	tests := []struct {
		name   string
		ret    []TxnRet
		expect bool
	}{
		{"nil", nil, true},
		{"empty", []TxnRet{}, true},
		{"success", []TxnRet{{ContractRet: "SUCCESS"}}, true},
		{"failed", []TxnRet{{ContractRet: "FAILED"}}, false},
		{"out_of_energy", []TxnRet{{ContractRet: "OUT_OF_ENERGY"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Txn{Ret: tt.ret}
			if got := tx.Successful(); got != tt.expect {
				t.Errorf("Successful() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestTxnInfoFailed(t *testing.T) {
	tests := []struct {
		name   string
		result string
		recRes string
		expect bool
	}{
		{"ok", "", "", false},
		{"result_failed", "FAILED", "", true},
		{"receipt_failed", "", "FAILED", true},
		{"receipt_out_of_energy", "", "OUT_OF_ENERGY", true},
		{"receipt_success", "", "SUCCESS", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := &TxnInfo{Result: tt.result, Receipt: Receipt{Result: tt.recRes}}
			if got := ti.Failed(); got != tt.expect {
				t.Errorf("Failed() = %v, want %v", got, tt.expect)
			}
		})
	}
}
