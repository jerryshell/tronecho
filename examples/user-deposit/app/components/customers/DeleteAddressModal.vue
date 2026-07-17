<script setup lang="ts">
import type { TronechoAddr } from "~/types";

const open = ref(false);
const addr = ref<TronechoAddr | null>(null);
const deleting = ref(false);

const { showSuccess, showError } = useCustomerForm();
const emit = defineEmits<{
  success: [];
}>();

function openModal(a: TronechoAddr) {
  addr.value = a;
  open.value = true;
}

async function confirmDelete() {
  if (!addr.value) return;
  deleting.value = true;
  try {
    await $fetch(`/api/tronecho-addresses/${addr.value.address}`, { method: "DELETE" });
    showSuccess("地址已删除");
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
    title="确认删除"
    description="确定要从 tronecho 监听列表中删除该地址？"
  >
    <template #body>
      <div v-if="addr" class="space-y-4">
        <div class="p-4 bg-elevated rounded-lg">
          <p class="text-xs font-mono break-all">{{ addr.address }}</p>
          <p class="text-xs text-muted mt-1">标签: {{ addr.label || "(无)" }}</p>
        </div>

        <div class="flex justify-end gap-2">
          <UButton label="取消" color="neutral" variant="ghost" @click="open = false" />
          <UButton label="删除" color="error" :loading="deleting" @click="confirmDelete" />
        </div>
      </div>
    </template>
  </UModal>
</template>
