import { startNatsSubscription, stopNatsSubscription } from "../utils/nats-sub";
import { getNatsConnection, setPrefix, syncAllAddresses } from "../utils/tronecho";
import { logger } from "../utils/logger";

export default defineNitroPlugin(async (nitroApp) => {
  const config = useRuntimeConfig();

  setPrefix(config.tronechoPrefix);

  try {
    const nc = await getNatsConnection(config.natsUrl);

    // 启动时同步所有地址到 tronecho
    await syncAllAddresses();

    await startNatsSubscription(nc, config.tronechoPrefix);
    logger.info("NATS 订阅已启动");
  } catch (error) {
    logger.error("NATS 订阅启动失败", { error: String(error) });
  }

  nitroApp.hooks.hook("close", async () => {
    logger.info("关闭 NATS 订阅...");
    await stopNatsSubscription();
  });
});
