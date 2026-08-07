plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.kapt)
}

android {
    namespace = "cc.orbexa.ylven"
    compileSdk = 35

    defaultConfig {
        applicationId = "cc.orbexa.ylven"
        minSdk = 26
        targetSdk = 35
        val ownerVersionName = providers.gradleProperty("ylvenVersionName").orElse("1.0.0").get()
        val versionParts = ownerVersionName.split(".").map(String::toInt)
        versionCode = versionParts[0] * 1_000_000 + versionParts[1] * 10_000 + versionParts[2] * 100
        versionName = ownerVersionName

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        vectorDrawables { useSupportLibrary = true }
        val apiBaseUrl = providers.gradleProperty("ylvenApiBaseUrl")
            .orElse("https://ai-admin.orbexa.cc")
            .get()
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
        buildConfigField("String", "API_BASE_URL", "\"$apiBaseUrl\"")
    }

    val signingValues = mapOf(
        "storeFile" to providers.gradleProperty("ylvenSigningStoreFile")
            .orElse(providers.environmentVariable("YLVEN_SIGNING_STORE_FILE")).orNull,
        "storePassword" to providers.gradleProperty("ylvenSigningStorePassword")
            .orElse(providers.environmentVariable("YLVEN_SIGNING_STORE_PASSWORD")).orNull,
        "keyAlias" to providers.gradleProperty("ylvenSigningKeyAlias")
            .orElse(providers.environmentVariable("YLVEN_SIGNING_KEY_ALIAS")).orNull,
        "keyPassword" to providers.gradleProperty("ylvenSigningKeyPassword")
            .orElse(providers.environmentVariable("YLVEN_SIGNING_KEY_PASSWORD")).orNull,
    )
    val ownerSigningRequired = providers.gradleProperty("ylvenRequireOwnerSigning")
        .map(String::toBoolean).orElse(false).get()
    val ownerSigningConfigured = signingValues.values.all { !it.isNullOrBlank() }
    if (ownerSigningRequired && !ownerSigningConfigured) {
        throw GradleException(
            "Owner release signing is required. Configure ylvenSigningStoreFile, " +
                "ylvenSigningStorePassword, ylvenSigningKeyAlias and ylvenSigningKeyPassword " +
                "through private server settings.",
        )
    }
    if (ownerSigningConfigured) {
        signingConfigs.create("owner") {
            storeFile = file(requireNotNull(signingValues["storeFile"]))
            storePassword = signingValues["storePassword"]
            keyAlias = signingValues["keyAlias"]
            keyPassword = signingValues["keyPassword"]
        }
    }

    buildTypes {
        debug {
            signingConfigs.findByName("owner")?.let { signingConfig = it }
        }
        release {
            isMinifyEnabled = false
            signingConfigs.findByName("owner")?.let { signingConfig = it }
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions { jvmTarget = "17" }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }

    testOptions {
        animationsDisabled = true
    }
}

dependencies {
    implementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(platform(libs.androidx.compose.bom))

    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    implementation(libs.androidx.room.runtime)
    implementation(libs.androidx.room.ktx)
    kapt(libs.androidx.room.compiler)

    debugImplementation(libs.androidx.compose.ui.tooling)
    debugImplementation(libs.androidx.compose.ui.test.manifest)

    testImplementation(libs.junit)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
}
