import './assets/css/main.css';

import ui from '@nuxt/ui/vue-plugin';
import { createPinia } from 'pinia';
import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import App from './App.vue';
import { useThemeStore } from './stores/theme';

const router = createRouter({
	routes: [],
	history: createWebHistory(),
});

const app = createApp(App).use(createPinia()).use(router).use(ui);

// Antes de montar: aplica la clase .light/.dark que main.css usa para
// elegir la paleta, para no arrancar con un flash del tema equivocado.
useThemeStore().init();

app.mount('#app');
