#!/usr/bin/env bash
set -Eeuo pipefail

: "${ANDROID_BUILD_COORDINATED:?server build must hold the shared Android build coordinator lease}"
: "${ANDROID_BUILD_RUN_ID:?missing coordinator run ID}"
: "${YLVEN_PHASE:?missing phase}"
: "${YLVEN_VERSION:?missing version}"
: "${YLVEN_COMMIT:?missing commit SHA}"
: "${YLVEN_SOURCE_ARCHIVE_SHA256:?missing source archive SHA-256}"
: "${YLVEN_PREVIOUS_APK:?missing previous owner APK path}"
: "${YLVEN_ARTIFACT_ROOT:=/artifacts}"

[[ "$YLVEN_PHASE" =~ ^P[0-9]{2}$ ]] || { echo "invalid phase" >&2; exit 2; }
[[ "$YLVEN_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "invalid version" >&2; exit 2; }
[[ "$YLVEN_COMMIT" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid commit SHA" >&2; exit 2; }
[[ "$YLVEN_SOURCE_ARCHIVE_SHA256" =~ ^[0-9a-f]{64}$ ]] || { echo "invalid source archive SHA-256" >&2; exit 2; }
[[ -f "$YLVEN_PREVIOUS_APK" ]] || { echo "previous owner APK is missing" >&2; exit 2; }
[[ "$(java -version 2>&1 | sed -n '1p')" == *'21.'* ]] || { echo "JDK 21 is required" >&2; exit 2; }
command -v gradle >/dev/null || { echo "Gradle is missing" >&2; exit 2; }
command -v apksigner >/dev/null || { echo "apksigner is missing" >&2; exit 2; }
command -v aapt >/dev/null || { echo "aapt is missing" >&2; exit 2; }

mkdir -p "$YLVEN_ARTIFACT_ROOT"
test -z "$(find "$YLVEN_ARTIFACT_ROOT" -mindepth 1 -maxdepth 1 -print -quit)" || {
  echo "artifact directory must be empty" >&2
  exit 2
}

IFS='.' read -r major minor patch <<< "$YLVEN_VERSION"
expected_code=$((major * 1000000 + minor * 10000 + patch * 100))
build_log="$YLVEN_ARTIFACT_ROOT/server-android-build.log"
toolchain_log="$YLVEN_ARTIFACT_ROOT/server-toolchain.txt"
{
  java -version
  gradle --version
  printf 'android_home=%s\n' "${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}"
  printf 'aapt=%s\n' "$(command -v aapt)"
  printf 'apksigner=%s\n' "$(command -v apksigner)"
} >"$toolchain_log" 2>&1

set -o pipefail
gradle --no-daemon --max-workers=2 \
  -PylvenVersionName="$YLVEN_VERSION" \
  -PylvenApiBaseUrl="${YLVEN_API_BASE_URL:-https://ai-admin.orbexa.cc}" \
  -PylvenRequireOwnerSigning=true \
  clean testDebugUnitTest lintDebug assembleDebug assembleDebugAndroidTest \
  2>&1 | tee "$build_log"

app_apk="android/app/build/outputs/apk/debug/app-debug.apk"
test_apk="android/app/build/outputs/apk/androidTest/debug/app-debug-androidTest.apk"
[[ -f "$app_apk" ]] || { echo "owner APK was not built" >&2; exit 3; }
[[ -f "$test_apk" ]] || { echo "instrumentation APK was not built" >&2; exit 3; }

badging="$(aapt dump badging "$app_apk")"
grep -q "package: name='cc.orbexa.ylven' versionCode='$expected_code' versionName='$YLVEN_VERSION'" <<<"$badging" || {
  echo "APK identity mismatch; expected cc.orbexa.ylven $YLVEN_VERSION/$expected_code" >&2
  sed -n '1p' <<<"$badging" >&2
  exit 3
}
certificate_digest() {
  apksigner verify --print-certs "$1" 2>/dev/null |
    sed -n 's/.*certificate SHA-256 digest: //p' | head -n 1
}
current_certificate="$(certificate_digest "$app_apk")"
previous_certificate="$(certificate_digest "$YLVEN_PREVIOUS_APK")"
[[ -n "$current_certificate" && -n "$previous_certificate" ]] || {
  echo "could not read APK signing certificate" >&2
  exit 3
}
signing_migration=false
signing_migration_install_mode="adb_install_r"
signing_migration_data_preserved=true
if [[ "$current_certificate" != "$previous_certificate" ]]; then
  migration_file="contracts/signing-migrations/$YLVEN_PHASE.properties"
  [[ -f "$migration_file" ]] || {
    echo "owner signing certificate mismatch without an approved phase migration: current=$current_certificate previous=$previous_certificate" >&2
    exit 3
  }
  migration_property() {
    sed -n "s/^$1=//p" "$migration_file" | tr -d '' | head -n 1
  }
  [[ "$(migration_property phase)" == "$YLVEN_PHASE" && "$(migration_property approved)" == "true" ]] || {
    echo "invalid signing migration approval for $YLVEN_PHASE" >&2
    exit 3
  }
  [[ "$(migration_property previous_certificate_sha256)" == "$previous_certificate" ]] || {
    echo "signing migration previous certificate differs from the owner APK" >&2
    exit 3
  }
  [[ "$(migration_property new_certificate_sha256)" == "$current_certificate" ]] || {
    echo "signing migration new certificate differs from the configured owner keystore" >&2
    exit 3
  }
  [[ "$(migration_property install_mode)" == "one_time_uninstall_then_install" && "$(migration_property data_preserved)" == "false" ]] || {
    echo "signing migration must explicitly record destructive one-time installation" >&2
    exit 3
  }
  signing_migration=true
  signing_migration_install_mode="one_time_uninstall_then_install"
  signing_migration_data_preserved=false
fi

owner_name="YLVEN-${YLVEN_VERSION}-${YLVEN_PHASE}.apk"
test_name="YLVEN-${YLVEN_VERSION}-${YLVEN_PHASE}-androidTest.apk"
cp "$app_apk" "$YLVEN_ARTIFACT_ROOT/$owner_name"
cp "$test_apk" "$YLVEN_ARTIFACT_ROOT/$test_name"
if [[ -d android/app/build/reports/tests/testDebugUnitTest ]]; then
  tar -czf "$YLVEN_ARTIFACT_ROOT/android-unit-test-report.tar.gz" -C android/app/build/reports/tests testDebugUnitTest
fi
for lint_report in android/app/build/reports/lint-results-debug.html android/app/build/reports/lint-results-debug.xml; do
  [[ -f "$lint_report" ]] && cp "$lint_report" "$YLVEN_ARTIFACT_ROOT/"
done

owner_sha="$(sha256sum "$YLVEN_ARTIFACT_ROOT/$owner_name" | awk '{print $1}')"
test_sha="$(sha256sum "$YLVEN_ARTIFACT_ROOT/$test_name" | awk '{print $1}')"
previous_sha="$(sha256sum "$YLVEN_PREVIOUS_APK" | awk '{print $1}')"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
gradle_version="$(gradle --version | sed -n 's/^Gradle //p' | head -n 1)"
java_version="$(java -version 2>&1 | sed -n '1s/.*version "\([^"]*\)".*/\1/p')"
cat >"$YLVEN_ARTIFACT_ROOT/服务器构建来源证明.json" <<EOF
{
  "schema_version": "2.0",
  "phase": "$YLVEN_PHASE",
  "version": "$YLVEN_VERSION",
  "version_code": $expected_code,
  "commit_sha": "$YLVEN_COMMIT",
  "source_archive_sha256": "$YLVEN_SOURCE_ARCHIVE_SHA256",
  "build_host_class": "connected_online_server",
  "build_coordinator": "shared_cross_project_fifo",
  "build_run_id": "$ANDROID_BUILD_RUN_ID",
  "builder_image": "${YLVEN_BUILDER_IMAGE:-orbexa/android-builder:gradle8.9-api35-v1}",
  "java_version": "$java_version",
  "gradle_version": "$gradle_version",
  "android_platform": 35,
  "android_build_tools": "36.0.0",
  "application_id": "cc.orbexa.ylven",
  "signing_certificate_sha256": "$current_certificate",
  "previous_apk_sha256": "$previous_sha",
  "previous_signing_certificate_sha256": "$previous_certificate",
  "signing_migration": $signing_migration,
  "signing_migration_install_mode": "$signing_migration_install_mode",
  "signing_migration_data_preserved": $signing_migration_data_preserved,
  "apk": "$owner_name",
  "apk_sha256": "$owner_sha",
  "instrumentation_apk": "$test_name",
  "instrumentation_apk_sha256": "$test_sha",
  "android_unit_tests": "PASS",
  "android_lint": "PASS",
  "generated_at": "$generated_at"
}
EOF
(
  cd "$YLVEN_ARTIFACT_ROOT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS.txt -printf '%P\n' | sort |
    while IFS= read -r file; do sha256sum "$file"; done > SHA256SUMS.txt
)
printf 'SERVER_ANDROID_BUILD_PASS phase=%s version=%s commit=%s apk_sha256=%s\n' \
  "$YLVEN_PHASE" "$YLVEN_VERSION" "$YLVEN_COMMIT" "$owner_sha"
