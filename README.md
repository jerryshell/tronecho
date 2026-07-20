# TronEcho

Tron 区块链事件监听服务。实时解析 Tron 链上区块，筛选出你关注的地址的转账事件，通过 NATS JetStream 推送给下游消费者。轻量级、配置简单、易于部署。

## 设计理念

- **配置简单**：只需配置 RPC 和 NATS 两个地址，其余均有合理默认值，支持环境变量覆盖
- **容易部署**：单二进制文件，内置 BadgerDB 无需外部数据库，仅依赖 NATS
- **轻量级**：嵌入式存储，最小化外部依赖，资源占用低

## 功能特性

- **多资产解析**：同时解析 TRX（原生币）、TRC10 和 TRC20 代币转账，包括批量转账（multisend）
- **NATS JetStream 事件流**：持久化、可重放的事件流，支持 72 小时消息保留和精确去重
- **RPC 故障转移**：多节点有序切换，自动绕过故障节点，内置令牌桶限流
- **地址注册管理**：通过 NATS API 动态增删启停监控地址，支持游标分页列表
- **资产元数据解析**：自动调用链上合约获取代币 symbol 和 decimals，带缓存和负缓存
- **补块与断点续传**：启动时自动补全历史区块，进度持久化到 BadgerDB，重启不丢数据
- **告警通知**：失败块丢弃、RPC 节点不可用等事件通过 NATS 告警主题推送
- **零外部数据库**：内置 BadgerDB 嵌入式存储，无需额外部署数据库

## 架构概览

```
Tron RPC 节点 ──(HTTP)──> 轮询器 ──> 区块解析器 ──> 地址匹配器 ──> NATS JetStream ──> 下游消费者
                               │                    │
                               v                    v
                           BadgerDB            NATS API
                          (进度/地址/缓存)      (地址管理/状态查询)
```

1. 定时轮询 Tron 固化节点，拉取最新区块
2. 解析区块中的 TRX / TRC10 / TRC20 转账
3. 与地址注册表匹配，筛选出关注的地址
4. 匹配到的转账事件发布到 NATS JetStream
5. 同时提供 NATS request-reply API，用于管理地址和查询状态

## 环境要求

