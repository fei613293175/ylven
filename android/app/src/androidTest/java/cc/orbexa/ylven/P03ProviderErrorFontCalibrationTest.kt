package cc.orbexa.ylven

import android.content.ContentValues
import android.graphics.Bitmap
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import androidx.activity.compose.setContent
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.ui.ProviderErrorFontCalibration
import cc.orbexa.ylven.ui.YlvenAcceptanceState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

class P03ProviderErrorFontCalibrationTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun captureProviderErrorFontCalibrationGrid() {
        val activeVariant = mutableStateOf(PROVIDER_ERROR_VARIANTS.first())
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(activeVariant.value.name) {
                        YlvenAcceptanceState(PROVIDER_ERROR_STATE, activeVariant.value.calibration)
                    }
                }
            }
        }
        PROVIDER_ERROR_VARIANTS.forEach { variant ->
            composeRule.runOnIdle { activeVariant.value = variant }
            composeRule.waitForIdle()
            composeRule.onNodeWithTag("acceptance-state-$PROVIDER_ERROR_STATE").assertExists()
            capture(variant.name)
        }
    }

    private fun capture(name: String) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val bitmap = composeRule.onNodeWithTag("acceptance-state-$PROVIDER_ERROR_STATE")
            .captureToImage()
            .asAndroidBitmap()
        check(bitmap.width > 0 && bitmap.height > 0)
        check(Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q)
        val resolver = context.contentResolver
        val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/ylven-p03-calibration/"
        resolver.delete(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            "${MediaStore.MediaColumns.DISPLAY_NAME} = ? AND ${MediaStore.MediaColumns.RELATIVE_PATH} = ?",
            arrayOf("$name.png", relativePath),
        )
        val values = ContentValues().apply {
            put(MediaStore.MediaColumns.DISPLAY_NAME, "$name.png")
            put(MediaStore.MediaColumns.MIME_TYPE, "image/png")
            put(MediaStore.MediaColumns.RELATIVE_PATH, relativePath)
            put(MediaStore.MediaColumns.IS_PENDING, 1)
        }
        val uri = requireNotNull(resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values))
        try {
            requireNotNull(resolver.openOutputStream(uri)).use { stream ->
                check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream))
            }
            resolver.update(
                uri,
                ContentValues().apply { put(MediaStore.MediaColumns.IS_PENDING, 0) },
                null,
                null,
            )
        } catch (failure: Throwable) {
            resolver.delete(uri, null, null)
            throw failure
        } finally {
            bitmap.recycle()
        }
    }
}

private const val PROVIDER_ERROR_STATE = "YL-A-023-S12_PROVIDER_ERROR"

private data class ProviderErrorVariant(
    val name: String,
    val calibration: ProviderErrorFontCalibration,
)

private val PROVIDER_ERROR_VARIANTS = buildList {
    val strokeWidths = listOf(0f, .2f, .4f, .6f, .8f, 1f)
    val scaleXs = listOf(1f, 1.002f, 1.004f, 1.006f, 1.008f)
    val baselineOffsets = listOf(-.6f, -.3f, 0f)
    strokeWidths.forEach { strokeWidth ->
        scaleXs.forEach { scaleX ->
            baselineOffsets.forEach { baselineOffset ->
                val strokeCode = (strokeWidth * 100).toInt().toString().padStart(3, '0')
                val scaleCode = (scaleX * 1000).toInt().toString()
                val baselineCode = if (baselineOffset < 0) {
                    "n${(-baselineOffset * 10).toInt().toString().padStart(2, '0')}"
                } else {
                    "p00"
                }
                add(
                    ProviderErrorVariant(
                        name = "provider-f0-sw$strokeCode-sx$scaleCode-by$baselineCode",
                        calibration = ProviderErrorFontCalibration(
                            fakeBold = false,
                            strokeWidth = strokeWidth,
                            scaleX = scaleX,
                            baselineOffset = baselineOffset,
                        ),
                    ),
                )
            }
        }
    }
}
