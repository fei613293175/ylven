package cc.orbexa.ylven.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable

private val LightColors = lightColorScheme(
    primary = YlvenLightColors.Primary,
    onPrimary = YlvenLightColors.Surface,
    secondary = YlvenLightColors.Secondary,
    tertiary = YlvenLightColors.Tertiary,
    background = YlvenLightColors.Background,
    onBackground = YlvenLightColors.TextPrimary,
    surface = YlvenLightColors.Surface,
    surfaceVariant = YlvenLightColors.SurfaceSubtle,
    onSurface = YlvenLightColors.TextPrimary,
    onSurfaceVariant = YlvenLightColors.TextSecondary,
    outline = YlvenLightColors.Border,
    outlineVariant = YlvenLightColors.BorderStrong,
    error = YlvenLightColors.Error,
)

private val DarkColors = darkColorScheme(
    primary = YlvenDarkColors.Primary,
    onPrimary = YlvenDarkColors.Background,
    secondary = YlvenDarkColors.Secondary,
    tertiary = YlvenDarkColors.Tertiary,
    background = YlvenDarkColors.Background,
    onBackground = YlvenDarkColors.TextPrimary,
    surface = YlvenDarkColors.Surface,
    surfaceVariant = YlvenDarkColors.SurfaceSubtle,
    onSurface = YlvenDarkColors.TextPrimary,
    onSurfaceVariant = YlvenDarkColors.TextSecondary,
    outline = YlvenDarkColors.Border,
    outlineVariant = YlvenDarkColors.BorderStrong,
    error = YlvenDarkColors.Error,
)

@Composable
fun YlvenTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = YlvenTypography,
        content = content,
    )
}
