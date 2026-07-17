<script setup lang="ts">
import type { TronechoStatus } from "~/composables/useTronechoStatus";

const props = defineProps<{
  status: TronechoStatus;
  uptime: string;
}>();

const toast = useToast();

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.add({
    title: "已复制",
    description: "已复制到剪贴板",
  });
}
</script>

<template>
  <UCard>
    <template #header>
      <h3 class="font-semibold text-sm">详细信息</h3>
    </template>
    <div class="space-y-3 text-sm">
      <div class="flex justify-between">
        <span class="text-muted">RPC 节点</span>
        <div class="flex items-center gap-1">
          <span class="font-mono text-xs truncate max-w-48">{{ props.status.activeNode }}</span>
          <UButton
            icon="i-lucide-copy"
            size="xs"
            color="neutral"
            variant="ghost"
            @click="copyToClipboard(props.status.activeNode)"
          />
        </div>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">监听地址数</span>
        <span class="font-mono">{{ props.status.addresses }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">资产缓存</span>
        <span class="font-mono">{{ props.status.assetsCached }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-muted">运行时间</span>
        <span class="font-mono">{{ props.uptime }}</span>
      </div>
    </div>
  </UCard>
</template>
