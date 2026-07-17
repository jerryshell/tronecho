<script setup lang="ts">
import type { Customer, Deposit } from "~/types";
import { formatUsdtAmount, formatTimestamp } from "~/utils/customers";

const open = ref(false);
const customer = ref<Customer | null>(null);
const deposits = ref<Deposit[]>([]);
const loading = ref(false);

const toast = useToast();

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.add({
    title: "已复制",
    description: "已复制到剪贴板",
  });
}

async function openDrawer(c: Customer) {
  customer.value = c;
  open.value = true;
  loading.value = true;

  try {
    const data = await $fetch<Deposit[]>(`/api/customers/${c.id}/deposits`);
    deposits.value = data;
  } catch (error) {
    console.error("Failed to fetch deposits:", error);
  } finally {
    loading.value = false;
  }
}

defineExpose({ openDrawer });
</script>

<template>
  <USlideover v-model:open="open">
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-history" class="w-5 h-5" />
        <div>
          <h3 class="text-lg font-semibold">充值历史</h3>
          <p v-if="customer" class="text-sm text-muted">
            {{ customer.label || customer.address }}
          </p>
        </div>
      </div>
      <div v-if="customer" class="flex items-center gap-1 mt-1">
        <p class="text-xs font-mono break-all">{{ customer.address }}</p>
        <UButton
          icon="i-lucide-copy"
          size="xs"
          color="neutral"
          variant="ghost"
          @click="copyToClipboard(customer.address)"
        />
      </div>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-12">
        <UIcon name="i-lucide-loader-2" class="w-6 h-6 animate-spin" />
        <span class="ml-2">加载中...</span>
      </div>

      <div
        v-else-if="deposits.length === 0"
        class="flex flex-col items-center justify-center py-12 text-muted"
      >
        <UIcon name="i-lucide-inbox" class="w-12 h-12 mb-4" />
        <p>暂无充值记录</p>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="deposit in deposits"
          :key="deposit.id"
          class="border border-default rounded-lg p-4"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <UBadge color="success" variant="subtle">充值</UBadge>
              <span class="text-sm font-medium">
                {{ formatUsdtAmount(deposit.amountUsdt) }}
              </span>
            </div>
            <span class="text-xs text-muted">
              {{ formatTimestamp(deposit.blockTime) }}
            </span>
          </div>

          <div class="grid grid-cols-2 gap-2 text-sm">
            <div class="text-muted">交易哈希</div>
            <div class="font-mono break-all flex items-center gap-1">
              <span>{{ deposit.txHash }}</span>
              <UButton
                icon="i-lucide-copy"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="copyToClipboard(deposit.txHash)"
              />
            </div>

            <div class="text-muted">发送方</div>
            <div class="font-mono break-all flex items-center gap-1">
              <span>{{ deposit.from }}</span>
              <UButton
                icon="i-lucide-copy"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="copyToClipboard(deposit.from)"
              />
            </div>

            <div class="text-muted">原始金额</div>
            <div>{{ deposit.amountRaw }} ({{ deposit.decimals }} decimals)</div>

            <div class="text-muted">USDT 金额</div>
            <div>{{ formatUsdtAmount(deposit.amountUsdt) }}</div>

            <div class="text-muted">费率</div>
            <div>{{ (deposit.feeRate * 100).toFixed(1) }}%</div>

            <div class="text-muted">手续费</div>
            <div>{{ formatUsdtAmount(deposit.feeAmount) }}</div>

            <div class="text-muted">入账金额</div>
            <div class="font-medium text-success">${{ deposit.creditAmount.toFixed(2) }}</div>

            <div class="text-muted">区块号</div>
            <div class="font-mono">{{ deposit.blockNumber }}</div>

            <div class="text-muted">区块哈希</div>
            <div class="font-mono text-xs break-all flex items-center gap-1">
              <span>{{ deposit.blockHash }}</span>
              <UButton
                icon="i-lucide-copy"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="copyToClipboard(deposit.blockHash)"
              />
            </div>

            <div class="text-muted">资产</div>
            <div class="font-mono text-xs break-all flex items-center gap-1">
              <span>{{ deposit.asset }}</span>
              <UButton
                icon="i-lucide-copy"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="copyToClipboard(deposit.asset)"
              />
            </div>

            <div class="text-muted">用户标签</div>
            <div>{{ deposit.label || "-" }}</div>
          </div>
        </div>
      </div>
    </template>
  </USlideover>
</template>
