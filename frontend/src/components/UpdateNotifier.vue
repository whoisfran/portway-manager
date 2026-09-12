<script setup lang="ts">
// No renderiza nada: solo escucha "update:available" (ver
// checkForUpdatesOnStartup, en update_check.go) y lo traduce en un
// toast persistente con un boton a la pagina del release. Esta app no
// se autoactualiza -- el usuario baja e instala la version nueva el
// mismo.
import { appApi } from '@/api/app';
import { onUpdateAvailable } from '@/api/events';
import type { UpdateInfo } from '@/types/domain';
import { onMounted } from 'vue';

const toast = useToast();

function notify(info: UpdateInfo) {
  toast.add({
    title: `Hay una versión nueva: ${info.latestVersion}`,
    description: `Estás usando ${info.currentVersion}.`,
    color: 'info',
    duration: 0, // se queda hasta que el usuario lo cierre: no es algo pasajero
    actions: [{ label: 'Ver en GitHub', onClick: () => appApi.openUpdateUrl(info.url) }],
  });
}

onMounted(() => {
  // window.runtime todavia no existe corriendo el frontend suelto
  // (vite dev sin Wails, p.ej.): sin el try/catch, EventsOn tira un
  // error sincronico (ver el mismo patron en stores/theme.ts).
  try {
    onUpdateAvailable(notify);
  } catch {
    // noop: sin Wails no hay este evento que escuchar.
  }
});
</script>
