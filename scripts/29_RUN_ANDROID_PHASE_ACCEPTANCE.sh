#!/usr/bin/env bash
set -euo pipefail
PHASE="${1:-AUTO}"; VERSION="${2:-AUTO}"
if [[ "$PHASE" == "AUTO" ]]; then PHASE="P00"; VERSION="1.0.0"; fi
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
mkdir -p build/owner-release/screenshots build/owner-release/test-results
APK="$(find ./android/app/build/outputs/apk/debug -maxdepth 1 -type f -name 'app-debug.apk' -print -quit 2>/dev/null || true)"
if [[ -z "$APK" ]]; then echo 'No Gradle APK found.' >&2; exit 1; fi
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
EXPECTED_CODE=$((MAJOR * 1000000 + MINOR * 10000 + PATCH * 100))
AAPT="$(find "${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}/build-tools" -type f -name aapt -perm -111 2>/dev/null | sort | tail -n 1)"
if [[ -z "$AAPT" ]]; then echo 'Android aapt was not found under the configured SDK.' >&2; exit 1; fi
BADGING="$("$AAPT" dump badging "$APK")"
grep -q "versionCode='$EXPECTED_CODE' versionName='$VERSION'" <<< "$BADGING" || {
  echo "APK manifest version mismatch: expected $VERSION/$EXPECTED_CODE" >&2
  echo "$BADGING" | head -n 2 >&2
  exit 1
}
PACKAGE_ID="$(sed -n "s/^package: name='\([^']*\)'.*/\1/p" <<< "$BADGING")"
if [[ -z "$PACKAGE_ID" ]]; then
  echo 'APK package name could not be read from aapt badging.' >&2
  exit 1
fi
adb wait-for-device
adb install -r "$APK"
PM_PATH="$(adb shell pm path "$PACKAGE_ID")"
grep -q '^package:' <<< "$PM_PATH" || {
  echo "Installed APK package was not found on the emulator: $PACKAGE_ID" >&2
  echo "$PM_PATH" >&2
  exit 1
}
# The actual Compose test suite must enumerate Interaction IDs and emit state screenshots.
./gradlew --no-daemon -PylvenVersionName="$VERSION" -PylvenApiBaseUrl="https://ai-admin.orbexa.cc" -PylvenStagingTurnstileToken="test-pass" connectedDebugAndroidTest
if [[ "$PHASE" == "P01" ]]; then
  # API 35 blocks direct adb access to /sdcard/Android/data. The test writes
  # screenshots to MediaStore Downloads so they survive test-app cleanup.
  for screenshot in \
    P01-AUTH-LOGIN.png \
    P01-AUTH-REGISTER.png \
    P01-AUTH-REGISTER-OTP.png \
    P01-AUTH-REGISTERED.png \
    P01-AUTH-ACCOUNT.png; do
    remote="/sdcard/Download/ylven-p01/$screenshot"
    output="build/owner-release/screenshots/$screenshot"
    adb shell test -s "$remote"
    adb pull "$remote" "$output" >/dev/null
    test -s "$output"
  done
fi
cp "$APK" "build/owner-release/YLVEN-${VERSION}-${PHASE}.apk"
python scripts/30_COMPARE_ANDROID_SCREENSHOTS.py --phase "$PHASE" --screenshots build/owner-release/screenshots --report build/owner-release/VISUAL_DIFF_REPORT.md
python scripts/31_VALIDATE_INTERACTION_TEST_COVERAGE.py
printf '# Automated Test Report\n\n- Phase: %s\n- Version: %s\n- Result: PASS\n' "$PHASE" "$VERSION" > build/owner-release/AUTOMATED_TEST_REPORT.md
