<script setup lang="ts">
import * as z from "zod";
import type { FormSubmitEvent } from "@nuxt/ui";

const schema = z.object({
  address: z.string().min(34, "钱包地址格式不正确").max(34, "钱包地址格式不正确"),
  label: z.string().optional(),
  feeRate: z.number().min(0, "费率不能为负").max(0.99, "费率不能超过 99%"),
});

const open = ref(false);

type Schema = z.output<typeof schema>;

const state = reactive<Partial<Schema>>({
  address: "",
  label: "",
  feeRate: 0,
});

const { showSuccess, showError } = useCustomerForm();
const emit = defineEmits<{
  success: [];
}>();

async function onSubmit(event: FormSubmitEvent<Schema>) {
  try {
    await $fetch("/api/customers", {
      method: "POST",
      body: {
        address: event.data.address,
        label: event.data.label || "",
        feeRate: event.data.feeRate,
      },
    });

    showSuccess("用户已创建");
    open.value = false;
    state.address = "";
    state.label = "";
    state.feeRate = 0;
    emit("success");
  } catch (error: any) {
    showError(error, "创建用户失败");
  }
}
</script>

<template>
  <UModal v-model:open="open" title="新增用户" description="添加一个需要监控充值的用户">
    <UButton label="新增用户" icon="i-lucide-plus" />

    <template #body>
      <UForm :schema="schema" :state="state" class="space-y-4" @submit="onSubmit">
        <UFormField label="用户标签" name="label" help="可选，用于标识用户">
          <UInput v-model="state.label" class="w-full" placeholder="例如：VIP客户" />
        </UFormField>

        <UFormField label="钱包地址" name="address">
          <UInput v-model="state.address" class="w-full" placeholder="T..." />
        </UFormField>

        <CustomersFeeRateField v-model="state.feeRate" />

        <div class="flex justify-end gap-2">
          <UButton label="取消" color="neutral" variant="subtle" @click="open = false" />
          <UButton label="创建" color="primary" variant="solid" type="submit" />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
