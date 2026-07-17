<script setup lang="ts">
import type { TronechoAddr } from "~/types";

const open = ref(false);
const addresses = ref<TronechoAddr[]>([]);
const deleting = ref(false);

const { showSuccess, showError } = useCustomerForm();
const emit = defineEmits<{
  success: [];
}>();

function openModal(list: TronechoAddr[]) {
  addresses.value = list;
  open.value = true;
}

async function confirmDelete() {
  deleting.value = true;
  try {
    const result = await $fetch<{ deleted: number }>("/api/tronecho-addresses/unlinked", {
      method: "DELETE",
    });
    showSuccess(`已删除 ${result.deleted} 个地址`);
    open.value = false;
    emit("success");
  } catch (error: any) {
    showError(error, "删除失败");
  } finally {
    deleting.value = false;
  }
}

defineExpose({ openModal });
</script>

<template>
  <UModal
    v-model:open="open"
    title="批量删除未关联地址"
    description="以下地址未关联任何用户，将从监听列表中删除："
  >
    <template #body>
      <div class="space-y-4">
        <div class="max-h-60 overflow-y-auto space-y-2">
          <div v-for="addr in addresses" :key="addr.address" class="p-3 bg-elevated rounded-lg">
            <p class="text-xs font-mono break-all">{{ addr.address }}</p>
            <p class="text-xs text-muted mt-1">{{ addr.label || "(无标签)" }}</p>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <UButton label="取消" color="neutral" variant="ghost" @click="open = false" />
          <UButton label="确认删除" color="error" :loading="deleting" @click="confirmDelete" />
        </div>
      </div>
    </template>
  </UModal>
</template>
