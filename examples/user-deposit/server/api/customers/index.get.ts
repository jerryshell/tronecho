export default eventHandler(async () => {
  logger.info("[GET /api/customers]");
  return await listCustomers();
});
