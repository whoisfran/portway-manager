import './assets/css/main.css';

import ui from '@nuxt/ui/vue-plugin';
import { createPinia } from 'pinia';
import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import App from './App.vue';

const router = createRouter({
	routes: [],
	history: createWebHistory(),
});

createApp(App).use(createPinia()).use(router).use(ui).mount('#app');
