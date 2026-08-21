<script setup lang="ts">
import { usePortAvailability } from '@/composables/usePortAvailability';
import { useAwsEnvironmentStore } from '@/stores/awsEnvironment';
import { useProfileUiStore } from '@/stores/profileUi';
import { useProfilesStore } from '@/stores/profiles';
import type { ConnectionProfile } from '@/types/domain';
import { computed, reactive, ref, watch } from 'vue';

const props = defineProps<{ profile: ConnectionProfile | null }>();
const open = defineModel<boolean>('open', { default: false });

const profilesStore = useProfilesStore();
const profileUi = useProfileUiStore();
const awsEnvironment = useAwsEnvironmentStore();
const toast = useToast();

function emptyProfile(): ConnectionProfile {
  return {
    id: '',
    label: '',
    profile: '',
    region: '',
    instanceId: '',
    instanceLabel: '',
    localPort: 0,
    remotePort: 0,
    remoteHost: '',
  };
}

const state = reactive<ConnectionProfile>(emptyProfile());
const useRemoteHost = ref(false);
const saving = ref(false);
const formError = ref<string | null>(null);

const isEditing = computed(() => Boolean(props.profile?.id));

const localPortRef = computed(() => state.localPort || null);
const { status: portStatus, checking: checkingPort } = usePortAvailability(localPortRef);

const profileOptions = computed(() => awsEnvironment.profiles.map((p) => ({ label: p, value: p })));
const regionOptions = computed(() => awsEnvironment.regions.map((r) => ({ label: r, value: r })));
const instanceOptions = computed(() =>
  awsEnvironment.instances.map((i) => ({
    label: i.name ? `${i.name} — ${i.instanceId}` : i.instanceId,
    value: i.instanceId,
  })),
);

watch(
  () => open.value,
  (isOpen) => {
    if (!isOpen) return;

    Object.assign(state, props.profile ? { ...props.profile } : emptyProfile());
    useRemoteHost.value = Boolean(props.profile?.remoteHost);
    formError.value = null;

    if (awsEnvironment.profiles.length === 0) awsEnvironment.loadProfiles();
    if (awsEnvironment.regions.length === 0) awsEnvironment.loadRegions();

    // Solo para un perfil nuevo (no al editar ni duplicar, donde ya
    // hay un perfil/region que no queremos pisar): preselecciona con
    // lo que la AWS CLI usaria por defecto en este equipo.
    if (!props.profile) {
      prefillFromAwsDefaults();
    }
  },
  { immediate: true },
);

async function prefillFromAwsDefaults() {
  if (!awsEnvironment.defaults) {
    await awsEnvironment.loadDefaults();
  }
  if (!open.value || props.profile) return; // se cerro, o cambio a editar/duplicar mientras cargaba

  if (!state.profile && awsEnvironment.defaults?.profile) {
    state.profile = awsEnvironment.defaults.profile;
  }
  if (!state.region && awsEnvironment.defaults?.region) {
    state.region = awsEnvironment.defaults.region;
  }
}

// Carga las instancias solas en cuanto perfil y region quedan
// definidos -- por el default preseleccionado, al editar un perfil
// existente, o al cambiarlos a mano -- sin esperar a que el usuario
// le de al boton de refresh.
watch([() => state.profile, () => state.region], ([profile, region]) => {
  if (profile && region) {
    loadInstances();
  }
});

function selectInstance(instanceId: string) {
  state.instanceId = instanceId;
  state.instanceLabel = awsEnvironment.instances.find((i) => i.instanceId === instanceId)?.name || instanceId;
}

function loadInstances() {
  if (!state.profile || !state.region) return;
  return awsEnvironment.loadInstances(state.profile, state.region);
}

function validate(): string | null {
  if (!state.label.trim()) return 'El perfil debe tener un nombre.';
  if (!state.instanceId) return 'Selecciona una instancia.';
  if (!state.localPort || state.localPort <= 0) return 'El puerto local es invalido.';
  if (!state.remotePort || state.remotePort <= 0) return 'El puerto remoto es invalido.';
  return null;
}

async function submit() {
  formError.value = validate();
  if (formError.value) return;

  if (!useRemoteHost.value) {
    state.remoteHost = '';
  }

  saving.value = true;
  try {
    const saved = await profilesStore.save({ ...state });
    profileUi.selectProfile(saved.id);
    toast.add({ title: isEditing.value ? 'Perfil actualizado' : 'Perfil creado', color: 'success' });
    open.value = false;
  } catch (err) {
    formError.value = (err as Error).message;
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="isEditing ? 'Editar perfil' : 'Nuevo perfil'" close>
    <template #body>
      <div class="flex flex-col gap-4">
        <UFormField label="Nombre" required>
          <UInput v-model="state.label" placeholder="ej. DB producción" class="w-full" />
        </UFormField>

        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Perfil de AWS" required>
            <USelect v-model="state.profile" :items="profileOptions" :loading="awsEnvironment.loadingProfiles"
              placeholder="Selecciona un perfil" class="w-full" />
          </UFormField>
          <UFormField label="Región" required>
            <USelect v-model="state.region" :items="regionOptions" placeholder="Selecciona una región" class="w-full" />
          </UFormField>
        </div>

        <UFormField label="Instancia" required>
          <div class="flex gap-2">
            <USelect :model-value="state.instanceId" :items="instanceOptions" :loading="awsEnvironment.loadingInstances"
              :disabled="!state.profile || !state.region" placeholder="Selecciona una instancia" class="w-full"
              @update:model-value="selectInstance" />
            <UButton icon="i-lucide-refresh-cw" color="neutral" variant="soft"
              :loading="awsEnvironment.loadingInstances" :disabled="!state.profile || !state.region"
              @click="loadInstances" />
          </div>
          <p v-if="awsEnvironment.instancesError" class="mt-1 text-xs text-error">{{ awsEnvironment.instancesError }}
          </p>
        </UFormField>

        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Puerto local" required>
            <UInputNumber v-model="state.localPort" :min="1" :max="65535" class="w-full" />
            <p v-if="checkingPort" class="mt-1 text-xs text-muted">Comprobando puerto…</p>
            <p v-else-if="portStatus && !portStatus.available && portStatus.inUseBySameApp"
              class="mt-1 text-xs text-warning">
              En uso por el túnel «{{ portStatus.conflictLabel }}». Podrías detenerlo antes de iniciar este.
            </p>
            <p v-else-if="portStatus && !portStatus.available" class="mt-1 text-xs text-error">
              Puerto ocupado por el sistema. Elige otro.
            </p>
          </UFormField>
          <UFormField label="Puerto remoto" required>
            <UInputNumber v-model="state.remotePort" :min="1" :max="65535" class="w-full" />
          </UFormField>
        </div>

        <UFormField>
          <USwitch v-model="useRemoteHost" color="primary" variant="soft"
            label="Túnel hacia un host remoto distinto de la instancia" />
        </UFormField>
        <UFormField v-if="useRemoteHost" label="Host remoto">
          <UInput v-model="state.remoteHost" placeholder="ej. mi-base.internal" class="w-full" />
        </UFormField>

        <p v-if="formError" class="text-sm text-error">{{ formError }}</p>
      </div>
    </template>

    <template #footer="{ close }">
      <div class="flex w-full justify-end gap-2">
        <UButton label="Cancelar" color="neutral" variant="soft" @click="close" />
        <UButton label="Guardar" variant="soft" :loading="saving" @click="submit" />
      </div>
    </template>
  </UModal>
</template>
