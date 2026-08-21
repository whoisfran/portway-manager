<script setup lang="ts">
import CircularProgress from '@/components/CircularProgress.vue';
import { usePrerequisitesStore } from '@/stores/prerequisites';
import { onMounted } from 'vue';

const prerequisites = usePrerequisitesStore();

onMounted(() => {
  prerequisites.check();
});
</script>

<template>
  <div class="shrink-0 border-t border-default px-3 py-2">
    <div v-if="prerequisites.loading" class="flex items-center gap-2 text-muted">
      <CircularProgress class="h-3.5 w-3.5" />
      <span class="font-mono-data text-xs">Verificando AWS CLI…</span>
    </div>

    <div v-else-if="prerequisites.result && !prerequisites.result.allOk" class="flex flex-col gap-1">
      <div class="flex items-center gap-1.5 text-error">
        <UIcon name="i-lucide-triangle-alert" class="size-3.5 shrink-0" />
        <span class="font-mono-data text-xs">Falta AWS CLI o Session Manager Plugin</span>
      </div>
      <UButton
        label="Ver instalación"
        icon="i-lucide-external-link"
        color="error"
        variant="link"
        size="xs"
        class="w-fit p-0"
        @click="prerequisites.openInstallDocs()"
      />
    </div>

    <div v-else-if="prerequisites.result" class="flex flex-col gap-1 font-mono-data text-xs text-muted">
      <div class="flex items-center justify-between gap-2">
        <span>AWS CLI</span>
        <span class="text-dimmed">v{{ prerequisites.result.awsCliVersion }}</span>
      </div>
      <div class="flex items-center justify-between gap-2">
        <span>Session Manager Plugin</span>
        <span class="text-dimmed">v{{ prerequisites.result.sessionPluginVersion }}</span>
      </div>
    </div>
  </div>
</template>
