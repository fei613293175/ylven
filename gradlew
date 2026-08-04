#!/usr/bin/env sh
set -eu

if command -v gradle >/dev/null 2>&1; then
  exec gradle "$@"
fi

echo "Gradle is not installed. Use the GitHub Actions Android runner or install the pinned Gradle toolchain." >&2
exit 127
