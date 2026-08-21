import ui from '@nuxt/ui/vite';
import vue from '@vitejs/plugin-vue';
import path from 'node:path';
import { defineConfig } from 'vite';

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [
		vue(),
		ui({
			ui: {
				colors: {
					primary: 'blue',
					neutral: 'zinc',
				},
				toast: {
					slots: {
						title: 'text-sm font-semibold',
						description: 'text-xs text-muted',
					},
				},
			},
			icon: {
				clientBundle: { scan: true },
			},
		}),
	],
	resolve: {
		alias: {
			'@': path.resolve(__dirname, 'src'),
			'@/assets': path.resolve(__dirname, 'src/assets'),
			'@/layouts': path.resolve(__dirname, 'src/layouts'),
			'@/components': path.resolve(__dirname, 'src/components'),
			'@/composables': path.resolve(__dirname, 'src/composables'),
			'@/types': path.resolve(__dirname, 'src/types'),
			'@wailsjs': path.resolve(__dirname, 'wailsjs'),
		},
	},
});
