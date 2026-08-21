<script setup lang="ts">
import ConnectionListItem from '@/components/connections/ConnectionListItem.vue';
import PrerequisitesFooter from '@/components/PrerequisitesFooter.vue';
import CircularProgress from '@/components/CircularProgress.vue';
import { useProfileUiStore } from '@/stores/profileUi';
import { useProfilesStore } from '@/stores/profiles';
import { ref } from 'vue';

const profilesStore = useProfilesStore();
const profileUi = useProfileUiStore();

const expanded = ref(true);
</script>

<template>
  <aside class="flex w-72 shrink-0 flex-col border-r border-default">
    <button
      type="button"
      class="flex items-center gap-1.5 px-3 py-2.5 text-xs font-semibold uppercase tracking-wide text-muted"
      @click="expanded = !expanded"
    >
      <UIcon :name="expanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" class="size-3.5" />
      Conexiones
    </button>

    <div v-if="expanded" class="flex-1 overflow-y-auto px-2 pb-2">
      <div v-if="profilesStore.loading" class="flex flex-col items-center justify-center gap-2 py-10">
        <CircularProgress class="h-6 w-6" />
        <span class="text-xs font-semibold select-none font-mono-data">Cargando…</span>
      </div>
      <div v-else-if="profilesStore.profiles.length === 0" class="flex flex-col items-center justify-center gap-2 py-10 text-center text-muted">
        <UIcon name="i-lucide-inbox" class="size-5" />
        <span class="text-xs font-semibold select-none font-mono-data">Sin conexiones</span>
      </div>
      <div v-else class="flex flex-col gap-0.5">
        <ConnectionListItem
          v-for="profile in profilesStore.profiles"
          :key="profile.id"
          :profile="profile"
          :selected="profile.id === profileUi.selectedProfileId"
        />
      </div>
    </div>
    <div v-else class="flex-1" />

    <PrerequisitesFooter />
  </aside>
</template>
