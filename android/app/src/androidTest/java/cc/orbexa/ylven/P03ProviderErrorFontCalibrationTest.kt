package cc.orbexa.ylven

import android.content.ContentValues
import android.graphics.Bitmap
import android.graphics.Paint
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
import kotlin.math.roundToInt
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
    val subpixelValues = listOf(false, true)
    val linearValues = listOf(false, true)
    val hintingValues = listOf(Paint.HINTING_OFF, Paint.HINTING_ON)
    val strokeWidths = listOf(0f, .04f, .08f)
    val scaleXs = listOf(1.002f, 1.004f)
    subpixelValues.forEach { subpixelText ->
        linearValues.forEach { linearText ->
            hintingValues.forEach { hinting ->
                strokeWidths.forEach { strokeWidth ->
                    scaleXs.forEach { scaleX ->
                        val strokeCode = (strokeWidth * 1000).roundToInt().toString().padStart(3, '0')
                        val scaleCode = (scaleX * 1000).roundToInt().toString()
                        add(
                            ProviderErrorVariant(
                                name = "provider-sp${subpixelText.toInt()}-ln${linearText.toInt()}-hi$hinting-sw$strokeCode-sx$scaleCode",
                                calibration = ProviderErrorFontCalibration(
                                    fontWeight = 550,
                                    fakeBold = true,
                                    strokeWidth = strokeWidth,
                                    scaleX = scaleX,
                                    subpixelText = subpixelText,
                                    linearText = linearText,
                                    hinting = hinting,
                                ),
                            ),
                        )
                    }
                }
            }
        }
    }
}

private fun Boolean.toInt(): Int = if (this) 1 else 0
