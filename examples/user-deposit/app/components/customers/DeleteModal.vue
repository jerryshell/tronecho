<script setup lang="ts">
import type { Customer } from "~/types";

const open = ref(false);
const customer = ref<Customer | null>(null);

const toast = useToast();

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.add({
    title: "已复制",
    description: "钱包地址已复制到剪贴板",
  });
}

const { showSuccess, showError } = useCustomerForm();
const emit = defineEmits<{
  success: [];
}>();

function openModal(c: Customer) {
  customer.value = c;
  open.value = true;
}

async function confirmDelete() {
  if (!customer.value) return;

  try {
    await $fetch(`/api/customers/${customer.value.id}`, {
      method: "DELETE",
    });

    showSuccess("用户已删除");
    open.value = false;
    emit("success");
  } catch (error: any) {
    showError(error, "删除失败");
  }
}

defineExpose({ openModal });
</script>

<template>
  <UModal v-model:open="open" title="删除用户" description="确认删除此用户？">
    <template #body>
      <div v-if="customer" class="space-y-4">
        <div class="p-4 bg-error/10 rounded-lg">
          <p class="text-sm">
            即将删除用户：
            <span class="font-medium">{{ customer.label || customer.address }}</span>
          </p>
          <div class="flex items-center gap-1 mt-1">
            <p class="text-xs font-mono break-all">{{ customer.address }}</p>
            <UButton
              icon="i-lucide-copy"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="copyToClipboard(customer.address)"
            />
          </div>
          <p class="text-sm text-muted mt-1">此操作将同时删除该用户的所有充值记录，且无法撤销。</p>
        </div>

        <div class="flex justify-end gap-2">
          <UButton label="取消" color="neutral" variant="subtle" @click="open = false" />
          <UButton label="删除" color="error" variant="solid" @click="confirmDelete" />
        </div>
      </div>
    </template>
  </UModal>
</template>
