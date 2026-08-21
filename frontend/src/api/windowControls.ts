import { HideToTray } from '@wailsjs/go/main/App';
import { Quit, WindowIsMaximised, WindowToggleMaximise } from '@wailsjs/runtime';

/** Adaptador sobre los controles de ventana nativos, para la barra de titulo personalizada (ventana sin marco). */
export const windowControlsApi = {
	// La ventana no tiene marco nativo (Frameless), asi que minimizarla
	// de verdad la dejaria sin forma de recuperarse: en su lugar se
	// oculta a la bandeja del sistema (ver tray.go), de donde se
	// recupera con un clic en el icono o en "Abrir" de su menu.
	minimise: (): Promise<void> => HideToTray(),
	toggleMaximise: (): void => WindowToggleMaximise(),
	isMaximised: (): Promise<boolean> => WindowIsMaximised(),
	close: (): void => Quit(),
};
