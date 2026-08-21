<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
    confirmLabel?: string;
    danger?: boolean;
  }>(),
  { confirmLabel: 'Confirmar', danger: false },
);

const open = defineModel<boolean>('open', { default: false });
const emit = defineEmits<{ confirm: [] }>();

function confirm(close: () => void) {
  close();
  emit('confirm');
}
</script>

<template>
  <UModal v-model:open="open" :title="title" :description="description" close>
    <template #footer="{ close }">
      <div class="flex w-full justify-end gap-2">
        <UButton label="Cancelar" color="neutral" variant="ghost" @click="close" />
        <UButton :label="confirmLabel" :color="danger ? 'error' : 'primary'" @click="confirm(close)" />
      </div>
    </template>
  </UModal>
</template>
