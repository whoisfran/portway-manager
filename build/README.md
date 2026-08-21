# Build Directory

The build directory is used to house all the build files and assets for your application. 

The structure is:

* bin - Output directory
* darwin - macOS specific files
* windows - Windows specific files
* linux - Linux specific files (empaquetado RPM para Fedora)
* appicon.svg / appicon.png - Fuente del logo (ver "Icono" mas abajo)

## Mac

The `darwin` directory holds files specific to Mac builds.
These may be customised and used as part of the build. To return these files to the default state, simply delete them
and
build with `wails build`.

The directory contains the following files:

- `Info.plist` - the main plist file used for Mac builds. It is used when building using `wails build`.
- `Info.dev.plist` - same as the main plist file but used when building using `wails dev`.

## Windows

The `windows` directory contains the manifest and rc files used when building with `wails build`.
These may be customised for your application. To return these files to the default state, simply delete them and
build with `wails build`.

- `icon.ico` - The icon used for the application. This is used when building using `wails build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer. These are used when building using `wails build`.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.

## Linux

A diferencia de macOS/Windows, `wails build` no empaqueta nada para Linux
por si solo (solo produce el binario). El paquete RPM para Fedora se
arma a mano, orquestado por `.github/workflows/release-linux-fedora.yaml`:

- `linux/rpm/ssm-portway.spec.template` - Plantilla del spec de RPM. El
  workflow reemplaza los tokens `{{.Info.*}}`/`{{.Author.*}}` (leidos de
  `wails.json`) y los guarda como `linux/rpm/ssm-portway.spec` antes de
  correr `rpmbuild`. Ese `.spec` generado no se versiona (se regenera en
  cada build).
- `linux/ssm-portway_0.0.0_amd64/usr/share/applications/ssm-portway.desktop` -
  Plantilla del `.desktop` (mismo mecanismo de tokens). El nombre de la
  carpeta padre (`ssm-portway_0.0.0_amd64`) es solo la convencion de
  layout FHS que usa el workflow para ubicar el icono; no significa que
  exista una version "0.0.0" real.
- `linux/ssm-portway_0.0.0_amd64/usr/share/icons/hicolor/512x512/apps/ssm-portway.png` -
  Icono en 512x512, generado desde `appicon.svg`.

Para probar el empaquetado en un Fedora real (fuera de CI):

```bash
wails build -platform linux/amd64 -tags webkit2_41 -o ssm-portway
# rellenar los tokens del .spec/.desktop a mano o adaptar los pasos
# "Setup spec template"/"Setup desktop template" del workflow, luego:
rpmbuild --define "_topdir $PWD/build/linux/rpmbuild" -bb build/linux/rpm/ssm-portway.spec
```

## Icono

`appicon.svg` es la fuente vectorial del logo; todo lo demas (`appicon.png`,
`windows/icon.ico`, el PNG de Linux) se genera a partir de el. Si cambias el
logo, regenera los tres:

```bash
convert -background none -density 384 build/appicon.svg -resize 1024x1024 build/appicon.png

for s in 16 32 48 64 128 256; do
  convert -background none -density $((s*3)) build/appicon.svg -resize ${s}x${s} /tmp/icon_${s}.png
done
convert /tmp/icon_{16,32,48,64,128,256}.png build/windows/icon.ico

convert -background none -density 384 build/appicon.svg -resize 512x512 \
  build/linux/ssm-portway_0.0.0_amd64/usr/share/icons/hicolor/512x512/apps/ssm-portway.png
```

macOS no necesita un paso manual: `wails build` genera el `.icns` a partir
de `appicon.png` automaticamente.