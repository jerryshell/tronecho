import { getNatsConnection } from "../utils/tronecho";
import { JSONCodec } from "nats";
import { logger } from "../utils/logger";

const jc = JSONCodec();

export default eventHandler(async () => {
  const config = useRuntimeConfig();
  const nc = await getNatsConnection(config.natsUrl);
  const prefix = config.tronechoPrefix;

  try {
    const resp = await nc.request(`${prefix}.status.v1.get`, undefined, { timeout: 5000 });
    const result = jc.decode(resp.data) as {
      ok: boolean;
      data?: {
        chainHeight: number;
        processedHeight: number;
        lag: number;
        failedBlocks: number;
        addresses: number;
        assetsCached: number;
        activeNode: string;
        startedAt: number;
        uptimeSec: number;
      };
      error?: { code: string; message: string };
    };

    if (!result.ok) {
      throw createError({
        statusCode: 502,
        statusMessage: result.error?.message || "tronecho 返回错误",
      });
    }

    return result.data;
  } catch (error: any) {
    if (error.statusCode) throw error;
    logger.error("获取 tronecho 状态失败", { error: String(error) });
    throw createError({
      statusCode: 502,
      statusMessage: "无法连接 tronecho",
    });
  }
});
