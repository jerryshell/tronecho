export default eventHandler(async (event) => {
  const id = validateId(getRouterParam(event, "id"));
  const body = await readBody(event);

  logger.info("[PATCH /api/customers/:id]", { id, ...body });

  const customer = await getCustomer(id);
  if (!customer) {
    throw createError({
      statusCode: 404,
      statusMessage: "用户不存在",
    });
  }

  const updateData: { label?: string; feeRate?: number } = {};
  if (body.label !== undefined) {
    updateData.label = body.label;
  }
  if (body.feeRate !== undefined) {
    updateData.feeRate = validateFeeRate(body.feeRate);
  }

  const result = await updateCustomer(id, updateData);
  if (!result) {
    throw createError({
      statusCode: 404,
      statusMessage: "用户不存在",
    });
  }

  if (updateData.label !== undefined) {
    await registerAddress(customer.address, updateData.label);
  }

  return result;
});
