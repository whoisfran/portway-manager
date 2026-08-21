<script setup lang="ts">
import ConnectionListItem from '@/components/connections/ConnectionListItem.vue';
import PrerequisitesFooter from '@/components/PrerequisitesFooter.vue';
import CircularProgress from '@/components/CircularProgress.vue';
import { useProfileUiStore } from '@/stores/profileUi';
import { useProfilesStore } from '@/stores/profiles';
import type { ConnectionProfile } from '@/types/domain';
import { computed, ref } from 'vue';

const profilesStore = useProfilesStore();
const profileUi = useProfileUiStore();

const expanded = ref(true);

type ProfileGroup = { name: string | null; profiles: ConnectionProfile[] };
type TypeSection = { type: string; label: string; groups: ProfileGroup[] };

const byLabel = (a: ConnectionProfile, b: ConnectionProfile) => a.label.localeCompare(b.label);

// Agrupa los perfiles de un mismo tipo por su campo "group" (libre,
// opcional): cada nombre de grupo se muestra como un sub-encabezado
// dentro de la seccion del tipo, y los que no tienen grupo quedan
// sueltos al final, sin encabezado.
function groupProfiles(profiles: ConnectionProfile[]): ProfileGroup[] {
  const byGroup = new Map<string, ConnectionProfile[]>();
  const ungrouped: ConnectionProfile[] = [];

  for (const profile of profiles) {
    const group = profile.group?.trim();
    if (group) {
      const bucket = byGroup.get(group) ?? [];
      bucket.push(profile);
      byGroup.set(group, bucket);
    } else {
      ungrouped.push(profile);
    }
  }

  const groups: ProfileGroup[] = Array.from(byGroup.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, groupProfiles]) => ({ name, profiles: groupProfiles.sort(byLabel) }));

  if (ungrouped.length > 0) {
    groups.push({ name: null, profiles: ungrouped.sort(byLabel) });
  }
  return groups;
}

// SSM primero (el tipo original de la app), luego SSH; una seccion
// sin perfiles no se muestra -- no tiene sentido un encabezado vacio.
const sections = computed<TypeSection[]>(() => {
  const all = [
    { type: 'ssm', label: 'SSM' },
    { type: 'ssh', label: 'SSH' },
  ];
  return all
    .map(({ type, label }) => ({
      type,
      label,
      groups: groupProfiles(profilesStore.profiles.filter((p) => (p.type || 'ssm') === type)),
    }))
    .filter((section) => section.groups.length > 0);
});
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
      <div v-else class="flex flex-col gap-2">
        <div v-for="section in sections" :key="section.type" class="flex flex-col gap-0.5">
          <p class="px-2 pt-1 text-xs font-semibold uppercase tracking-wide text-dimmed">{{ section.label }}</p>

          <template v-for="group in section.groups" :key="group.name ?? '__ungrouped__'">
            <p v-if="group.name" class="truncate px-2 pt-1 text-xs text-muted">{{ group.name }}</p>
            <ConnectionListItem
              v-for="profile in group.profiles"
              :key="profile.id"
              :profile="profile"
              :selected="profile.id === profileUi.selectedProfileId"
            />
          </template>
        </div>
      </div>
    </div>
    <div v-else class="flex-1" />

    <PrerequisitesFooter />
  </aside>
</template>
