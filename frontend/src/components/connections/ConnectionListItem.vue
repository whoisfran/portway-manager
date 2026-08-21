<script setup lang="ts">
import { connectionStatus } from '@/composables/useConnectionStatus';
import { useProfileUiStore } from '@/stores/profileUi';
import { useTunnelsStore } from '@/stores/tunnels';
import type { ConnectionProfile } from '@/types/domain';
import { computed } from 'vue';

const props = defineProps<{ profile: ConnectionProfile; selected: boolean }>();

const tunnelsStore = useTunnelsStore();
const profileUi = useProfileUiStore();

const status = computed(() => connectionStatus(props.profile, tunnelsStore.findFor(props.profile)));

// Clases fijas de Tailwind (no las semanticas de Nuxt UI, que estan
// pensadas para props de componente, no para usarse sueltas como
// clase): verde=corriendo, naranja=falta configurar, rojo=fallo, gris=detenido.
const DOT_CLASS: Record<string, string> = {
  success: 'bg-green-500',
  warning: 'bg-orange-500',
  error: 'bg-red-500',
  neutral: 'bg-zinc-400',
};
</script>

<template>
  <button
    type="button"
    class="flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors"
    :class="selected ? 'border-primary bg-primary/10' : 'border-transparent hover:bg-elevated/60'"
    @click="profileUi.selectProfile(profile.id)"
  >
    <span class="size-2.5 shrink-0 rounded-full" :class="DOT_CLASS[status.color]" />

    <span class="flex min-w-0 flex-1 flex-col">
      <span class="truncate text-sm font-medium">{{ profile.label }}</span>
      <span class="truncate font-mono-data text-xs text-muted">{{ profile.region }}</span>
    </span>

    <UIcon name="i-lucide-chevron-right" class="size-4 shrink-0 text-dimmed" />
  </button>
</template>
