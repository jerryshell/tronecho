export default eventHandler(async (event) => {
  const id = getRouterParam(event, "id");

  logger.info("[GET /api/customers/:id/deposits]", { id });

  if (!id) {
    throw createError({
      statusCode: 400,
      statusMessage: "用户 ID 必填",
    });
  }

  return await listDepositsByCustomer(id);
});
