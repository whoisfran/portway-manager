<script setup lang="ts">
import { windowControlsApi } from '@/api/windowControls';
import AppLogo from '@/components/AppLogo.vue';
import { onMounted, ref } from 'vue';

const isMaximised = ref(false);

onMounted(async () => {
  isMaximised.value = await windowControlsApi.isMaximised();
});

function toggleMaximise() {
  windowControlsApi.toggleMaximise();
  isMaximised.value = !isMaximised.value;
}
</script>

<template>
  <div
    class="app-region-drag flex h-11 shrink-0 items-center justify-between border-b border-default bg-elevated/60 pl-3 pr-2"
    @dblclick="toggleMaximise">
    <div class="flex items-center gap-2 select-none">
      <AppLogo :size="20" />
      <span class="text-sm font-semibold">Portway Manager</span>
    </div>

    <div class="app-region-no-drag flex items-center gap-1">
      <UButton icon="i-lucide-minus" color="neutral" variant="soft" size="xs" aria-label="Minimizar"
        @click="windowControlsApi.minimise()" />
      <UButton icon="i-lucide-x" color="error" variant="soft" size="xs" aria-label="Cerrar"
        @click="windowControlsApi.close()" />
    </div>
  </div>
</template>
