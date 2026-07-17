import { type NatsConnection, type JetStreamClient, JSONCodec, AckPolicy } from "nats";
import {
  getCustomerByAddress,
  updateCustomerBalance,
  createDeposit,
  getDepositByEventId,
  isUsdtAsset,
} from "./storage";
import { logger } from "./logger";

interface TronechoTransfer {
  v: number;
  id: string;
  chain: string;
  blockNumber: number;
  blockHash: string;
  txHash: string;
  logIndex: number;
  from: string;
  to: string;
  asset: string;
  symbol: string;
  decimals: number;
  amount: string;
  fee: string;
  blockTime: number;
  direction: string;
  label?: string;
}

const jc = JSONCodec();

let js: JetStreamClient | null = null;
let nc: NatsConnection | null = null;
let running = false;

export async function startNatsSubscription(conn: NatsConnection, prefix: string) {
  if (running) {
    logger.warn("NATS 订阅已在运行");
    return;
  }

  nc = conn;
  js = conn.jetstream();
  running = true;

  logger.info("开始消费 JetStream 事件");
  consumeEvents(prefix);
}

async function processMessage(msg: any): Promise<void> {
  try {
    const event = jc.decode(msg.data) as TronechoTransfer;
    await processTransferEvent(event);
    msg.ack();
  } catch (error) {
    logger.error("处理事件失败", { error: String(error) });
    msg.nak();
  }
}

async function consumeEvents(prefix: string) {
  if (!js || !nc || !running) return;

  try {
    let consumer;
    try {
      consumer = await js.consumers.get(prefix, `${prefix}-user-deposit`);
    } catch {
      const jsm = await nc.jetstreamManager();
      await jsm.consumers.add(prefix, {
        durable_name: `${prefix}-user-deposit`,
        ack_policy: AckPolicy.Explicit,
      });
      consumer = await js.consumers.get(prefix, `${prefix}-user-deposit`);
    }
    logger.info("JetStream 消费者就绪", { stream: prefix });

    const messages = await consumer.consume({
      max_messages: 100,
      expires: 5000,
    });

    for await (const msg of messages) {
      if (!running) break;
      await processMessage(msg);
    }
  } catch (error) {
    logger.error("JetStream 消费错误", { error: String(error) });
    if (running) {
      setTimeout(() => consumeEvents(prefix), 5000);
    }
  }
}

function isValidTransfer(event: TronechoTransfer): boolean {
  return isUsdtAsset(event.asset) && event.direction === "in";
}

async function processTransferEvent(event: TronechoTransfer) {
  logger.debug("收到事件", {
    id: event.id,
    asset: event.asset,
    direction: event.direction,
    to: event.to,
  });

  if (!isValidTransfer(event)) return;

  logger.info("收到 USDT 充值事件", {
    eventId: event.id,
    from: event.from,
    to: event.to,
    amount: event.amount,
  });

  const existing = await getDepositByEventId(event.id);
  if (existing) {
    logger.warn("重复事件，跳过", { eventId: event.id });
    return;
  }

  const customer = await getCustomerByAddress(event.to);
  if (!customer) {
    logger.warn("未找到用户，跳过充值", { address: event.to });
    return;
  }

  const amountUsdt = Number(event.amount) / Math.pow(10, event.decimals);
  const feeAmount = amountUsdt * customer.feeRate;
  const creditAmount = amountUsdt - feeAmount;

  logger.info("处理充值", {
    customerId: customer.id,
    amountUsdt,
    feeRate: customer.feeRate,
    feeAmount,
    creditAmount,
  });

  await createDeposit({
    customerId: customer.id,
    eventId: event.id,
    txHash: event.txHash,
    logIndex: event.logIndex,
    asset: event.asset,
    symbol: event.symbol,
    decimals: event.decimals,
    amountRaw: event.amount,
    amountUsdt,
    feeRate: customer.feeRate,
    feeAmount,
    creditAmount,
    blockNumber: event.blockNumber,
    blockHash: event.blockHash,
    blockTime: event.blockTime,
    from: event.from,
    to: event.to,
    direction: event.direction,
    label: customer.label,
  });

  await updateCustomerBalance(customer.id, creditAmount);

  logger.info("充值完成", {
    customerId: customer.id,
    creditAmount,
    newBalance: customer.balance + creditAmount,
  });
}

export async function stopNatsSubscription() {
  running = false;
}
