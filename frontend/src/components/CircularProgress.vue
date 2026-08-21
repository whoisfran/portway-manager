<script setup lang="ts">
defineOptions({ name: "CircularProgress", inheritAttrs: false })
</script>

<template>
  <div class="flex items-center justify-center">
    <svg v-bind="$attrs" class="w-10 h-10 text-blue-600 animate-spin" xmlns="http://w3.org" fill="none"
      viewBox="0 0 24 24">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" />
      <circle class="opacity-75" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" stroke-dasharray="80"
        stroke-dashoffset="24" />
    </svg>
  </div>
</template>
