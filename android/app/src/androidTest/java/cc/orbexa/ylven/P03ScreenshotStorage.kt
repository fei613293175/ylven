package cc.orbexa.ylven

import android.content.ContentValues
import android.content.Context
import android.graphics.Bitmap
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import java.io.File
import java.io.FileOutputStream

/** Persists physical-device screenshots without loading Android 10 APIs on Android 9. */
internal object P03ScreenshotStorage {
    fun resetDirectory(
        context: Context,
        legacyDirectory: String,
        scopedDownloadDirectory: String,
    ) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ScopedDownloadMediaWriter.resetDirectory(context, scopedDownloadDirectory)
        } else {
            val output = File(requireNotNull(context.getExternalFilesDir(null)), legacyDirectory)
            check(!output.exists() || output.deleteRecursively()) {
                "Could not clear legacy screenshot directory $legacyDirectory"
            }
        }
    }

    fun writePng(
        context: Context,
        bitmap: Bitmap,
        legacyDirectory: String,
        scopedDownloadDirectory: String,
        fileName: String,
        subject: String,
    ) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ScopedDownloadMediaWriter.writePng(
                context = context,
                bitmap = bitmap,
                directory = scopedDownloadDirectory,
                fileName = fileName,
                subject = subject,
            )
        } else {
            val output = File(requireNotNull(context.getExternalFilesDir(null)), "$legacyDirectory/$fileName")
            output.parentFile?.mkdirs()
            FileOutputStream(output).use { stream ->
                check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream)) {
                    "Could not write $subject"
                }
            }
        }
    }
}

/** Loaded only on API 29+, where MediaStore.Downloads and scoped-storage columns exist. */
private object ScopedDownloadMediaWriter {
    @Suppress("NewApi")
    fun resetDirectory(context: Context, directory: String) {
        val resolver = context.contentResolver
        val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/$directory/"
        resolver.delete(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            "${MediaStore.MediaColumns.RELATIVE_PATH} = ?",
            arrayOf(relativePath),
        )
        resolver.query(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            arrayOf(MediaStore.MediaColumns._ID),
            "${MediaStore.MediaColumns.RELATIVE_PATH} = ?",
            arrayOf(relativePath),
            null,
        ).use { remaining ->
            check(remaining == null || !remaining.moveToFirst()) {
                "MediaStore still contains entries for $relativePath"
            }
        }
    }

    @Suppress("NewApi")
    fun writePng(
        context: Context,
        bitmap: Bitmap,
        directory: String,
        fileName: String,
        subject: String,
    ) {
        val resolver = context.contentResolver
        val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/$directory/"
        resolver.delete(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            "${MediaStore.MediaColumns.DISPLAY_NAME} = ? AND ${MediaStore.MediaColumns.RELATIVE_PATH} = ?",
            arrayOf(fileName, relativePath),
        )
        val values = ContentValues().apply {
            put(MediaStore.MediaColumns.DISPLAY_NAME, fileName)
            put(MediaStore.MediaColumns.MIME_TYPE, "image/png")
            put(MediaStore.MediaColumns.RELATIVE_PATH, relativePath)
            put(MediaStore.MediaColumns.IS_PENDING, 1)
        }
        val uri = requireNotNull(resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values)) {
            "Could not create $subject media entry"
        }
        try {
            requireNotNull(resolver.openOutputStream(uri)).use { stream ->
                check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream)) {
                    "Could not write $subject"
                }
            }
            resolver.update(uri, ContentValues().apply {
                put(MediaStore.MediaColumns.IS_PENDING, 0)
            }, null, null)
        } catch (failure: Throwable) {
            resolver.delete(uri, null, null)
            throw failure
        }
    }
}
