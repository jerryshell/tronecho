<script setup lang="ts">
const modelValue = defineModel<number>({ default: 0 });

defineProps<{
  help?: string;
}>();
</script>

<template>
  <UFormField label="充值费率" name="feeRate" :help="help || '0.02 表示收取 2% 手续费'">
    <UInput
      v-model.number="modelValue"
      class="w-full"
      type="number"
      step="0.01"
      min="0"
      max="0.99"
      placeholder="0.02"
    />
  </UFormField>
</template>
