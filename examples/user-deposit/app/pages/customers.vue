<script setup lang="ts">
import type { Customer } from "~/types";
import { formatBalance } from "~/utils/customers";

const table = useTemplateRef("table");
const { columns } = useCustomersColumns();

const { data, status, refresh } = await useFetch<Customer[]>("/api/customers", {
  lazy: true,
});

const updateModal = useTemplateRef("updateModal");
const historyDrawer = useTemplateRef("historyDrawer");
const deleteModal = useTemplateRef("deleteModal");

function handleEdit(customer: Customer) {
  updateModal.value?.openModal(customer);
}

function handleHistory(customer: Customer) {
  historyDrawer.value?.openDrawer(customer);
}

function handleDelete(customer: Customer) {
  deleteModal.value?.openModal(customer);
}

const toast = useToast();

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.add({
    title: "已复制",
    description: "钱包地址已复制到剪贴板",
  });
}

// Calculate total stats
const totalCustomers = computed(() => data.value?.length || 0);
const totalBalance = computed(() => data.value?.reduce((sum, c) => sum + c.balance, 0) || 0);
</script>

<template>
  <UDashboardPanel id="customers">
    <template #header>
      <UDashboardNavbar title="用户管理">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            icon="i-lucide-refresh-cw"
            size="xs"
            color="neutral"
            variant="ghost"
            :loading="status === 'pending'"
            @click="() => refresh()"
          />
          <CustomersAddModal @success="refresh" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <!-- Stats -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <UCard>
          <div class="flex items-center gap-3">
            <div class="p-2 bg-primary/10 rounded-lg">
              <UIcon name="i-lucide-users" class="w-5 h-5 text-primary" />
            </div>
            <div>
              <p class="text-sm text-muted">用户总数</p>
              <p class="text-2xl font-bold">{{ totalCustomers }}</p>
            </div>
          </div>
        </UCard>

        <UCard>
          <div class="flex items-center gap-3">
            <div class="p-2 bg-success/10 rounded-lg">
              <UIcon name="i-lucide-wallet" class="w-5 h-5 text-success" />
            </div>
            <div>
              <p class="text-sm text-muted">总余额</p>
              <p class="text-2xl font-bold">{{ formatBalance(totalBalance) }}</p>
            </div>
          </div>
        </UCard>
      </div>

      <!-- Table -->
      <UTable
        ref="table"
        :data="data"
        :columns="columns"
        :loading="status === 'pending'"
        :ui="{
          base: 'table-auto border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0',
        }"
      >
        <template #actions-cell="{ row }">
          <div class="flex items-center justify-end gap-1">
            <UButton
              icon="i-lucide-pencil"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="handleEdit(row.original)"
            />
            <UButton
              icon="i-lucide-history"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="handleHistory(row.original)"
            />
            <UButton
              icon="i-lucide-trash"
              size="xs"
              color="error"
              variant="ghost"
              @click="handleDelete(row.original)"
            />
          </div>
        </template>

        <template #address-cell="{ row }">
          <div class="flex items-center gap-1">
            <span class="font-mono text-xs break-all">{{ row.original.address }}</span>
            <UButton
              icon="i-lucide-copy"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="copyToClipboard(row.original.address)"
            />
          </div>
        </template>
      </UTable>

      <!-- Modals & Drawers -->
      <CustomersUpdateModal ref="updateModal" @success="refresh" />
      <CustomersDepositHistoryDrawer ref="historyDrawer" />
      <CustomersDeleteModal ref="deleteModal" @success="refresh" />
    </template>
  </UDashboardPanel>
</template>
