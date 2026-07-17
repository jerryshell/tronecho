package tron

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

const TRC20TransferTopic = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

// ParseBlock 从区块与收据中提取全部转账（TRX / TRC10 / TRC20）。
//
// logIndex 语义（v3.1 修正）：
//   - TRC20：receipt.log[] 数组下标（允许稀疏，非 Transfer 日志占位）
//   - 原生 / TRC10：tx.RawData.Contract[] 数组下标
//
// symbol/decimals 不在此处填充，由 Resolver 在命中后解析。
func ParseBlock(blk *Block, infos []*TxnInfo) []Transfer {
	if blk == nil {
		return nil
	}
	infoByID := make(map[string]*TxnInfo, len(infos))
	for _, ti := range infos {
		if ti != nil && ti.ID != "" {
			infoByID[ti.ID] = ti
		}
	}

	var out []Transfer

	// --- TRC20：从收据 log 解析 ---
	for _, ti := range infos {
		if ti == nil || ti.Failed() {
			continue
		}
		fee := strconv.FormatInt(ti.TotalFeeSun(), 10)
		for idx := range ti.Log {
			tr, ok := parseTRC20Log(blk, ti, &ti.Log[idx], idx, fee)
			if ok {
				out = append(out, tr)
			}
		}
	}

	// --- 原生 TRX / TRC10：从区块交易解析 ---
	for i := range blk.Txs {
		tx := &blk.Txs[i]
		if !tx.Successful() {
			continue
		}
		ti := infoByID[tx.TxID]
		if ti != nil && ti.Failed() {
			continue
		}
		fee := "0"
		ts := blk.BlockHeader.RawData.Timestamp
		if ti != nil {
			fee = strconv.FormatInt(ti.TotalFeeSun(), 10)
			if ti.BlockTimestamp > 0 {
				ts = ti.BlockTimestamp
			}
		}
		for cidx := range tx.RawData.Contract {
			c := &tx.RawData.Contract[cidx]
			switch c.Type {
			case ContractTypeTransfer:
				var tc TransferContract
				if err := json.Unmarshal(c.Parameter.Value, &tc); err != nil {
					continue
				}
				out = append(out, newTransfer(blk, tx.TxID, cidx,
					HexToBase58(tc.OwnerAddress), HexToBase58(tc.ToAddress),
					AssetTRX, strconv.FormatInt(tc.Amount, 10), fee, ts))
			case ContractTypeTransferAsset:
				var ta TransferAssetContract
				if err := json.Unmarshal(c.Parameter.Value, &ta); err != nil {
					continue
				}
				out = append(out, newTransfer(blk, tx.TxID, cidx,
					HexToBase58(ta.OwnerAddress), HexToBase58(ta.ToAddress),
					AssetTRC10Prefix+ta.AssetName, strconv.FormatInt(ta.Amount, 10), fee, ts))
			}
		}
	}
	return out
}

// parseTRC20Log 解析单条 receipt log；非 Transfer 事件返回 ok=false。
func parseTRC20Log(blk *Block, ti *TxnInfo, lg *Log, logIndex int, fee string) (Transfer, bool) {
	if len(lg.Topics) < 3 {
		return Transfer{}, false
	}
	if strings.ToLower(strings.TrimPrefix(lg.Topics[0], "0x")) != TRC20TransferTopic {
		return Transfer{}, false
	}
	amount, ok := new(big.Int).SetString(strings.TrimPrefix(lg.Data, "0x"), 16)
	if !ok {
		amount = new(big.Int)
	}
	ts := ti.BlockTimestamp
	if ts <= 0 {
		ts = blk.BlockHeader.RawData.Timestamp
	}
	from := HexToBase58("41" + last40(lg.Topics[1]))
	to := HexToBase58("41" + last40(lg.Topics[2]))
	asset := AssetTRC20Prefix + HexToBase58(lg.Address)
	return newTransfer(blk, ti.ID, logIndex, from, to, asset, amount.String(), fee, ts), true
}

func newTransfer(blk *Block, txHash string, logIndex int, from, to, asset, amount, fee string, ts int64) Transfer {
	blockNum := uint64(blk.BlockHeader.RawData.Number)
	return Transfer{
		V:           1,
		ID:          EventID("tron", blockNum, txHash, logIndex),
		Chain:       "tron",
		BlockNumber: blockNum,
		BlockHash:   blk.BlockID,
		TxHash:      txHash,
		LogIndex:    logIndex,
		From:        from,
		To:          to,
		Asset:       asset,
		Amount:      amount,
		Fee:         fee,
		BlockTime:   ts,
		Direction:   DirectionIn,
	}
}

func last40(s string) string {
	s = strings.TrimPrefix(s, "0x")
	if len(s) <= 40 {
		return s
	}
	return s[len(s)-40:]
}
