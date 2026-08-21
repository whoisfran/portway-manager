<script setup lang="ts">
import { useProfileUiStore } from '@/stores/profileUi';
import { useProfilesStore } from '@/stores/profiles';
import { ref } from 'vue';

const profilesStore = useProfilesStore();
const profileUi = useProfileUiStore();
const toast = useToast();

const exporting = ref(false);
const importing = ref(false);

async function exportProfiles() {
  exporting.value = true;
  try {
    const path = await profilesStore.exportProfiles();
    if (path) {
      toast.add({ title: 'Perfiles exportados', description: path, color: 'success' });
    }
  } catch (err) {
    toast.add({ title: 'No se pudieron exportar los perfiles', description: (err as Error).message, color: 'error' });
  } finally {
    exporting.value = false;
  }
}

async function importProfiles() {
  importing.value = true;
  try {
    const result = await profilesStore.importProfiles();
    if (!result || (result.importedCount === 0 && result.failures.length === 0)) {
      return; // el usuario cerro el dialogo sin elegir archivo
    }

    toast.add({
      title: `Se importaron ${result.importedCount} perfil(es)`,
      description:
        result.failures.length > 0
          ? `${result.failures.length} no se pudieron importar: ${result.failures.map((f) => f.label).join(', ')}`
          : 'Revisa el perfil de AWS de cada uno antes de usarlo: no viaja en la exportación.',
      color: result.failures.length > 0 ? 'warning' : 'success',
    });
  } catch (err) {
    toast.add({ title: 'No se pudo importar el archivo', description: (err as Error).message, color: 'error' });
  } finally {
    importing.value = false;
  }
}
</script>

<template>
  <div class="flex h-12 shrink-0 items-center justify-between gap-2 border-b border-default px-3">
    <div class="flex items-center gap-1.5">
      <UButton icon="i-lucide-plus" label="Nueva" size='sm' variant="soft" @click="profileUi.openCreate" />
      <UButton icon="i-lucide-download" label="Importar" size='sm' color="neutral" variant="soft" :loading="importing"
        @click="importProfiles" />
      <UButton icon="i-lucide-upload" label="Exportar" size='sm' color="neutral" variant="soft" :loading="exporting"
        @click="exportProfiles" />
    </div>

    <!--
      Ajustes: sin nada que configurar todavia (ver SettingsPanel.vue),
      se deja listo pero apagado para no mostrar un boton que no hace
      nada.

      <UButton icon="i-lucide-settings" color="neutral" variant="ghost" aria-label="Ajustes" @click="settingsOpen = true" />
      <SettingsPanel v-model:open="settingsOpen" />
    -->
  </div>
</template>
