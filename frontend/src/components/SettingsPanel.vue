<script setup lang="ts">
// Placeholder: todavia no hay nada que configurar en la app. Existe
// para no tener que armar el modal desde cero el dia que se agregue
// la primera opcion real (tema, ubicacion del archivo de perfiles,
// etc.); ver el boton comentado en MainToolbar.vue.
const open = defineModel<boolean>('open', { default: false });
</script>

<template>
  <UModal v-model:open="open" title="Ajustes" close>
    <template #body>
      <p class="text-sm text-muted">Todavía no hay ajustes configurables.</p>
    </template>
  </UModal>
</template>
