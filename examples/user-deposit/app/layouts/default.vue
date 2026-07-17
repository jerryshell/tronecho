<script setup lang="ts">
import type { NavigationMenuItem } from "@nuxt/ui";

const route = useRoute();

const open = ref(false);

const links = [
  [
    {
      label: "用户管理",
      icon: "i-lucide-users",
      to: "/customers",
      onSelect: () => {
        open.value = false;
      },
    },
    {
      label: "TronEcho 状态",
      icon: "i-lucide-activity",
      to: "/status",
      onSelect: () => {
        open.value = false;
      },
    },
  ],
] satisfies NavigationMenuItem[][];

const groups = computed(() => [
  {
    id: "links",
    label: "导航",
    items: links.flat(),
  },
]);
</script>

<template>
  <UDashboardGroup unit="rem">
    <UDashboardSidebar
      id="default"
      v-model:open="open"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-wallet" class="w-6 h-6 text-primary" />
          <span v-if="!collapsed" class="font-bold">USDT 充值</span>
        </div>
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[0]"
          orientation="vertical"
          tooltip
          popover
        />
      </template>

      <template #footer="{ collapsed }">
        <div class="flex items-center gap-2 text-sm text-muted">
          <UIcon name="i-lucide-info" class="w-4 h-4" />
          <span v-if="!collapsed">tronecho 示例</span>
        </div>
      </template>
    </UDashboardSidebar>

    <slot />
  </UDashboardGroup>
</template>
