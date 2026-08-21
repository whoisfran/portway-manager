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
    type: 'ssm',
    localPort: 0,
    remotePort: 0,
    remoteHost: '',
    profile: '',
    region: '',
    instanceId: '',
    instanceLabel: '',
    host: '',
    port: 22,
    user: '',
    authMethod: 'password',
    password: '',
    privateKeyPath: '',
    passphrase: '',
  };
}

const state = reactive<ConnectionProfile>(emptyProfile());
const useRemoteHost = ref(false);
const saving = ref(false);
const pickingKey = ref(false);
const formError = ref<string | null>(null);

const isEditing = computed(() => Boolean(props.profile?.id));

const localPortRef = computed(() => state.localPort || null);
const { status: portStatus, checking: checkingPort } = usePortAvailability(localPortRef);

// value queda como string abierto (no ConnectionType/SSHAuthMethod)
// para que coincida con el tipo de state.type/state.authMethod, que
// tambien es string abierto -- ver el comentario en types/domain.ts.
const typeOptions: Array<{ label: string; value: string }> = [
  { label: 'AWS Systems Manager (SSM)', value: 'ssm' },
  { label: 'SSH', value: 'ssh' },
];
const authMethodOptions: Array<{ label: string; value: string }> = [
  { label: 'Contraseña', value: 'password' },
  { label: 'Llave privada', value: 'privateKey' },
];

const profileOptions = computed(() => awsEnvironment.profiles.map((p) => ({ label: p, value: p })));
const regionOptions = computed(() => awsEnvironment.regions.map((r) => ({ label: r, value: r })));
const instanceOptions = computed(() =>
  awsEnvironment.instances.map((i) => ({
    label: i.name ? `${i.name} — ${i.instanceId}` : i.instanceId,
    value: i.instanceId,
  })),
);

const remoteHostToggleLabel = computed(() =>
  state.type === 'ssh'
    ? 'Túnel hacia un host remoto distinto del servidor SSH'
    : 'Túnel hacia un host remoto distinto de la instancia',
);

watch(
  () => open.value,
  (isOpen) => {
    if (!isOpen) return;

    // Primero se limpia a los valores por defecto y luego se
    // sobreescriben con el perfil a editar: si no se hiciera asi, al
    // editar un perfil de un tipo justo despues de haber editado uno
    // de otro tipo, campos como host/user (SSH) quedarian con el
    // valor del perfil anterior en vez de vacios (esos campos ni
    // siquiera vienen en el JSON de un favorito SSM, por lo que un
    // Object.assign directo nunca los habria limpiado).
    Object.assign(state, emptyProfile(), props.profile ? { ...props.profile } : {});
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

// Si el usuario cambia a SSH sin haber tocado el puerto todavia,
// parte de 22 (el default real de un servidor SSH) en vez de 0.
watch(
  () => state.type,
  (type) => {
    if (type === 'ssh' && !state.port) {
      state.port = 22;
    }
  },
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

async function pickPrivateKey() {
  pickingKey.value = true;
  try {
    const path = await profilesStore.pickPrivateKeyFile();
    if (path) state.privateKeyPath = path;
  } catch (err) {
    formError.value = (err as Error).message;
  } finally {
    pickingKey.value = false;
  }
}

function validate(): string | null {
  if (!state.label.trim()) return 'El perfil debe tener un nombre.';
  if (!state.localPort || state.localPort <= 0) return 'El puerto local es inválido.';
  if (!state.remotePort || state.remotePort <= 0) return 'El puerto remoto es inválido.';

  if (state.type === 'ssm') {
    if (!state.instanceId) return 'Selecciona una instancia.';
    return null;
  }

  if (!state.host?.trim()) return 'Indica el host del servidor SSH.';
  if (!state.user?.trim()) return 'Indica el usuario de la conexión SSH.';

  if (state.authMethod === 'password') {
    // Al editar, un campo vacio significa "no la cambies" (el backend
    // nunca la devuelve, ver ConnectionProfile.password) -- solo es
    // obligatoria al crear un perfil nuevo.
    if (!state.password && !isEditing.value) return 'Indica la contraseña de la conexión SSH.';
  } else if (state.authMethod === 'privateKey') {
    if (!state.privateKeyPath?.trim()) return 'Selecciona el archivo de la llave privada.';
  } else {
    return 'Selecciona un método de autenticación.';
  }
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
        <UFormField label="Tipo de conexión" required>
          <USelect v-model="state.type" :items="typeOptions" class="w-full" />
        </UFormField>

        <UFormField label="Nombre" required>
          <UInput v-model="state.label" placeholder="ej. DB producción" class="w-full" />
        </UFormField>

        <template v-if="state.type === 'ssm'">
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
        </template>

        <template v-else>
          <UFormField label="Host" required>
            <UInput v-model="state.host" placeholder="ej. bastion.midominio.com" class="w-full" />
          </UFormField>

          <div class="grid grid-cols-2 gap-4">
            <UFormField label="Puerto SSH">
              <UInputNumber v-model="state.port" :min="1" :max="65535" class="w-full" />
            </UFormField>
            <UFormField label="Usuario" required>
              <UInput v-model="state.user" placeholder="ej. ec2-user" class="w-full" />
            </UFormField>
          </div>

          <UFormField label="Método de autenticación" required>
            <USelect v-model="state.authMethod" :items="authMethodOptions" class="w-full" />
          </UFormField>

          <UFormField v-if="state.authMethod === 'password'" label="Contraseña" :required="!isEditing">
            <UInput v-model="state.password" type="password" :placeholder="isEditing ? 'Sin cambios' : ''"
              class="w-full" />
            <p v-if="isEditing" class="mt-1 text-xs text-muted">Déjala en blanco para conservar la contraseña ya
              guardada.</p>
          </UFormField>

          <template v-else>
            <UFormField label="Llave privada" required>
              <div class="flex gap-2">
                <UInput :model-value="state.privateKeyPath" readonly placeholder="Selecciona un archivo…"
                  class="w-full" />
                <UButton label="Buscar…" icon="i-lucide-folder-open" color="neutral" variant="soft"
                  :loading="pickingKey" @click="pickPrivateKey" />
              </div>
            </UFormField>
            <UFormField label="Passphrase (opcional)">
              <UInput v-model="state.passphrase" type="password" :placeholder="isEditing ? 'Sin cambios' : ''"
                class="w-full" />
              <p v-if="isEditing" class="mt-1 text-xs text-muted">Déjala en blanco para conservar la ya guardada.</p>
            </UFormField>
          </template>
        </template>

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
          <USwitch v-model="useRemoteHost" color="primary" variant="soft" :label="remoteHostToggleLabel" />
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
