<script setup lang="ts">
// Version inline del logo (ver build/appicon.svg) para que se vea
// nitido a tamanos chicos (la barra de titulo, etc.) sin depender de
// un asset rasterizado.
withDefaults(defineProps<{ size?: number }>(), { size: 24 });
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <linearGradient id="app-logo-bg" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#4f8cf7" />
        <stop offset="1" stop-color="#0f1a2b" />
      </linearGradient>
      <radialGradient id="app-logo-glow" cx="12" cy="14" r="7.5" gradientUnits="userSpaceOnUse">
        <stop offset="0" stop-color="#8fb6ff" stop-opacity="0.55" />
        <stop offset="1" stop-color="#4f8cf7" stop-opacity="0" />
      </radialGradient>
    </defs>

    <rect width="1024" height="1024" rx="208" fill="url(#app-logo-bg)" />

    <g transform="translate(176 176) scale(28)">
      <path d="M7 21.5V12A5 5 0 0 1 17 12V21.5Z" fill="url(#app-logo-glow)" />
      <path
        d="M2.5 20V12A9.5 9.5 0 0 1 21.5 12V20A1.5 1.5 0 0 1 20 21.5H18.5A1.5 1.5 0 0 1 17 20V12A5 5 0 0 0 7 12V20A1.5 1.5 0 0 1 5.5 21.5H4A1.5 1.5 0 0 1 2.5 20Z"
        fill="#f4f7fb"
      />
      <path
        d="M10 12.6L13.8 16L10 19.4"
        fill="none"
        stroke="#f4f7fb"
        stroke-width="2.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </g>
  </svg>
</template>
