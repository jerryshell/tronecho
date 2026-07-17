<script setup lang="ts">
import type { TronechoAddr } from "~/types";

const { status, error, loading } = useTronechoStatus(3000);

const { data: addresses, refresh: refreshAddresses } = await useFetch<TronechoAddr[]>(
  "/api/tronecho-addresses",
  {
    lazy: true,
    default: () => [],
    key: "tronecho-addresses",
  },
);

const unlinkedAddresses = computed(() => (addresses.value || []).filter((a) => !a.linked));

const deleteModal = useTemplateRef("deleteModal");
const batchDeleteModal = useTemplateRef("batchDeleteModal");

function handleDelete(addr: TronechoAddr) {
  deleteModal.value?.openModal(addr);
}

function handleBatchDelete() {
  batchDeleteModal.value?.openModal(unlinkedAddresses.value);
}

const toast = useToast();

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.add({
    title: "已复制",
    description: "已复制到剪贴板",
  });
}

const lagPercent = computed(() => {
  if (!status.value || status.value.chainHeight === 0) return 0;
  return Math.min(
    100,
    ((status.value.chainHeight - status.value.lag) / status.value.chainHeight) * 100,
  );
});

const uptime = computed(() => {
  if (!status.value) return "";
  const s = status.value.uptimeSec;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
});

const lagColor = computed(() => {
  if (!status.value) return "neutral";
  if (status.value.lag <= 19) return "success";
  if (status.value.lag <= 100) return "warning";
  return "error";
});

const columns = [
  { accessorKey: "address", header: "地址" },
  { accessorKey: "label", header: "标签" },
  { accessorKey: "linked", header: "关联用户" },
  { id: "actions", header: "" },
];
</script>

<template>
  <UDashboardPanel id="status">
    <template #header>
      <UDashboardNavbar title="TronEcho 状态">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UBadge :color="error ? 'error' : 'success'" variant="subtle" size="sm">
            {{ error ? "离线" : "在线" }}
          </UBadge>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UAlert v-if="error" color="error" variant="soft" :title="error" class="mb-6" />

      <div v-if="loading" class="text-center py-12 text-muted">加载中...</div>

      <template v-else-if="status">
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2 bg-primary/10 rounded-lg">
                <UIcon name="i-lucide-blocks" class="w-5 h-5 text-primary" />
              </div>
              <div>
                <p class="text-sm text-muted">链高度</p>
                <p class="text-2xl font-bold font-mono">
                  {{ status.chainHeight.toLocaleString() }}
                </p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2 bg-success/10 rounded-lg">
                <UIcon name="i-lucide-check-circle" class="w-5 h-5 text-success" />
              </div>
              <div>
                <p class="text-sm text-muted">已处理</p>
                <p class="text-2xl font-bold font-mono">
                  {{ status.processedHeight.toLocaleString() }}
                </p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2 bg-warning/10 rounded-lg">
                <UIcon name="i-lucide-clock" class="w-5 h-5 text-warning" />
              </div>
              <div>
                <p class="text-sm text-muted">延迟</p>
                <p class="text-2xl font-bold font-mono">{{ status.lag }} 块</p>
              </div>
            </div>
          </UCard>

          <UCard>
            <div class="flex items-center gap-3">
              <div class="p-2 bg-error/10 rounded-lg">
                <UIcon name="i-lucide-alert-triangle" class="w-5 h-5 text-error" />
              </div>
              <div>
                <p class="text-sm text-muted">失败块</p>
                <p class="text-2xl font-bold font-mono">{{ status.failedBlocks }}</p>
              </div>
            </div>
          </UCard>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <StatusSyncPanel :status="status" :lag-percent="lagPercent" :lag-color="lagColor" />
          <StatusInfoPanel :status="status" :uptime="uptime" />
        </div>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <h3 class="font-semibold text-sm">监听地址</h3>
              <UButton
                v-if="unlinkedAddresses.length > 0"
                color="error"
                variant="soft"
                size="sm"
                @click="handleBatchDelete"
              >
                删除未关联地址 ({{ unlinkedAddresses.length }})
              </UButton>
            </div>
          </template>

          <UTable
            :data="addresses"
            :columns="columns"
            :ui="{
              base: 'table-auto border-separate border-spacing-0',
              thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
              tbody: '[&>tr]:last:[&>td]:border-b-0',
              th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
              td: 'border-b border-default',
              separator: 'h-0',
            }"
          >
            <template #address-cell="{ row }">
              <div class="flex items-center gap-1">
                <span class="font-mono text-xs">{{ row.original.address }}</span>
                <UButton
                  icon="i-lucide-copy"
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  @click="copyToClipboard(row.original.address)"
                />
              </div>
            </template>
            <template #linked-cell="{ row }">
              <UBadge
                :color="row.original.linked ? 'success' : 'neutral'"
                variant="subtle"
                size="sm"
              >
                {{ row.original.linked ? "是" : "否" }}
              </UBadge>
            </template>
            <template #actions-cell="{ row }">
              <UButton
                v-if="!row.original.linked"
                color="error"
                variant="ghost"
                size="xs"
                icon="i-lucide-trash-2"
                @click="handleDelete(row.original)"
              />
            </template>
          </UTable>
        </UCard>
      </template>

      <CustomersDeleteAddressModal ref="deleteModal" @success="refreshAddresses" />
      <CustomersBatchDeleteModal ref="batchDeleteModal" @success="refreshAddresses" />
    </template>
  </UDashboardPanel>
</template>
