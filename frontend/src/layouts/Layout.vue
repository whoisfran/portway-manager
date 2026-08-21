<script setup lang="ts">
import MainToolbar from '@/components/MainToolbar.vue';
import TitleBar from '@/components/TitleBar.vue';
import TunnelNotifier from '@/components/tunnels/TunnelNotifier.vue';
</script>

<template>
  <div class="flex h-full flex-col overflow-hidden bg-default text-default">
    <TitleBar />
    <MainToolbar />
    <slot />
  </div>

  <TunnelNotifier />
</template>
