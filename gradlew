#!/usr/bin/env sh
set -eu

if command -v gradle >/dev/null 2>&1; then
  exec gradle "$@"
fi

echo "Gradle is not installed. Run the pinned Gradle toolchain on the connected online build server." >&2
exit 127
