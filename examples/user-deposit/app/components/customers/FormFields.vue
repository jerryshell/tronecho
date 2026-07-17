<script setup lang="ts">
defineProps<{
  showLabel?: boolean;
}>();

const label = defineModel<string>("label", { default: "" });
const feeRate = defineModel<number>("feeRate", { default: 0 });
</script>

<template>
  <UFormField v-if="showLabel" label="用户标签" name="label" help="可选，用于标识用户">
    <UInput v-model="label" class="w-full" placeholder="例如：VIP客户" />
  </UFormField>

  <CustomersFeeRateField v-model="feeRate" />
</template>
