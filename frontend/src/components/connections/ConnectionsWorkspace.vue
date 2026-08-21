<script setup lang="ts">
import ConnectionDetailPanel from '@/components/connections/ConnectionDetailPanel.vue';
import ConnectionsSidebar from '@/components/connections/ConnectionsSidebar.vue';
import { useProfilesStore } from '@/stores/profiles';
import { useTunnelsStore } from '@/stores/tunnels';
import { onMounted } from 'vue';

const profilesStore = useProfilesStore();
const tunnelsStore = useTunnelsStore();

onMounted(() => {
  profilesStore.fetchAll();
  tunnelsStore.fetchAll();
  tunnelsStore.subscribe();
});
</script>

<template>
  <div class="flex min-h-0 flex-1 overflow-hidden">
    <ConnectionsSidebar />
    <ConnectionDetailPanel />
  </div>
</template>
