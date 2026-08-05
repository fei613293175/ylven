FROM 127.0.0.1:5000/hhy-android-toolchain:r08-api36-cache@sha256:0b4a09d836f3508e68440f231591677704301e42dccec2996fbb88dca7e91738

ARG GRADLE_VERSION=8.9
ARG GRADLE_SHA256=d725d707bfabd4dfdc958c624003b3c80accc03f7037b5122c4b1d0ef15cecab

ENV ANDROID_HOME=/opt/android-sdk \
    ANDROID_SDK_ROOT=/opt/android-sdk \
    GRADLE_HOME=/opt/gradle \
    PATH=/opt/gradle/bin:/opt/android-sdk/cmdline-tools/latest/bin:/opt/android-sdk/platform-tools:/opt/android-sdk/build-tools/36.0.0:/opt/java/openjdk/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    GRADLE_USER_HOME=/root/.gradle

RUN set -eux; \
    curl --fail --location --retry 3 --connect-timeout 20 \
      --output /tmp/gradle.zip \
      "https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip"; \
    echo "${GRADLE_SHA256}  /tmp/gradle.zip" | sha256sum -c -; \
    unzip -q /tmp/gradle.zip -d /opt; \
    mv "/opt/gradle-${GRADLE_VERSION}" /opt/gradle; \
    rm -f /tmp/gradle.zip; \
    yes | sdkmanager --licenses >/dev/null || true; \
    sdkmanager --install \
      "platforms;android-35" \
      "platforms;android-36" \
      "platforms;android-37.0" \
      "build-tools;34.0.0" \
      "build-tools;36.0.0" \
      "platform-tools"; \
    gradle --version

WORKDIR /workspace
ENTRYPOINT []
CMD ["gradle", "--version"]
