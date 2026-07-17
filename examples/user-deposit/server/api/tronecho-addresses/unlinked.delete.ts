import { listAddresses, removeAddress } from "../../utils/tronecho";
import { listCustomers } from "../../utils/storage";
import { logger } from "../../utils/logger";

export default eventHandler(async () => {
  const [addresses, customers] = await Promise.all([listAddresses(), listCustomers()]);

  const customerAddresses = new Set(customers.map((c) => c.address));
  const unlinked = addresses.filter((a) => !customerAddresses.has(a.address));

  if (unlinked.length === 0) {
    return { ok: true, deleted: 0 };
  }

  logger.info("批量删除未关联地址", { count: unlinked.length });

  let deleted = 0;
  for (const addr of unlinked) {
    const ok = await removeAddress(addr.address);
    if (ok) deleted++;
  }

  return { ok: true, deleted };
});
