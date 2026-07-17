export default eventHandler(async (event) => {
  const id = getRouterParam(event, "id");

  logger.info("[DELETE /api/customers/:id]", { id });

  if (!id) {
    throw createError({
      statusCode: 400,
      statusMessage: "用户 ID 必填",
    });
  }

  // Get customer first to get address
  const customer = await getCustomer(id);
  if (!customer) {
    throw createError({
      statusCode: 404,
      statusMessage: "用户不存在",
    });
  }

  // Remove address from tronecho
  await removeAddress(customer.address);

  // Delete customer from storage
  await deleteCustomer(id);

  return { success: true };
});
