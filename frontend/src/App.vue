<script lang="ts" setup>
import ConnectionsWorkspace from '@/components/connections/ConnectionsWorkspace.vue';
import ProfileFormModal from '@/components/profiles/ProfileFormModal.vue';
import Layout from '@/layouts/Layout.vue';
import { useProfileUiStore } from '@/stores/profileUi';

const profileUi = useProfileUiStore();
</script>

<template>
  <UApp>
    <Layout>
      <ConnectionsWorkspace />
    </Layout>

    <ProfileFormModal v-model:open="profileUi.formOpen" :profile="profileUi.editingProfile" />
  </UApp>
</template>
