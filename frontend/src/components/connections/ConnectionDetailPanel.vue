<script setup lang="ts">
import ConfirmModal from '@/components/shared/ConfirmModal.vue';
import StatusBadge from '@/components/shared/StatusBadge.vue';
import { regionLabel } from '@/constants/awsRegions';
import { useAwsEnvironmentStore } from '@/stores/awsEnvironment';
import { useProfileUiStore } from '@/stores/profileUi';
import { useProfilesStore } from '@/stores/profiles';
import { useTunnelsStore } from '@/stores/tunnels';
import { computed, ref, watch } from 'vue';

const profilesStore = useProfilesStore();
const tunnelsStore = useTunnelsStore();
const profileUi = useProfileUiStore();
const awsEnvironment = useAwsEnvironmentStore();
const toast = useToast();

const profile = computed(() => profileUi.selectedProfile);
const activeTunnel = computed(() => (profile.value ? tunnelsStore.findFor(profile.value) : undefined));
const logLines = computed(() => (activeTunnel.value ? tunnelsStore.logsByTunnel[activeTunnel.value.id] ?? [] : []));

const starting = ref(false);
const stopping = ref(false);
const confirmingDelete = ref(false);

const typeLabel = computed(() => (profile.value?.type === 'ssh' ? 'SSH' : 'SSM'));

const destination = computed(() => {
  if (!profile.value) return '';

  if (profile.value.type === 'ssh') {
    const target = profile.value.remoteHost
      ? `${profile.value.remoteHost}:${profile.value.remotePort}`
      : `localhost:${profile.value.remotePort}`;
    const source = profile.value.user ? `${profile.value.user}@${profile.value.host}` : profile.value.host;
    return `${source} · :${profile.value.localPort} → ${target}`;
  }

  const target = profile.value.remoteHost
    ? `${profile.value.remoteHost}:${profile.value.remotePort}`
    : `${profile.value.instanceId}:${profile.value.remotePort}`;
  return `${profile.value.instanceLabel || profile.value.instanceId} · :${profile.value.localPort} → ${target}`;
});

const authMethod = computed(() =>
  profile.value?.type === 'ssm' ? awsEnvironment.authMethods[profile.value.profile ?? ''] : undefined,
);

// Solo tiene sentido resolver el metodo de autenticacion de AWS (SSO,
// rol asumido, claves...) para perfiles SSM; una conexion SSH no tiene
// un "perfil de AWS" del que pedirlo.
watch(
  () => (profile.value?.type === 'ssm' ? profile.value?.profile : undefined),
  (awsProfile) => {
    if (awsProfile !== undefined) awsEnvironment.loadAuthMethod(awsProfile);
  },
  { immediate: true },
);

type DetailTile = { icon: string; label: string; value: string };

// Que datos mostrar en la grilla de detalle depende del tipo de
// conexion: un mismo layout (icono + etiqueta + valor) reutilizado
// con contenido distinto, en vez de duplicar el markup por tipo.
const detailTiles = computed<DetailTile[]>(() => {
  if (!profile.value) return [];

  if (profile.value.type === 'ssh') {
    return [
      { icon: 'i-lucide-server', label: 'Host', value: profile.value.host || '—' },
      { icon: 'i-lucide-plug-2', label: 'Puerto SSH', value: String(profile.value.port || 22) },
      { icon: 'i-lucide-user', label: 'Usuario', value: profile.value.user || '—' },
      {
        icon: 'i-lucide-key',
        label: 'Autenticación',
        value: profile.value.authMethod === 'privateKey' ? 'Llave privada' : 'Contraseña',
      },
      { icon: 'i-lucide-clock', label: 'Última conexión', value: formatLastConnected(profile.value.lastConnectedAt) },
    ];
  }

  return [
    { icon: 'i-lucide-cloud', label: 'Región', value: regionLabel(profile.value.region ?? '') },
    { icon: 'i-lucide-wrench', label: 'Método', value: authMethod.value ?? '—' },
    { icon: 'i-lucide-user', label: 'Perfil', value: profile.value.profile || 'Por defecto' },
    { icon: 'i-lucide-clock', label: 'Última conexión', value: formatLastConnected(profile.value.lastConnectedAt) },
  ];
});

