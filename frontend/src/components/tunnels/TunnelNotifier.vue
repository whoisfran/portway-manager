<script setup lang="ts">
import { useProfilesStore } from '@/stores/profiles';
import { useTunnelsStore } from '@/stores/tunnels';
import { watch } from 'vue';

// No renderiza nada: solo traduce las transiciones de estado del
// store de tuneles (notice) en avisos visibles para el usuario. Solo
// llega para tuneles que terminaron con error: una desconexion manual
// no avisa (ver stores/tunnels.ts).
const tunnelsStore = useTunnelsStore();
const profilesStore = useProfilesStore();
const toast = useToast();

watch(
  () => tunnelsStore.notice,
  (notice) => {
    if (!notice) return;

    // El perfil guardado (ver favoriteId) es el nombre que el usuario
    // realmente reconoce; si ya no existe (se borro mientras el tunel
    // seguia activo) se cae al nombre de instancia/perfil de AWS.
    const label =
      profilesStore.getById(notice.tunnel.request.favoriteId)?.label ||
      notice.tunnel.request.instanceLabel ||
      notice.tunnel.request.instanceId;
    toast.add({
      title: `El túnel "${label}" terminó con error`,
      description: notice.tunnel.message || undefined,
      color: 'error',
    });
  },
);
</script>
