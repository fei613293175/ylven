package cc.orbexa.ylven

import android.os.ParcelFileDescriptor
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage

internal fun launchP03TargetActivity(): MainActivity {
    val instrumentation = InstrumentationRegistry.getInstrumentation()
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
    repeat(100) {
        var resumed: MainActivity? = null
        instrumentation.runOnMainSync {
            resumed = ActivityLifecycleMonitorRegistry.getInstance()
                .getActivitiesInStage(Stage.RESUMED)
                .filterIsInstance<MainActivity>()
                .singleOrNull()
        }
        resumed?.let { return it }
        Thread.sleep(100)
    }
    error("The target APK MainActivity did not reach RESUMED within 10 seconds.")
}