function formatLastConnected(iso: string | undefined): string {
  if (!iso) return 'Nunca';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return 'Nunca';
  return new Intl.DateTimeFormat('es', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function logColor(line: string): string {
  const lower = line.toLowerCase();
  if (lower.includes('error') || lower.includes('fall')) return 'text-red-400';
  if (lower.includes('iniciado') || lower.includes('correctamente') || lower.includes('exitosa')) return 'text-green-400';
  if (lower.includes('conectando') || lower.includes('iniciando')) return 'text-blue-400';
  return 'text-muted';
}

async function start() {
  if (!profile.value) return;
  starting.value = true;
  try {
    await tunnelsStore.start(profile.value.id);
    await profilesStore.markConnected(profile.value.id);
  } catch (err) {
    toast.add({ title: 'No se pudo iniciar el túnel', description: (err as Error).message, color: 'error' });
  } finally {
    starting.value = false;
  }
}

async function stop() {
  if (!activeTunnel.value) return;
  stopping.value = true;
  try {
    await tunnelsStore.stop(activeTunnel.value.id);
  } catch (err) {
    toast.add({ title: 'No se pudo detener el túnel', description: (err as Error).message, color: 'error' });
  } finally {
    stopping.value = false;
  }
}

async function remove() {
  if (!profile.value) return;
  try {
    await profilesStore.remove(profile.value.id);
  } catch (err) {
    toast.add({ title: 'No se pudo eliminar el perfil', description: (err as Error).message, color: 'error' });
  }
}

function clearLogs() {
  if (activeTunnel.value) tunnelsStore.clearLogs(activeTunnel.value.id);
}
</script>

<template>
  <section v-if="profile" class="flex min-w-0 flex-1 flex-col overflow-hidden">
    <div class="flex items-start justify-between gap-3 border-b border-default px-4 py-3">
      <div class="flex min-w-0 flex-col gap-1">
        <div class="flex items-center gap-2">
          <h2 class="truncate text-lg font-semibold">{{ profile.label }}</h2>
          <UBadge :label="typeLabel" color="neutral" variant="subtle" size="sm" />
          <StatusBadge :profile="profile" :tunnel="activeTunnel" />
        </div>
        <p class="truncate font-mono-data text-xs text-muted">{{ destination }}</p>
      </div>

      <div class="flex shrink-0 items-center gap-1.5">
        <UButton v-if="!activeTunnel" icon="i-lucide-play" variant="soft" color="primary" size='sm' :loading="starting"
          @click="start" />
        <UButton v-else icon="i-lucide-square" color="error" variant="soft" size='sm' :loading="stopping"
          @click="stop" />
        <UButton icon="i-lucide-pencil" color="neutral" variant="soft" size='sm' aria-label="Editar"
          @click="profileUi.openEdit(profile)" />
        <UButton icon="i-lucide-copy" color="neutral" variant="soft" size='sm' aria-label="Duplicar"
          @click="profileUi.openDuplicate(profile)" />
        <UButton icon="i-lucide-trash-2" color="error" variant="soft" size='sm' aria-label="Eliminar"
          @click="confirmingDelete = true" />
      </div>
    </div>

    <div class="grid grid-cols-2 gap-4 border-b border-default px-4 py-4 text-sm">
      <div v-for="tile in detailTiles" :key="tile.label" class="flex items-start gap-2">
        <UIcon :name="tile.icon" class="mt-0.5 size-4 shrink-0 text-muted" />
        <div class="flex flex-col">
          <span class="text-xs text-muted">{{ tile.label }}</span>
          <span>{{ tile.value }}</span>
        </div>
      </div>
    </div>

    <div class="flex min-h-0 flex-1 flex-col px-4 py-3">
      <div class="mb-2 flex items-center justify-between">
        <h3 class="text-sm font-semibold">Logs</h3>
        <UButton label="Limpiar" icon="i-lucide-trash" color="neutral" variant="soft" size="xs"
          :disabled="logLines.length === 0" @click="clearLogs" />
      </div>

      <div class="flex-1 overflow-y-auto rounded-md bg-default/60 p-2">
        <p v-if="logLines.length === 0" class="font-mono-data text-xs text-dimmed">Sin actividad en el log todavía.</p>
        <p v-for="(line, index) in logLines" :key="index" class="whitespace-pre-wrap font-mono-data text-xs"
          :class="logColor(line)">
          {{ line }}
        </p>
      </div>
    </div>
  </section>

  <section v-else class="flex flex-1 flex-col items-center justify-center gap-2 text-muted">
    <UIcon name="i-lucide-cable" class="size-8" />
    <span class="font-mono-data text-xs font-semibold select-none">Selecciona o crea una conexión</span>
  </section>

  <ConfirmModal v-if="profile" v-model:open="confirmingDelete" title="Eliminar perfil"
    :description="`Se eliminará «${profile.label}» de la lista de conexiones guardadas.`" confirm-label="Eliminar"
    danger @confirm="remove" />
</template>
