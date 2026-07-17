package tron

import "fmt"

// 资产规范标识符前缀：tron:trx / tron:trc10/<assetID> / tron:trc20/<contract>
const (
	AssetTRX         = "tron:trx"
	AssetTRC10Prefix = "tron:trc10/"
	AssetTRC20Prefix = "tron:trc20/"
)

const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

// AssetMeta 是资产元数据（symbol/decimals），由 Resolver 解析后填充。
type AssetMeta struct {
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// Transfer 是发布到 JetStream 的转账事件（schema v1）。
// JSON 字段名即对外契约，修改必须升 v。
type Transfer struct {
	V           int    `json:"v"`
	ID          string `json:"id"`
	Chain       string `json:"chain"`
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	TxHash      string `json:"txHash"`
	LogIndex    int    `json:"logIndex"` // 原始数组下标：TRC20=receipt.log[]，原生/TRC10=contract[]
	From        string `json:"from"`
	To          string `json:"to"`
	Asset       string `json:"asset"`
	Symbol      string `json:"symbol,omitempty"`
	Decimals    int    `json:"decimals,omitempty"`
	Amount      string `json:"amount"` // 最小单位原始整数
	Fee         string `json:"fee"`    // sun，整笔 tx 总手续费
	BlockTime   int64  `json:"blockTime"`
	Direction   string `json:"direction"`
	Label       string `json:"label,omitempty"` // 命中地址的注册 label
}

// EventID 生成全局唯一事件 ID：chain:block:txHash:logIndex。
// 是链上数据的确定性函数，重放/重解析结果稳定。
func EventID(chain string, blockNumber uint64, txHash string, logIndex int) string {
	return fmt.Sprintf("%s:%d:%s:%d", chain, blockNumber, txHash, logIndex)
}
