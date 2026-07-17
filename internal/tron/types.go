package tron

import "encoding/json"

type ContractType string

const (
	ContractTypeTransfer      ContractType = "TransferContract"
	ContractTypeTransferAsset ContractType = "TransferAssetContract"
	ContractTypeTriggerSmart  ContractType = "TriggerSmartContract"
)

type Block struct {
	BlockID     string      `json:"blockID"`
	BlockHeader BlockHeader `json:"block_header"`
	Txs         []Txn       `json:"transactions"`
}

type BlockHeader struct {
	RawData BlockRawData `json:"raw_data"`
}

type BlockRawData struct {
	Number     int64  `json:"number"`
	Timestamp  int64  `json:"timestamp"` // 毫秒
	ParentHash string `json:"parentHash"`
}

type Txn struct {
	TxID    string     `json:"txID"`
	RawData TxnRawData `json:"raw_data"`
	Ret     []TxnRet   `json:"ret"`
}

type TxnRawData struct {
	Contract  []Contract `json:"contract"`
	Timestamp int64      `json:"timestamp"`
}

type TxnRet struct {
	ContractRet string `json:"contractRet"`
}

// Successful 判断交易是否执行成功。ret 为空时按成功处理
// （部分节点对非合约交易不填 contractRet）。
func (t *Txn) Successful() bool {
	for _, r := range t.Ret {
		if r.ContractRet != "" && r.ContractRet != "SUCCESS" {
			return false
		}
	}
	return true
}

type Contract struct {
	Type      ContractType  `json:"type"`
	Parameter ContractParam `json:"parameter"`
}

type ContractParam struct {
	Value json.RawMessage `json:"value"`
}

type TransferContract struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
}

type TransferAssetContract struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	AssetName    string `json:"asset_name"` // 实际是 TRC10 asset ID（数字字符串）
	Amount       int64  `json:"amount"`
}

// TxnInfo 是 gettransactioninfobyblocknum 返回的收据。
type TxnInfo struct {
	ID             string  `json:"id"`
	Fee            int64   `json:"fee"`
	BlockNumber    int64   `json:"blockNumber"`
	BlockTimestamp int64   `json:"blockTimeStamp"` // 毫秒
	Result         string  `json:"result"`         // "FAILED" 表示失败
	Log            []Log   `json:"log"`
	EnergyFee      int64   `json:"energy_fee"`
	NetFee         int64   `json:"net_fee"`
	Receipt        Receipt `json:"receipt"`
}

type Receipt struct {
	Result             string `json:"result"` // "FAILED" / "OUT_OF_ENERGY" / "REVERT" 等
	EnergyFee          int64  `json:"energy_fee"`
	NetFee             int64  `json:"net_fee"`
	EnergyPenaltyTotal int64  `json:"energy_penalty_total"`
}

// Failed 判断收据对应的交易是否失败。失败的合约交易其 log 必须忽略。
func (ti *TxnInfo) Failed() bool {
	if ti == nil {
		return true
	}
	if ti.Result == "FAILED" {
		return true
	}
	switch ti.Receipt.Result {
	case "", "SUCCESS":
		return false
	default:
		// receipt.result 出现具体错误（OUT_OF_ENERGY / REVERT / ...）视为失败
		return true
	}
}

// TotalFeeSun 计算整笔交易总手续费（sun）。
func (ti *TxnInfo) TotalFeeSun() int64 {
	if ti == nil {
		return 0
	}
	total := ti.Fee + ti.EnergyFee + ti.NetFee
	if total == 0 {
		total = ti.Receipt.EnergyFee + ti.Receipt.NetFee
	}
	return total + ti.Receipt.EnergyPenaltyTotal
}

type Log struct {
	Address string   `json:"address"` // 40-hex，无 41 前缀
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// triggerconstantcontract 响应（只取需要的字段）
type TriggerConstantResult struct {
	Result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	ConstantResult []string `json:"constant_result"`
}

// getassetissuebyid 响应
type AssetIssue struct {
	ID        string `json:"id"`
	Name      string `json:"name"`      // hex 编码
	Abbr      string `json:"abbr"`      // hex 编码
	Precision int    `json:"precision"` // decimals
}
