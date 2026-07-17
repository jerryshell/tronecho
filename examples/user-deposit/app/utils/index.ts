export function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

export function randomFrom<T>(array: T[]): T {
  return array[Math.floor(Math.random() * array.length)]!;
}

export function formatTimeAgoZh(date: Date): string {
  const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
  if (seconds < 10) return "刚刚";

  const tiers = [
    { limit: 60, divisor: 1, label: "秒前" },
    { limit: 3600, divisor: 60, label: "分钟前" },
    { limit: 86400, divisor: 3600, label: "小时前" },
    { limit: 2592000, divisor: 86400, label: "天前" },
    { limit: 31536000, divisor: 2592000, label: "个月前" },
    { limit: Infinity, divisor: 31536000, label: "年前" },
  ] as const;

  const tier = tiers.find((t) => seconds < t.limit)!;
  return `${Math.floor(seconds / tier.divisor)} ${tier.label}`;
}
