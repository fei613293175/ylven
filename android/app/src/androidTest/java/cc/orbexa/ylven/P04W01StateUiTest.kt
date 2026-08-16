package cc.orbexa.ylven

import android.os.SystemClock
import androidx.activity.compose.setContent
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.ui.P03ProductionContractState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

/** Captures the P04-W01 sheet states from the production selector composables. */
class P04W01StateUiTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun everyP04W01StateIsCapturedFromProductionSelectors() {
        check(P04_W01_STATE_IDS.size == 12) { "P04-W01 state catalog changed; update this test." }
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "screenshots-p04",
            scopedDownloadDirectory = "ylven-p04",
        )
        val activeState = mutableStateOf(P04_W01_STATE_IDS.first())
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(activeState.value) { P03ProductionContractState(activeState.value) }
                }
            }
        }

        P04_W01_STATE_IDS.forEach { stateId ->
            composeRule.runOnIdle { activeState.value = stateId }
            composeRule.waitForIdle()
            val rootTag = "production-state-$stateId"
            composeRule.onNodeWithTag(rootTag).assertExists()
            SystemClock.sleep(350)
            val bitmap = requireNotNull(instrumentation.uiAutomation.takeScreenshot()) {
                "Could not capture P04-W01 state $stateId"
            }
            try {
                check(bitmap.width > 0 && bitmap.height > 0) { "P04-W01 screenshot $stateId is empty" }
                P03ScreenshotStorage.writePng(
                    context = context,
                    bitmap = bitmap,
                    legacyDirectory = "screenshots-p04",
                    scopedDownloadDirectory = "ylven-p04",
                    fileName = "$stateId.png",
                    subject = "P04-W01 production selector screenshot",
                )
            } finally {
                bitmap.recycle()
            }
        }
    }
}

private val P04_W01_STATE_IDS = listOf(
    "YL-A-033-S01_POPULATED",
    "YL-A-033-S02_FILTER_ACTIVE",
    "YL-A-033-S03_EMPTY",
    "YL-A-033-S04_DISABLED",
    "YL-A-033-S05_SERVICE_DEGRADED",
    "YL-A-033-S06_SERVER_ERROR",
    "YL-A-034-S01_POPULATED",
    "YL-A-034-S02_FILTER_ACTIVE",
    "YL-A-034-S03_EMPTY",
    "YL-A-034-S04_DISABLED",
    "YL-A-034-S05_SERVICE_DEGRADED",
    "YL-A-034-S06_SERVER_ERROR",
)