- [NATS Server](https://nats.io/download/) 2.x（需启用 JetStream）
- Tron RPC 节点（支持 TronGrid 或自建节点）

编译需要 Go 1.26+。

## 配置说明

详见 `config.yaml` 中的注释，所有选项均可通过 `TRONECHO_*` 环境变量覆盖（优先级：环境变量 > yaml > 默认值）。

## 快速开始

只需 3 步即可运行：

### 1. 启动 NATS（需启用 JetStream）

```bash
nats-server -js
```

### 2. 编译

```bash
go build -o tronecho ./cmd/tronecho/
```

### 3. 配置并启动

编辑 `config.yaml`，填入 RPC 节点和 NATS 地址，然后启动：

```bash
./tronecho -config config.yaml
```

所有配置项均有合理默认值，只需填 `chain.rpc_urls` 和 `nats.url` 即可运行。

## 示例应用

`examples/user-deposit/` 目录包含一个基于 Nuxt 4 的用户充值管理系统，演示了如何消费 TronEcho 的 JetStream 事件流：

- 订阅 USDT 充值事件
- 管理客户及其 Tron 地址
- 自动同步地址到 TronEcho
- 实时充值记录与手续费计算
- 仪表盘展示服务状态和监控地址

```bash
cd examples/user-deposit
bun install
bun run dev
```

## NATS API 参考

所有 API 使用 NATS request-reply 模式，请求和响应均为 JSON。

主题前缀默认为 `tronecho`，可通过 `nats.prefix` 配置修改，例如 `tronecho.addr.v1.add`。

响应信封：

```jsonc
// 成功；data 为空时省略，即 {"ok": true}
{"ok": true, "data": {...}}

// 失败
{"ok": false, "error": {"code": "ERROR_CODE", "message": "..."}}
```

### `addr.v1.add`

添加单个地址。

**请求：**

```jsonc
{ "address": "Txxx...", "label": "用户1" }
```

**成功响应：**

```jsonc
{ "ok": true }
```

可能错误码：`BAD_REQUEST` / `INVALID_ADDRESS` / `INTERNAL`

### `addr.v1.batchAdd`

批量添加地址，上限 1000 条。

**请求：**

```jsonc
{ "items": [{ "address": "Txxx...", "label": "..." }] }
```

**成功响应：**

```jsonc
{
  "ok": true,
  "data": {
    "results": [
      { "address": "Txxx...", "ok": true },
      { "address": "Txxx...", "ok": false, "code": "INVALID_ADDRESS" },
    ],
  },
}
```

单条失败不影响其他条目。可能错误码：`BAD_REQUEST` / `PAYLOAD_TOO_LARGE`

### `addr.v1.remove`

删除地址。

**请求：**

```jsonc
{ "address": "Txxx..." }
```

**成功响应：**

```jsonc
{ "ok": true }
```

可能错误码：`BAD_REQUEST` / `INVALID_ADDRESS` / `INTERNAL`

### `addr.v1.setEnabled`

启用或停用地址。

**请求：**

```jsonc
{ "address": "Txxx...", "enabled": true }
```

**成功响应：**

```jsonc
{ "ok": true }
```

可能错误码：`BAD_REQUEST` / `INVALID_ADDRESS` / `NOT_FOUND` / `INTERNAL`

### `addr.v1.get`

查询单个地址。

**请求：**

```jsonc
{ "address": "Txxx..." }
```

**成功响应（未注册）：**

```jsonc
{ "ok": true, "data": { "found": false } }
```

**成功响应（已注册）：**

```jsonc
{
  "ok": true,
  "data": {
    "found": true,
    "label": "用户1",
    "enabled": true,
    "created_at": 1750000000,
  },
}
```

可能错误码：`BAD_REQUEST` / `INVALID_ADDRESS`

### `addr.v1.list`

分页列出已注册地址。

**请求：**

```jsonc
{ "cursor": "", "limit": 100 }
```

`limit` 默认 100，传入 `<= 0` 时按 100 处理。

**成功响应：**

```jsonc
{
  "ok": true,
  "data": {
    "items": [
      { "address": "Txxx...", "label": "用户1", "enabled": true, "created_at": 1750000000 },
    ],
    "next_cursor": "Txxx...",
  },
}
```

`next_cursor` 为空字符串表示已到末页。

可能错误码：`BAD_REQUEST`

### `status.v1.get`

查询服务状态。

**请求：**

无参数。

**成功响应：**

```jsonc
{
  "ok": true,
  "data": {
    "chainHeight": 70000000,
    "processedHeight": 69999980,
    "lag": 20,
    "failedBlocks": 0,
    "addresses": 100,
    "assetsCached": 5,
    "activeNode": "https://nile.trongrid.io",
    "startedAt": 1750000000,
    "uptimeSec": 3600,
  },
}
```

### 错误码汇总

| 错误码              | 说明                   |
| ------------------- | ---------------------- |
| `BAD_REQUEST`       | 请求 JSON 解析失败     |
| `INVALID_ADDRESS`   | 地址格式非法           |
| `NOT_FOUND`         | 地址未注册             |
| `PAYLOAD_TOO_LARGE` | 批量添加超过 1000 条   |
| `INTERNAL`          | 内部存储或其他内部错误 |

## 事件格式

TronEcho 通过 NATS 发送两类事件：转账事件（JetStream）和告警事件（普通 pub）。

### 转账事件

- 主题：`{prefix}.event.transfer`，默认 `tronecho.event.transfer`
- 投递方式：JetStream Publish，带 `Nats-Msg-Id` 消息头做去重，幂等键 = 事件 `id`
- 保留策略：72 小时，去重窗口 24 小时

**JSON Schema（v1）：**

```jsonc
{
  "v": 1,
  "id": "tron:70000000:<txHash>:0",
  "chain": "tron",
  "blockNumber": 70000000,
  "blockHash": "0x...",
  "txHash": "0x...",
  "logIndex": 0,
  "from": "Txxx...",
  "to": "Tyyy...",
  "asset": "tron:trx",
  "symbol": "TRX",
  "decimals": 6,
  "amount": "1000000000",
  "fee": "1000000",
  "blockTime": 1750000000,
  "direction": "in",
  "label": "用户1",
}
```

字段说明：

| 字段          | 类型   | 说明                                                                              |
| ------------- | ------ | --------------------------------------------------------------------------------- |
| `v`           | int    | 事件 schema 版本，当前为 1                                                        |
| `id`          | string | 全局唯一事件 ID：`chain:blockNumber:txHash:logIndex`，重放/重解析结果稳定         |
| `chain`       | string | 链标识，当前固定为 `tron`                                                         |
| `blockNumber` | uint64 | 区块号                                                                            |
| `blockHash`   | string | 区块哈希                                                                          |
| `txHash`      | string | 交易哈希                                                                          |
| `logIndex`    | int    | 日志下标：TRC20 对应 `receipt.log[]`，原生/TRC10 对应 `contract[]` 中转账合约下标 |
| `from`        | string | 转出地址（Base58）                                                                |
| `to`          | string | 转入地址（Base58）                                                                |
| `asset`       | string | 资产规范标识：`tron:trx`、`tron:trc10/<assetID>`、`tron:trc20/<contract>`         |
| `symbol`      | string | 代币符号，未解析成功时省略                                                        |
| `decimals`    | int    | 代币精度，未解析成功时省略                                                        |
| `amount`      | string | 转账数量，以最小单位表示的整数字符串（防止大数溢出）                              |
| `fee`         | string | 整笔交易总手续费，单位 sun（整数字符串）                                          |
| `blockTime`   | int64  | 区块时间戳（秒）                                                                  |
| `direction`   | string | 相对于监控地址的方向：`in` 表示地址接收，`out` 表示地址转出                       |
| `label`       | string | 命中地址的注册 label，未注册地址触发的事件不会发送                                |

### 告警事件

- 主题：`{prefix}.alert`，默认 `tronecho.alert`
- 投递方式：普通 NATS pub

**JSON Schema：**

```jsonc
{"type": "failed_block_dropped", "block": 70000000, "attempts": 10, "last_error": "..."}
{"type": "rpc_unavailable", "since": 1750000000, "consecutive_failures": 5}
```

字段说明：

| 字段                   | 类型   | 说明                                                 |
| ---------------------- | ------ | ---------------------------------------------------- |
| `type`                 | string | 告警类型：`failed_block_dropped` / `rpc_unavailable` |
| `block`                | uint64 | 失败块号（仅 `failed_block_dropped`）                |
| `attempts`             | int    | 已重试次数（仅 `failed_block_dropped`）              |
| `last_error`           | string | 最后一次错误信息（仅 `failed_block_dropped`）        |
| `since`                | int64  | 告警开始时间戳（秒，仅 `rpc_unavailable`）           |
| `consecutive_failures` | int    | 连续失败节点数（仅 `rpc_unavailable`）               |

## CLI 命令

```bash
# 从 CSV 文件导入地址（格式：address,label）
./tronecho addr import addresses.csv

# 导出全部已注册地址
./tronecho addr dump
```

## 许可证

[MIT](LICENSE)
