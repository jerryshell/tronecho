import { removeAddress } from "../../utils/tronecho";
import { listCustomers } from "../../utils/storage";
import { logger } from "../../utils/logger";

export default eventHandler(async (event) => {
  const address = getRouterParam(event, "address");

  if (!address) {
    throw createError({ statusCode: 400, statusMessage: "地址必填" });
  }

  // 检查是否关联用户
  const customers = await listCustomers();
  const linked = customers.some((c) => c.address === address);
  if (linked) {
    throw createError({
      statusCode: 409,
      statusMessage: "该地址关联了用户，无法删除",
    });
  }

  logger.info("删除 tronecho 监听地址", { address });
  const ok = await removeAddress(address);

  if (!ok) {
    throw createError({
      statusCode: 502,
      statusMessage: "删除失败",
    });
  }

  return { ok: true };
});
