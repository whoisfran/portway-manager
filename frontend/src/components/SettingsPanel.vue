<script setup lang="ts">
import type { ThemeMode } from '@/stores/theme';
import { useThemeStore } from '@/stores/theme';

const open = defineModel<boolean>('open', { default: false });
const themeStore = useThemeStore();

const themeOptions: { label: string; icon: string; value: ThemeMode }[] = [
  { label: 'Claro', icon: 'i-lucide-sun', value: 'light' },
  { label: 'Oscuro', icon: 'i-lucide-moon', value: 'dark' },
  { label: 'Sistema', icon: 'i-lucide-monitor', value: 'system' },
];
</script>

<template>
  <UModal v-model:open="open" title="Ajustes" close>
    <template #body>
      <div class="flex items-center justify-between gap-4">
        <div>
          <p class="text-sm font-medium">Tema</p>
          <p class="text-xs text-muted">"Sistema" sigue el tema claro/oscuro de tu equipo.</p>
        </div>
        <UTabs
          :model-value="themeStore.mode"
          :items="themeOptions"
          :content="false"
          size="xs"
          @update:model-value="(value: string | number) => themeStore.setMode(value as ThemeMode)"
        />
      </div>
    </template>
  </UModal>
</template>
