export function validateAddress(address: unknown): string {
  if (!address || typeof address !== "string") {
    throw createError({
      statusCode: 400,
      statusMessage: "钱包地址必填",
    });
  }
  return address;
}

export function validateFeeRate(feeRate: unknown): number {
  const rate = Number(feeRate) || 0;
  if (rate < 0 || rate >= 1) {
    throw createError({
      statusCode: 400,
      statusMessage: "费率必须在 0 到 1 之间",
    });
  }
  return rate;
}

export function validateId(id: string | undefined): string {
  if (!id) {
    throw createError({
      statusCode: 400,
      statusMessage: "用户 ID 必填",
    });
  }
  return id;
}
