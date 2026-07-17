export interface TronechoStatus {
  chainHeight: number;
  processedHeight: number;
  lag: number;
  failedBlocks: number;
  addresses: number;
  assetsCached: number;
  activeNode: string;
  startedAt: number;
  uptimeSec: number;
}

export function useTronechoStatus(intervalMs = 3000) {
  const status = ref<TronechoStatus | null>(null);
  const error = ref<string | null>(null);
  const loading = ref(true);

  const fetchStatus = async () => {
    try {
      const data = await $fetch<TronechoStatus>("/api/tronecho-status");
      status.value = data;
      error.value = null;
    } catch (e: any) {
      error.value = e.data?.statusMessage || "连接失败";
    } finally {
      loading.value = false;
    }
  };

  let timer: ReturnType<typeof setInterval> | null = null;

  onMounted(() => {
    fetchStatus();
    timer = setInterval(fetchStatus, intervalMs);
  });

  onUnmounted(() => {
    if (timer) clearInterval(timer);
  });

  return { status, error, loading };
}
