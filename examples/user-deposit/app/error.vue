<script setup lang="ts">
import { zh_cn } from "@nuxt/ui/locale";
import type { NuxtError } from "#app";

defineProps<{
  error: NuxtError;
}>();

useSeoMeta({
  title: "页面未找到",
  description: "很抱歉，找不到该页面。",
});

useHead({
  htmlAttrs: {
    lang: "zh-CN",
  },
});
</script>

<template>
  <UApp :locale="zh_cn">
    <UError :error="error" />
  </UApp>
</template>
