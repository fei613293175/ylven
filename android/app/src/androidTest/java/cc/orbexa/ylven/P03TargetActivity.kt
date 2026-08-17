package cc.orbexa.ylven

import android.content.Intent
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage

internal fun launchP03TargetActivity(): MainActivity {
    val instrumentation = InstrumentationRegistry.getInstrumentation()
    // The acceptance script already performs the install/launch smoke check.
    // Start the target through Instrumentation after ComposeTestRule has been
    // initialized; shelling out to `am force-stop` from a vivo test process can
    // tear down UiAutomation and terminate instrumentation before the hierarchy
    // is attached.
    val intent = Intent(instrumentation.targetContext, MainActivity::class.java).apply {
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP)
        addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
    }
    instrumentation.startActivitySync(intent)

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
