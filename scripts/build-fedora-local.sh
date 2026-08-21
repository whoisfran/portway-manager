#!/usr/bin/env bash
# Empaqueta un RPM de Fedora localmente, replicando los mismos pasos
# que .github/workflows/release-linux-fedora.yaml, sin necesitar CI
# ni un repo git. Pensado para probar el empaquetado antes de hacer
# un release real.
#
# Uso:
#   ./scripts/build-fedora-local.sh [version]
#
# version es opcional (default: 0.0.0-dev). No necesita el prefijo "v".
set -euo pipefail

VERSION="${1:-0.0.0-dev}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

for bin in wails rpmbuild jq; do
	command -v "$bin" >/dev/null || {
		echo "Falta '$bin' en el PATH." >&2
		exit 1
	}
done

# wails.json se modifica temporalmente (igual que en el workflow, que
# lo hace sobre un checkout descartable); aqui lo restauramos al salir
# para no dejar el working tree modificado por un build de prueba.
cp wails.json wails.json.bak
trap 'mv wails.json.bak wails.json 2>/dev/null || true' EXIT

jq --arg v "$VERSION" '.info.productVersion = $v' wails.json >wails.json.tmp
mv wails.json.tmp wails.json

echo "==> Compilando frontend + binario (wails build)"
CGO_ENABLED=1 wails build -platform linux/amd64 -tags webkit2_41 -o portway-manager

echo "==> Generando spec y .desktop desde las plantillas"
spec=$(cat build/linux/rpm/portway-manager.spec.template)
spec=$(echo "$spec" | sed -e "s/{{.Info.ProductVersion}}/$VERSION/g")
spec=$(echo "$spec" | sed -e "s/{{.Info.Comments}}/$(jq -r '.info.comments' wails.json)/g")
spec=$(echo "$spec" | sed -e "s/{{.Author.Name}}/$(jq -r '.author.name' wails.json)/g")
spec=$(echo "$spec" | sed -e "s/{{.libwebkit2gtk.PackageName}}/webkit2gtk4.1/g")
spec=$(echo "$spec" | sed -e "s/{{.ChangelogDate}}/$(LC_ALL=C date -u +'%a %b %d %Y')/g")
echo "$spec" >build/linux/rpm/portway-manager.spec

desktop=$(cat "build/linux/portway-manager_0.0.0_amd64/usr/share/applications/portway-manager.desktop")
desktop=$(echo "$desktop" | sed -e "s/{{.Info.ProductName}}/$(jq -r '.info.productName' wails.json)/g")
desktop=$(echo "$desktop" | sed -e "s/{{.Info.Comments}}/$(jq -r '.info.comments' wails.json)/g")
desktop=$(echo "$desktop" | sed -e "s#/usr/local/bin/portway-manager#/usr/bin/portway-manager#g")
echo "$desktop" >build/linux/portway-manager.desktop

echo "==> Empaquetando el RPM"
topdir="$ROOT_DIR/build/linux/rpmbuild"
rm -rf "$topdir"
mkdir -p "$topdir"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

install -m 0755 build/bin/portway-manager "$topdir/SOURCES/portway-manager"
install -m 0644 "build/linux/portway-manager_0.0.0_amd64/usr/share/icons/hicolor/512x512/apps/portway-manager.png" "$topdir/SOURCES/portway-manager.png"
install -m 0644 build/linux/portway-manager.desktop "$topdir/SOURCES/portway-manager.desktop"
cp build/linux/rpm/portway-manager.spec "$topdir/SPECS/portway-manager.spec"

rpmbuild --define "_topdir $topdir" -bb "$topdir/SPECS/portway-manager.spec"

output="build/linux/portway-manager_${VERSION}_amd64.rpm"
cp "$topdir"/RPMS/x86_64/*.rpm "$output"

echo
echo "Listo: $output"
echo
echo "Instalar:        sudo dnf install ./$output"
echo "Desinstalar:      sudo dnf remove portway-manager"
echo "Probar el binario sin empaquetar: ./build/bin/portway-manager"
