<script setup lang="ts">
import { appApi } from '@/api/app';
import type { ThemeMode } from '@/stores/theme';
import { useThemeStore } from '@/stores/theme';
import type { AppSettings } from '@/types/domain';
import { onMounted, ref } from 'vue';

const open = defineModel<boolean>('open', { default: false });
const themeStore = useThemeStore();

const themeOptions: { label: string; icon: string; value: ThemeMode }[] = [
  { label: 'Claro', icon: 'i-lucide-sun', value: 'light' },
  { label: 'Oscuro', icon: 'i-lucide-moon', value: 'dark' },
  { label: 'Sistema', icon: 'i-lucide-monitor', value: 'system' },
];

const version = ref('');
const settings = ref<AppSettings | null>(null);

onMounted(async () => {
  version.value = await appApi.getVersion();
  settings.value = await appApi.getSettings();
});

const toast = useToast();

async function updateSetting(key: keyof AppSettings, value: boolean) {
  if (!settings.value) return;
  const previous = settings.value;
  const next = { ...settings.value, [key]: value };
  settings.value = next; // optimista: la UI responde de inmediato
  try {
    settings.value = await appApi.saveSettings(next);
  } catch (err) {
    settings.value = previous;
    toast.add({ title: 'No se pudo guardar el ajuste', description: (err as Error).message, color: 'error' });
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Ajustes" close>
    <template #body>
      <div class="flex items-center justify-between gap-4">
        <div>
          <p class="text-sm font-medium">Tema</p>
          <p class="text-xs text-muted">"Sistema" sigue el tema claro/oscuro de tu equipo.</p>
        </div>
        <UTabs :model-value="themeStore.mode" :items="themeOptions" :content="false" size="xs"
          @update:model-value="(value: string | number) => themeStore.setMode(value as ThemeMode)" />
      </div>

      <div v-if="settings" class="mt-4 flex flex-col gap-3 border-t border-default pt-4">
        <p class="text-sm font-medium">Reconexión automática</p>
        <p class="text-xs text-muted">
          Si un túnel se cae solo, reintenta abrirlo unas pocas veces antes de avisarte.
        </p>

        <div class="flex items-center justify-between gap-4">
          <span class="text-sm">Conexiones SSM</span>
          <USwitch :model-value="settings.autoReconnectSsm"
            @update:model-value="(value: boolean) => updateSetting('autoReconnectSsm', value)" />
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-sm">Conexiones SSH</span>
          <USwitch :model-value="settings.autoReconnectSsh"
            @update:model-value="(value: boolean) => updateSetting('autoReconnectSsh', value)" />
        </div>
      </div>

      <p class="mt-4 border-t border-default pt-4 text-xs text-dimmed select-text">
        Portway Manager {{ version || '…' }}
      </p>
    </template>
  </UModal>
</template>
