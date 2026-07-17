import { connect, type NatsConnection, JSONCodec } from "nats";
import { listCustomers } from "./storage";
import { logger } from "./logger";

const jc = JSONCodec();

let nc: NatsConnection | null = null;
let prefix = "tronecho";

export function setPrefix(p: string) {
  prefix = p;
}

export async function getNatsConnection(natsUrl: string): Promise<NatsConnection> {
  if (nc && !nc.isClosed()) return nc;
  nc = await connect({ servers: natsUrl });
  return nc;
}

interface NatsResult {
  ok: boolean;
  error?: { code: string; message: string };
}

async function natsRequest(
  action: "add" | "remove",
  address: string,
  label?: string,
): Promise<boolean> {
  const conn = nc;
  if (!conn) {
    logger.error("NATS 未连接", { action, address });
    return false;
  }

  const subject = `${prefix}.addr.v1.${action}`;
  const payload = jc.encode({ address, label: label || "" });

  try {
    const resp = await conn.request(subject, payload, { timeout: 5000 });
    const result = jc.decode(resp.data) as NatsResult;

    if (result.ok) {
      logger.info(`地址${action === "add" ? "注册" : "移除"}成功`, { address, label });
      return true;
    }

    logger.error(`地址${action === "add" ? "注册" : "移除"}失败`, { address, error: result.error });
    return false;
  } catch (error) {
    logger.error(`地址操作请求超时`, { action, address, error: String(error) });
    return false;
  }
}

export async function registerAddress(address: string, label?: string): Promise<boolean> {
  return natsRequest("add", address, label);
}

export async function removeAddress(address: string): Promise<boolean> {
  return natsRequest("remove", address);
}

export interface TronechoAddress {
  address: string;
  label: string;
  enabled: boolean;
  created_at: number;
}

export async function listAddresses(): Promise<TronechoAddress[]> {
  const conn = nc;
  if (!conn) {
    logger.error("NATS 未连接，无法获取地址列表");
    return [];
  }

  try {
    const resp = await conn.request(`${prefix}.addr.v1.list`, jc.encode({}), { timeout: 5000 });
    const result = jc.decode(resp.data) as {
      ok: boolean;
      data?: { items: TronechoAddress[] };
    };

    if (result.ok && result.data?.items) {
      return result.data.items;
    }

    return [];
  } catch (error) {
    logger.error("获取地址列表超时", { error: String(error) });
    return [];
  }
}

export async function syncAllAddresses() {
  const customers = await listCustomers();
  logger.info("同步地址到 tronecho", { count: customers.length });

  let success = 0;
  let failed = 0;

  for (const customer of customers) {
    const ok = await registerAddress(customer.address, customer.label);
    if (ok) {
      success++;
    } else {
      failed++;
    }
  }

  logger.info("地址同步完成", { success, failed });
}
