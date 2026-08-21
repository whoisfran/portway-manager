<script setup lang="ts">
import { connectionStatus } from '@/composables/useConnectionStatus';
import type { ConnectionProfile, Tunnel } from '@/types/domain';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

const props = defineProps<{ profile: ConnectionProfile; tunnel: Tunnel | undefined }>();

const status = computed(() => connectionStatus(props.profile, props.tunnel));

const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now();
  }, 1000);
});
onBeforeUnmount(() => {
  if (timer) clearInterval(timer);
});

const elapsed = computed(() => {
  if (!props.tunnel || (props.tunnel.status !== 'running' && props.tunnel.status !== 'starting')) return null;

  const startedMs = new Date(props.tunnel.startedAt).getTime();
  if (Number.isNaN(startedMs)) return null;

  const totalSeconds = Math.max(0, Math.floor((now.value - startedMs) / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const pad = (n: number) => n.toString().padStart(2, '0');

  return hours > 0 ? `${pad(hours)}:${pad(minutes)}:${pad(seconds)}` : `${pad(minutes)}:${pad(seconds)}`;
});
</script>

<template>
  <div class="flex items-center gap-2">
    <UBadge :color="status.color" variant="subtle" size="sm">{{ status.label }}</UBadge>
    <span v-if="elapsed" class="flex items-center gap-1 font-mono-data text-xs text-muted">
      <UIcon name="i-lucide-clock" class="size-3.5" />
      {{ elapsed }}
    </span>
  </div>
</template>
