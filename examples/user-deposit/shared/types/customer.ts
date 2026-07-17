export interface Customer {
  id: string;
  label?: string;
  address: string;
  feeRate: number; // 0-1, e.g., 0.02 = 2%
  balance: number; // USD
  createdAt: number;
  updatedAt: number;
}

export interface Deposit {
  id: string;
  customerId: string;
  eventId: string; // tronecho event id for dedup
  txHash: string;
  logIndex: number;
  asset: string;
  symbol: string;
  decimals: number;
  amountRaw: string; // raw amount in smallest unit
  amountUsdt: number; // human-readable USDT amount
  feeRate: number; // at time of deposit
  feeAmount: number; // fee in USDT
  creditAmount: number; // amount credited in USD
  blockNumber: number;
  blockHash: string;
  blockTime: number;
  from: string;
  to: string;
  direction: string;
  label?: string; // customer label at time of deposit
  createdAt: number;
}
