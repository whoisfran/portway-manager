<script setup lang="ts">
// Version inline del logo (ver build/appicon.svg) para que se vea
// nitido a tamanos chicos (la barra de titulo, etc.) sin depender de
// un asset rasterizado.
withDefaults(defineProps<{ size?: number }>(), { size: 24 });
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <linearGradient id="app-logo-bg" x1="0" y1="0" x2="0" y2="512" gradientUnits="userSpaceOnUse">
        <stop offset="0" stop-color="#4f8cf7" />
        <stop offset="1" stop-color="#0f1a2b" />
      </linearGradient>
    </defs>

    <rect width="512" height="512" rx="104" fill="url(#app-logo-bg)" />

    <path
      d="M 166 366 L 166 256 A 90 90 0 0 1 346 256 L 346 366 Z"
      fill="#f4f7fb"
    />
    <path
      d="M 202 366 L 202 256 A 54 54 0 0 1 310 256 L 310 366 Z"
      fill="url(#app-logo-bg)"
    />

    <path
      d="M 234 292 L 256 314 L 234 336"
      fill="none"
      stroke="#f4f7fb"
      stroke-width="14"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M 270 292 L 292 314 L 270 336"
      fill="none"
      stroke="#f4f7fb"
      stroke-width="14"
      stroke-linecap="round"
      stroke-linejoin="round"
      opacity="0.85"
    />
  </svg>
</template>
