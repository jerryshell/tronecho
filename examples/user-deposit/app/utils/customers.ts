export function formatFeeRate(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

export function formatBalance(balance: number): string {
  return `$${balance.toFixed(2)}`;
}

export function formatUsdtAmount(amount: number): string {
  return `${amount.toFixed(6)} USDT`;
}

export function formatAddress(address: string): string {
  if (address.length <= 12) return address;
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
}

export function formatTxHash(hash: string): string {
  if (hash.length <= 16) return hash;
  return `${hash.slice(0, 8)}...${hash.slice(-6)}`;
}

export function formatTimestamp(ts: number): string {
  return new Date(ts).toLocaleString("zh-CN");
}
