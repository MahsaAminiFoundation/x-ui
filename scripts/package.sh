#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$REPO_DIR/dist"
PKG_NAME="x-ui-linux-amd64"
DOCKER_IMAGE="x-ui-build"
DOCKER_CONTAINER="x-ui-build-tmp"

echo "==> Cleaning dist/"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/$PKG_NAME/bin"

echo "==> Building Go binary via Docker"
docker build --platform linux/amd64 -t "$DOCKER_IMAGE" "$REPO_DIR"

echo "==> Extracting binary from Docker image"
docker create --name "$DOCKER_CONTAINER" "$DOCKER_IMAGE"
docker cp "$DOCKER_CONTAINER:/usr/local/x-ui/x-ui" "$DIST_DIR/$PKG_NAME/x-ui"
docker rm "$DOCKER_CONTAINER"

echo "==> Copying support files"
cp "$REPO_DIR/x-ui.service"              "$DIST_DIR/$PKG_NAME/"
cp "$REPO_DIR/x-ui.sh"                   "$DIST_DIR/$PKG_NAME/"
cp "$REPO_DIR/crontab/mahsa_amini_vpn"   "$DIST_DIR/$PKG_NAME/"
cp "$REPO_DIR/bin/geoip.dat"             "$DIST_DIR/$PKG_NAME/bin/"
cp "$REPO_DIR/bin/geosite.dat"           "$DIST_DIR/$PKG_NAME/bin/"
cp "$REPO_DIR/bin/iran.dat"              "$DIST_DIR/$PKG_NAME/bin/"
cp "$REPO_DIR/bin/xray-linux-amd64"     "$DIST_DIR/$PKG_NAME/bin/"

echo "==> Creating tarball"
cd "$DIST_DIR"
tar czvf "${PKG_NAME}.tar.gz" "$PKG_NAME/"

echo ""
echo "Done: $DIST_DIR/${PKG_NAME}.tar.gz"
echo ""
echo "Deploy:"
echo "  scp $DIST_DIR/${PKG_NAME}.tar.gz root@mahsa270cv4h.zahedan.buzz:~"
echo "  ssh root@mahsa270cv4h.zahedan.buzz 'cd ~ && tar xzf ${PKG_NAME}.tar.gz && systemctl stop x-ui && cp -r ${PKG_NAME}/. /usr/local/x-ui/ && systemctl start x-ui'"
