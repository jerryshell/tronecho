import { listAddresses, type TronechoAddress } from "../utils/tronecho";
import { listCustomers } from "../utils/storage";

export default eventHandler(async () => {
  const [addresses, customers] = await Promise.all([listAddresses(), listCustomers()]);

  const customerAddresses = new Set(customers.map((c) => c.address));

  return addresses.map((addr: TronechoAddress) => ({
    ...addr,
    linked: customerAddresses.has(addr.address),
  }));
});
