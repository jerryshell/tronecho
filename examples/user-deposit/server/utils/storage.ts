import { logger } from "./logger";
import type { Customer, Deposit } from "../../shared/types/customer";

// USDT contract addresses
export const USDT_CONTRACTS = {
  nile: "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf",
  mainnet: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
} as const;

const CUSTOMER_PREFIX = "customer:";
const CUSTOMER_ADDR_PREFIX = "customer:addr:";
const DEPOSIT_PREFIX = "deposit:";
const DEPOSIT_CUSTOMER_PREFIX = "deposit:customer:";

export async function createCustomer(
  data: Pick<Customer, "address"> & Partial<Pick<Customer, "label" | "feeRate">>,
): Promise<Customer> {
  const storage = useStorage("customers");
  const id = crypto.randomUUID();
  const now = Date.now();
  const customer: Customer = {
    id,
    label: data.label || "",
    address: data.address,
    feeRate: data.feeRate ?? 0,
    balance: 0,
    createdAt: now,
    updatedAt: now,
  };

  await storage.setItem(`${CUSTOMER_PREFIX}${id}`, customer);
  await storage.setItem(`${CUSTOMER_ADDR_PREFIX}${data.address}`, id);

  logger.info("创建用户", { id, address: data.address, label: data.label });
  return customer;
}

export async function getCustomer(id: string): Promise<Customer | null> {
  const storage = useStorage("customers");
  return await storage.getItem<Customer>(`${CUSTOMER_PREFIX}${id}`);
}

export async function getCustomerByAddress(address: string): Promise<Customer | null> {
  const storage = useStorage("customers");
  const id = await storage.getItem<string>(`${CUSTOMER_ADDR_PREFIX}${address}`);
  if (!id) return null;
  return await getCustomer(id);
}

export async function listCustomers(): Promise<Customer[]> {
  const storage = useStorage("customers");
  const keys = await storage.getKeys(CUSTOMER_PREFIX);
  const customers: Customer[] = [];

  for (const key of keys) {
    if (key.startsWith(CUSTOMER_ADDR_PREFIX)) continue;
    const customer = await storage.getItem<Customer>(key);
    if (customer) customers.push(customer);
  }

  return customers.sort((a, b) => b.createdAt - a.createdAt);
}

export async function updateCustomer(
  id: string,
  data: Partial<Pick<Customer, "label" | "feeRate">>,
): Promise<Customer | null> {
  const storage = useStorage("customers");
  const customer = await getCustomer(id);
  if (!customer) return null;

  const updated: Customer = {
    ...customer,
    ...data,
    updatedAt: Date.now(),
  };

  await storage.setItem(`${CUSTOMER_PREFIX}${id}`, updated);
  logger.info("更新用户", { id, ...data });
  return updated;
}

export async function updateCustomerBalance(id: string, amount: number): Promise<Customer | null> {
  const storage = useStorage("customers");
  const customer = await getCustomer(id);
  if (!customer) return null;

  const updated: Customer = {
    ...customer,
    balance: customer.balance + amount,
    updatedAt: Date.now(),
  };

  await storage.setItem(`${CUSTOMER_PREFIX}${id}`, updated);
  logger.info("更新用户余额", { id, amount, newBalance: updated.balance });
  return updated;
}

export async function deleteCustomer(id: string): Promise<boolean> {
  const storage = useStorage("customers");
  const customer = await getCustomer(id);
  if (!customer) return false;

  await storage.removeItem(`${CUSTOMER_PREFIX}${id}`);
  await storage.removeItem(`${CUSTOMER_ADDR_PREFIX}${customer.address}`);

  // Also clean up deposits
  const depositStorage = useStorage("deposits");
  const depositIds =
    (await depositStorage.getItem<string[]>(`${DEPOSIT_CUSTOMER_PREFIX}${id}`)) || [];
  for (const depositId of depositIds) {
    await depositStorage.removeItem(`${DEPOSIT_PREFIX}${depositId}`);
  }
  await depositStorage.removeItem(`${DEPOSIT_CUSTOMER_PREFIX}${id}`);

  logger.info("删除用户", { id, address: customer.address });
  return true;
}

export async function createDeposit(data: Omit<Deposit, "id" | "createdAt">): Promise<Deposit> {
  const storage = useStorage("deposits");
  const id = crypto.randomUUID();
  const now = Date.now();

  const deposit: Deposit = {
    id,
    ...data,
    createdAt: now,
  };

  await storage.setItem(`${DEPOSIT_PREFIX}${id}`, deposit);

  // Add to customer's deposit list
  const key = `${DEPOSIT_CUSTOMER_PREFIX}${data.customerId}`;
  const existing = (await storage.getItem<string[]>(key)) || [];
  existing.push(id);
  await storage.setItem(key, existing);

  logger.info("创建充值记录", {
    id,
    customerId: data.customerId,
    txHash: data.txHash,
    amountUsdt: data.amountUsdt,
    creditAmount: data.creditAmount,
  });
  return deposit;
}

export async function getDeposit(id: string): Promise<Deposit | null> {
  const storage = useStorage("deposits");
  return await storage.getItem<Deposit>(`${DEPOSIT_PREFIX}${id}`);
}

export async function listDepositsByCustomer(customerId: string): Promise<Deposit[]> {
  const storage = useStorage("deposits");
  const key = `${DEPOSIT_CUSTOMER_PREFIX}${customerId}`;
  const ids = (await storage.getItem<string[]>(key)) || [];

  const deposits: Deposit[] = [];
  for (const id of ids) {
    const deposit = await getDeposit(id);
    if (deposit) deposits.push(deposit);
  }

  return deposits.sort((a, b) => b.createdAt - a.createdAt);
}

export async function getDepositByEventId(eventId: string): Promise<Deposit | null> {
  const storage = useStorage("deposits");
  const keys = await storage.getKeys(DEPOSIT_PREFIX);

  for (const key of keys) {
    const deposit = await storage.getItem<Deposit>(key);
    if (deposit?.eventId === eventId) return deposit;
  }

  return null;
}

export function isUsdtAsset(asset: string): boolean {
  return (
    asset === `tron:trc20/${USDT_CONTRACTS.nile}` ||
    asset === `tron:trc20/${USDT_CONTRACTS.mainnet}`
  );
}
