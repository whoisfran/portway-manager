import { onSystemThemeChanged } from '@/api/events';
import { defineStore } from 'pinia';
import { computed, ref, watchEffect } from 'vue';

export type ThemeMode = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'portway-theme';
const DEFAULT_MODE: ThemeMode = 'dark';

function isThemeMode(value: unknown): value is ThemeMode {
	return value === 'light' || value === 'dark' || value === 'system';
}

function readStoredMode(): ThemeMode {
	const stored = localStorage.getItem(STORAGE_KEY);
	return isThemeMode(stored) ? stored : DEFAULT_MODE;
}

/**
 * Tema claro/oscuro/sistema de la app, persistido en localStorage.
 * Por defecto es 'dark' (asi es como se penso la app), pero un usuario
 * en Fedora recien instalado (modo claro) puede elegir 'light' o
 * 'system' para que combine con su escritorio.
 *
 * Reemplaza al plugin de color-mode de Nuxt UI (ver vite.config.ts,
 * colorMode: false): ese usaba useDark() de VueUse, que en 'system'
 * solo lee prefers-color-scheme una vez al iniciar y no reacciona si
 * el usuario cambia el tema del SO despues sin reiniciar la app.
 *
 * En Linux (webkit2gtk) el matchMedia ni siquiera se entera del cambio
 * al volver a consultarlo -- por eso el backend empuja el evento
 * 'system:theme-changed' (ver systemtheme_linux.go) y aqui solo lo
 * escuchamos; en Windows/macOS el 'change' del matchMedia ya es
 * suficiente y llega solo.
 */
export const useThemeStore = defineStore('theme', () => {
	const mode = ref<ThemeMode>(readStoredMode());
	const systemPrefersDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches);

	const resolvedTheme = computed<'light' | 'dark'>(() => {
		if (mode.value === 'system') return systemPrefersDark.value ? 'dark' : 'light';
		return mode.value;
	});

	function setMode(next: ThemeMode) {
		mode.value = next;
		localStorage.setItem(STORAGE_KEY, next);
	}

	/**
	 * Aplica la clase .light/.dark al <html> (de ahi leen los tokens
	 * --ui-* en main.css) y arranca la deteccion de cambios de tema del
	 * SO. Se llama una sola vez desde main.ts, antes de montar la app.
	 */
	function init() {
		window
			.matchMedia('(prefers-color-scheme: dark)')
			.addEventListener('change', (e) => (systemPrefersDark.value = e.matches));

		// window.runtime todavia no existe corriendo el frontend suelto
		// (vite dev sin Wails, p.ej.): sin este try/catch, EventsOn tira
		// un error sincronico aqui mismo (antes del mount en main.ts) que
		// tumba el arranque de toda la app.
		try {
			onSystemThemeChanged((prefersDark) => (systemPrefersDark.value = prefersDark));
		} catch {
			// noop: sin Wails no hay forma de detectar el cambio en
			// caliente en Linux, pero el resto del tema sigue funcionando.
		}

		watchEffect(() => {
			const root = document.documentElement;
			root.classList.toggle('dark', resolvedTheme.value === 'dark');
			root.classList.toggle('light', resolvedTheme.value === 'light');
		});
	}

	return { mode, resolvedTheme, setMode, init };
});
