#!/usr/bin/env bash
set -euo pipefail
PHASE="${1:-AUTO}"; VERSION="${2:-AUTO}"; UPGRADE_FROM="${3:-}"
if [[ "$PHASE" == "AUTO" ]]; then PHASE="P00"; VERSION="1.0.0"; UPGRADE_FROM="NONE"; fi
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
if [[ "$PHASE" != "P00" && -z "$UPGRADE_FROM" ]]; then
  UPGRADE_FROM="$(python scripts/32_VERIFY_ANDROID_VERSION_CONTRACT.py --phase "$PHASE" --print-upgrade-from | tail -n 1)"
fi
if [[ "$PHASE" != "P00" && ( -z "$UPGRADE_FROM" || "$UPGRADE_FROM" == "NONE" ) ]]; then
  echo "No upgrade source version is available for $PHASE." >&2
  exit 1
fi
mkdir -p build/owner-release/截图 build/owner-release/test-results
APK="$(find ./android/app/build/outputs/apk/debug -maxdepth 1 -type f -name 'app-debug.apk' -print -quit 2>/dev/null || true)"
if [[ -z "$APK" ]]; then echo 'No Gradle APK found.' >&2; exit 1; fi
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
EXPECTED_CODE=$((MAJOR * 1000000 + MINOR * 10000 + PATCH * 100))
AAPT="$(find "${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}/build-tools" -type f -name aapt -perm -111 2>/dev/null | sort | tail -n 1)"
if [[ -z "$AAPT" ]]; then echo 'Android aapt was not found under the configured SDK.' >&2; exit 1; fi
BADGING="$($AAPT dump badging "$APK")"
grep -q "versionCode='$EXPECTED_CODE' versionName='$VERSION'" <<< "$BADGING" || {
  echo "APK manifest version mismatch: expected $VERSION/$EXPECTED_CODE" >&2
  echo "$BADGING" | head -n 2 >&2
  exit 1
}
PACKAGE_ID="$(sed -n "s/^package: name='\([^']*\)'.*/\1/p" <<< "$BADGING")"
[[ "$PACKAGE_ID" == "cc.orbexa.ylven" ]] || { echo "Unexpected applicationId: $PACKAGE_ID" >&2; exit 1; }

APKSIGNER="$(find "${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}/build-tools" -type f -name apksigner -perm -111 2>/dev/null | sort | tail -n 1)"
[[ -n "$APKSIGNER" ]] || { echo 'Android apksigner was not found under the configured SDK.' >&2; exit 1; }
certificate_digest() {
  "$APKSIGNER" verify --print-certs "$1" 2>/dev/null | sed -n 's/.*certificate SHA-256 digest: //p' | head -n 1
}
CURRENT_CERT="$(certificate_digest "$APK")"
[[ -n "$CURRENT_CERT" ]] || { echo 'Current APK signing certificate could not be read.' >&2; exit 1; }

PREVIOUS_APK=""
if [[ "$PHASE" != "P00" && -n "$UPGRADE_FROM" && "$UPGRADE_FROM" != "NONE" ]]; then
  cp "$APK" build/current-upgrade.apk
  ./gradlew --no-daemon -PylvenVersionName="$UPGRADE_FROM" -PylvenApiBaseUrl="https://ai-admin.orbexa.cc" -PylvenStagingTurnstileToken="test-pass" assembleDebug >/dev/null
  PREVIOUS_APK="build/previous-upgrade.apk"
  cp "./android/app/build/outputs/apk/debug/app-debug.apk" "$PREVIOUS_APK"
  cp build/current-upgrade.apk "$APK"
  PREVIOUS_BADGING="$($AAPT dump badging "$PREVIOUS_APK")"
  grep -q "versionName='$UPGRADE_FROM'" <<< "$PREVIOUS_BADGING" || { echo "Previous APK version mismatch: $UPGRADE_FROM" >&2; exit 1; }
  PREVIOUS_PACKAGE="$(sed -n "s/^package: name='\([^']*\)'.*/\1/p" <<< "$PREVIOUS_BADGING")"
  [[ "$PREVIOUS_PACKAGE" == "$PACKAGE_ID" ]] || { echo 'Previous APK applicationId differs.' >&2; exit 1; }
  PREVIOUS_CERT="$(certificate_digest "$PREVIOUS_APK")"
  [[ "$PREVIOUS_CERT" == "$CURRENT_CERT" ]] || { echo 'Signing certificate changed; overlay installation is forbidden.' >&2; exit 1; }
fi

adb wait-for-device
if [[ -n "$PREVIOUS_APK" ]]; then
  adb install -r "$PREVIOUS_APK"
else
  adb install -r "$APK"
fi
adb shell pm path "$PACKAGE_ID" | grep -q '^package:' || {
  echo "APK package was not installed before run-as: $PACKAGE_ID" >&2
  exit 1
}
adb shell "run-as $PACKAGE_ID sh -c 'mkdir -p files; printf upgrade-ok > files/upgrade-marker; printf logged-in > files/login-state-marker'"
adb install -r "$APK"
adb shell "run-as $PACKAGE_ID test -s files/upgrade-marker"
adb shell "run-as $PACKAGE_ID test -s files/login-state-marker"
cat > build/owner-release/覆盖安装证据.md <<EOF
# 覆盖安装证据

- 阶段：$PHASE
- 当前版本：$VERSION
- applicationId：$PACKAGE_ID
- 安装命令：adb install -r
- 清除数据：未执行
- 上一版本：${UPGRADE_FROM:-P00 首版重装校验}
- 签名证书：当前与上一版本（如适用）一致
- 数据标记：upgrade-marker 和 login-state-marker 覆盖安装后仍存在
- 结果：PASS
EOF
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
  while IFS='|' read -r screenshot output_name; do
    remote="/sdcard/Download/ylven-p01/$screenshot"
    output="build/owner-release/截图/$output_name"
    adb shell test -s "$remote"
    adb pull "$remote" "$output" >/dev/null
    test -s "$output"
  done <<'SCREENSHOTS'
P01-AUTH-LOGIN.png|登录页.png
P01-AUTH-REGISTER.png|注册页.png
P01-AUTH-REGISTER-OTP.png|注册验证码页.png
P01-AUTH-REGISTERED.png|注册完成页.png
P01-AUTH-ACCOUNT.png|账户与设备页.png
SCREENSHOTS
fi
cp "$APK" "build/owner-release/YLVEN-${VERSION}-${PHASE}.apk"
python scripts/30_COMPARE_ANDROID_SCREENSHOTS.py --phase "$PHASE" --screenshots build/owner-release/截图 --report build/owner-release/视觉差异报告.md
python scripts/31_VALIDATE_INTERACTION_TEST_COVERAGE.py
printf '# 自动化测试报告\n\n- Phase: %s\n- Version: %s\n- Result: PASS\n' "$PHASE" "$VERSION" > build/owner-release/自动化测试报告.md
