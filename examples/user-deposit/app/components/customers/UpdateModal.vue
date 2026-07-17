<script setup lang="ts">
import * as z from "zod";
import type { FormSubmitEvent } from "@nuxt/ui";
import type { Customer } from "~/types";

const schema = z.object({
  label: z.string().optional(),
  feeRate: z.number().min(0, "费率不能为负").max(0.99, "费率不能超过 99%"),
});

const open = ref(false);
const customer = ref<Customer | null>(null);

type Schema = z.output<typeof schema>;

const state = reactive<Partial<Schema>>({
  label: "",
  feeRate: 0,
});

const { showSuccess, showError } = useCustomerForm();
const emit = defineEmits<{
  success: [];
}>();

function openModal(c: Customer) {
  customer.value = c;
  state.label = c.label || "";
  state.feeRate = c.feeRate;
  open.value = true;
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  if (!customer.value) return;

  try {
    await $fetch(`/api/customers/${customer.value.id}`, {
      method: "PATCH",
      body: {
        label: event.data.label || "",
        feeRate: event.data.feeRate,
      },
    });

    showSuccess("用户信息已更新");
    open.value = false;
    emit("success");
  } catch (error: any) {
    showError(error, "更新失败");
  }
}

defineExpose({ openModal });
</script>

<template>
  <UModal v-model:open="open" title="编辑用户" description="更新用户标签和充值费率">
    <template #body>
      <UForm :schema="schema" :state="state" class="space-y-4" @submit="onSubmit">
        <CustomersFormFields
          v-model:label="state.label"
          v-model:fee-rate="state.feeRate"
          show-label
        />

        <div class="flex justify-end gap-2">
          <UButton label="取消" color="neutral" variant="subtle" @click="open = false" />
          <UButton label="保存" color="primary" variant="solid" type="submit" />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
