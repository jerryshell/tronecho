export default eventHandler(async (event) => {
  const body = await readBody(event);
  const address = validateAddress(body.address);
  const feeRate = validateFeeRate(body.feeRate);

  logger.info("[POST /api/customers]", { address, label: body.label });

  const existing = await getCustomerByAddress(address);
  if (existing) {
    throw createError({
      statusCode: 409,
      statusMessage: "该地址已注册",
    });
  }

  // registerAddress 幂等，重复调用不报错
  await registerAddress(address, body.label || "");

  return await createCustomer({
    address,
    label: body.label || "",
    feeRate,
  });
});
