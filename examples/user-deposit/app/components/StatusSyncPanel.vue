<script setup lang="ts">
import type { TronechoStatus } from "~/composables/useTronechoStatus";

const props = defineProps<{
  status: TronechoStatus;
  lagPercent: number;
  lagColor: "success" | "warning" | "error" | "neutral";
}>();
</script>

<template>
  <UCard>
    <template #header>
      <h3 class="font-semibold text-sm">同步进度</h3>
    </template>
    <div class="flex items-center justify-between mb-2">
      <span class="text-xs text-muted">已同步 {{ props.lagPercent.toFixed(2) }}%</span>
      <span class="text-xs font-mono text-muted">
        {{ props.status.processedHeight.toLocaleString() }} /
        {{ props.status.chainHeight.toLocaleString() }}
      </span>
    </div>
    <UProgress :value="props.lagPercent" :color="props.lagColor" />
  </UCard>
</template>
