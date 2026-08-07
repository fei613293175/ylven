#!/usr/bin/env bash
set -Eeuo pipefail

[[ $# -eq 8 ]] || {
  echo "usage: $0 <workspace> <artifact-dir> <gradle-cache> <signing-dir> <previous-apk> <phase> <version> <commit>" >&2
  exit 2
}
workspace="$1"
artifact_dir="$2"
gradle_cache="$3"
signing_dir="$4"
previous_apk="$5"
phase="$6"
version="$7"
commit="$8"

for value in "$workspace" "$artifact_dir" "$gradle_cache" "$signing_dir" "$previous_apk"; do
  [[ "$value" == /* && "$value" != *$'\n'* ]] || { echo "invalid absolute server path" >&2; exit 2; }
done
[[ "$phase" =~ ^P[0-9]{2}$ ]] || { echo "invalid phase" >&2; exit 2; }
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "invalid version" >&2; exit 2; }
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid commit" >&2; exit 2; }
[[ -f "$workspace/source-archive.sha256" ]] || { echo "missing source archive digest" >&2; exit 2; }
[[ -f "$previous_apk" ]] || { echo "missing previous APK" >&2; exit 2; }
[[ -f "$signing_dir/owner-signing.env" ]] || {
  echo "missing private owner signing configuration on the server" >&2
  exit 3
}
chmod 0600 "$signing_dir/owner-signing.env"
mkdir -p "$artifact_dir" "$gradle_cache"
source_sha="$(tr -d '\r\n ' < "$workspace/source-archive.sha256")"
builder_image="orbexa/android-builder:gradle8.9-api35-v1"
container_artifact_dir="/ylven-artifacts"

android-build docker-run --project ylven --kind build --queue-timeout 21600 -- \
  --mount "type=bind,src=$workspace,dst=/workspace" \
  --mount "type=bind,src=$artifact_dir,dst=$container_artifact_dir" \
  --mount "type=bind,src=$gradle_cache,dst=/root/.gradle" \
  --mount "type=bind,src=$signing_dir,dst=/run/ylven-signing,readonly" \
  --mount "type=bind,src=$previous_apk,dst=/inputs/previous.apk,readonly" \
  --env-file "$signing_dir/owner-signing.env" \
  --env ANDROID_BUILD_COORDINATED=1 \
  --env ANDROID_BUILD_RUN_ID \
  --env YLVEN_PHASE="$phase" \
  --env YLVEN_VERSION="$version" \
  --env YLVEN_COMMIT="$commit" \
  --env YLVEN_SOURCE_ARCHIVE_SHA256="$source_sha" \
  --env YLVEN_PREVIOUS_APK=/inputs/previous.apk \
  --env YLVEN_ARTIFACT_ROOT="$container_artifact_dir" \
  --env YLVEN_SIGNING_STORE_FILE=/run/ylven-signing/owner.keystore \
  --env YLVEN_BUILDER_IMAGE="$builder_image" \
  --workdir /workspace \
  "$builder_image" \
  bash scripts/29_BUILD_ANDROID_ON_SERVER.sh
