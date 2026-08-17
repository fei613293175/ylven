package cc.orbexa.ylven

import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.os.ParcelFileDescriptor
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage

internal fun launchP03TargetActivity(): MainActivity {
    val instrumentation = InstrumentationRegistry.getInstrumentation()
    // The acceptance script performs an install/launch smoke check before it
    // starts instrumentation. Recreate the target after ComposeTestRule has
    // been initialized so Compose registers the hierarchy with this test.
    instrumentation.uiAutomation.executeShellCommand(
        "am force-stop cc.orbexa.ylven",
    ).close()
    val descriptor = instrumentation.uiAutomation.executeShellCommand(
        "am start -W -n cc.orbexa.ylven/.MainActivity",
    )
    val output = ParcelFileDescriptor.AutoCloseInputStream(descriptor)
        .bufferedReader()
        .use { it.readText() }
    check(Regex("Status:\\s*ok").containsMatchIn(output)) {
        "Could not launch the target APK MainActivity: $output"
    }

    instrumentation.waitForIdleSync()
    repeat(150) {
        var resumed: MainActivity? = null
        instrumentation.runOnMainSync {
            resumed = ActivityLifecycleMonitorRegistry.getInstance()
                .getActivitiesInStage(Stage.RESUMED)
                .filterIsInstance<MainActivity>()
                .singleOrNull()
            resumed?.let { activity ->
                val metrics = activity.resources.displayMetrics
                if (
                    activity.resources.configuration.orientation != Configuration.ORIENTATION_PORTRAIT ||
                    metrics.widthPixels >= metrics.heightPixels
                ) {
                    // Keep the physical display untouched while capturing the portrait-only
                    // commercial visual baseline from the instrumentation activity.
                    activity.requestedOrientation = ActivityInfo.SCREEN_ORIENTATION_PORTRAIT
                }
            }
        }
        resumed?.let { activity ->
            val metrics = activity.resources.displayMetrics
            if (
                activity.resources.configuration.orientation == Configuration.ORIENTATION_PORTRAIT &&
                metrics.widthPixels < metrics.heightPixels
            ) {
                return activity
            }
        }
        Thread.sleep(100)
    }
    error("The target APK MainActivity did not reach portrait RESUMED state within 15 seconds.")
}
