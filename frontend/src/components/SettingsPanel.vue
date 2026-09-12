<script setup lang="ts">
import { appApi } from '@/api/app';
import type { ThemeMode } from '@/stores/theme';
import { useThemeStore } from '@/stores/theme';
import { onMounted, ref } from 'vue';

const open = defineModel<boolean>('open', { default: false });
const themeStore = useThemeStore();

const themeOptions: { label: string; icon: string; value: ThemeMode }[] = [
  { label: 'Claro', icon: 'i-lucide-sun', value: 'light' },
  { label: 'Oscuro', icon: 'i-lucide-moon', value: 'dark' },
  { label: 'Sistema', icon: 'i-lucide-monitor', value: 'system' },
];

const version = ref('');
onMounted(async () => {
  version.value = await appApi.getVersion();
});
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

      <p class="mt-4 border-t border-default pt-4 text-xs text-dimmed select-text">
        Portway Manager {{ version || '…' }}
      </p>
    </template>
  </UModal>
</template>
