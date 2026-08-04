@echo off
where gradle >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  gradle %*
  exit /b %ERRORLEVEL%
)
echo Gradle is not installed. Use the GitHub Actions Android runner or install the pinned Gradle toolchain. 1>&2
exit /b 127
