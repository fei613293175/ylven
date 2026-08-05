package cc.orbexa.ylven.identity

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import cc.orbexa.ylven.BuildConfig
import java.net.HttpURLConnection
import java.net.URL
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

data class OtpChallenge(
    val id: String,
    val email: String,
    val debugCode: String?,
)

data class AuthSession(
    val email: String,
    val sessionId: String,
    val bearer: String,
    val renewal: String,
)

data class DeviceSession(
    val id: String,
    val createdAt: String,
    val expiresAt: String,
    val revoked: Boolean,
)

interface IdentityGateway {
    suspend fun startRegistration(email: String): OtpChallenge
    suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String)
    suspend fun startLogin(email: String): OtpChallenge
    suspend fun finishLogin(challenge: OtpChallenge, code: String): AuthSession
    suspend fun refresh(session: AuthSession): AuthSession
    suspend fun devices(bearer: String): List<DeviceSession>
    suspend fun revokeDevice(bearer: String, sessionId: String)
    suspend fun logout(bearer: String, allDevices: Boolean)
}

class ApiException(
    val status: Int,
    val code: String,
    message: String,
) : Exception(message)

class HttpIdentityGateway(
    private val baseUrl: String = BuildConfig.API_BASE_URL.trimEnd('/'),
    private val turnstileToken: () -> String? = {
        BuildConfig.STAGING_TURNSTILE_TOKEN.trim().ifBlank { null }
    },
) : IdentityGateway {
    override suspend fun startRegistration(email: String): OtpChallenge {
        val normalized = request("POST", "/api/v1/auth/email/normalize", JSONObject().put("email", email)).getString("email")
        val payload = JSONObject().put("email", normalized).put("purpose", "register")
        turnstileToken()?.let { payload.put("turnstile_token", it) }
        val challenge = request("POST", "/api/v1/auth/register/challenge", payload)
        val challengeId = challenge.getString("challenge_id")
        val delivery = request("POST", "/api/v1/auth/register/otp/send", JSONObject().put("challenge_id", challengeId))
        return OtpChallenge(challengeId, normalized, delivery.optString("debug_code").ifBlank { null })
    }

    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) {
        request(
            "POST",
            "/api/v1/auth/register/otp/verify",
            JSONObject().put("challenge_id", challenge.id).put("code", code),
        )
        request(
            "POST",
            "/api/v1/auth/register/complete",
            JSONObject().put("challenge_id", challenge.id).put("email", challenge.email).put("password", password),
        )
    }

    override suspend fun startLogin(email: String): OtpChallenge {
        val normalized = request("POST", "/api/v1/auth/email/normalize", JSONObject().put("email", email)).getString("email")
        val payload = JSONObject().put("email", normalized)
        turnstileToken()?.let { payload.put("turnstile_token", it) }
        val challenge = request("POST", "/api/v1/auth/login/challenge", payload)
        if (!challenge.has("challenge_id")) {
            throw ApiException(401, "login_unavailable", "无法为该账户创建登录验证")
        }
        val challengeId = challenge.getString("challenge_id")
        val delivery = request("POST", "/api/v1/auth/login/otp/send", JSONObject().put("challenge_id", challengeId))
        return OtpChallenge(challengeId, normalized, delivery.optString("debug_code").ifBlank { null })
    }

    override suspend fun finishLogin(challenge: OtpChallenge, code: String): AuthSession {
        request(
            "POST",
            "/api/v1/auth/login/otp/verify",
            JSONObject().put("challenge_id", challenge.id).put("code", code),
        )
        val body = request(
            "POST",
            "/api/v1/auth/sessions",
            JSONObject().put("challenge_id", challenge.id).put("email", challenge.email),
        )
        return body.toSession(challenge.email)
    }

    override suspend fun refresh(session: AuthSession): AuthSession {
        val body = request(
            "POST",
            "/api/v1/auth/token/refresh",
            JSONObject().put("refresh_token", session.renewal),
        )
        return body.toSession(session.email)
    }

    override suspend fun devices(bearer: String): List<DeviceSession> {
        val items = request("GET", "/api/v1/account/devices", bearer = bearer).getJSONArray("devices")
        return buildList {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(
                    DeviceSession(
                        id = item.getString("id"),
                        createdAt = item.optString("created_at"),
                        expiresAt = item.optString("refresh_expires_at"),
                        revoked = item.optBoolean("revoked"),
                    ),
                )
            }
        }
    }

    override suspend fun revokeDevice(bearer: String, sessionId: String) {
        request("DELETE", "/api/v1/account/devices/$sessionId/sessions", bearer = bearer)
    }

    override suspend fun logout(bearer: String, allDevices: Boolean) {
        request("POST", if (allDevices) "/api/v1/auth/logout-all" else "/api/v1/auth/logout", JSONObject(), bearer)
    }

    private suspend fun request(
        method: String,
        path: String,
        body: JSONObject? = null,
        bearer: String? = null,
    ): JSONObject = withContext(Dispatchers.IO) {
        val connection = (URL(baseUrl + path).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            if (!bearer.isNullOrBlank()) setRequestProperty("Authorization", "Bearer $bearer")
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json; charset=utf-8")
                outputStream.bufferedWriter(Charsets.UTF_8).use { it.write(body.toString()) }
            }
        }
        try {
            val status = connection.responseCode
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val raw = stream?.bufferedReader(Charsets.UTF_8)?.use { it.readText() }.orEmpty()
            val json = if (raw.isBlank()) JSONObject() else JSONObject(raw)
            if (status !in 200..299) {
                val error = json.optJSONObject("error")
                throw ApiException(
                    status,
                    error?.optString("code").orEmpty().ifBlank { "http_$status" },
                    error?.optString("message").orEmpty().ifBlank { "服务请求失败 ($status)" },
                )
            }
            json
        } finally {
            connection.disconnect()
        }
    }

    private fun JSONObject.toSession(email: String) = AuthSession(
        email = email,
        sessionId = getString("session_id"),
        bearer = getString("access_token"),
        renewal = getString("refresh_token"),
    )
}

class SessionStore(context: Context) {
    private val preferences = context.getSharedPreferences("ylven_identity", Context.MODE_PRIVATE)
    private val keyAlias = "ylven_identity_session_v1"

    fun load(): AuthSession? {
        val encoded = preferences.getString("session_blob", null) ?: return null
        return runCatching {
            val packed = Base64.decode(encoded, Base64.NO_WRAP)
            require(packed.size > 12)
            val cipher = Cipher.getInstance("AES/GCM/NoPadding")
            cipher.init(Cipher.DECRYPT_MODE, sessionKey(), GCMParameterSpec(128, packed.copyOfRange(0, 12)))
            val json = JSONObject(String(cipher.doFinal(packed.copyOfRange(12, packed.size)), Charsets.UTF_8))
            AuthSession(
                email = json.getString("email"),
                sessionId = json.getString("session_id"),
                bearer = json.getString("access_token"),
                renewal = json.getString("refresh_token"),
            )
        }.getOrElse {
            preferences.edit().clear().commit()
            null
        }
    }

    fun save(session: AuthSession?) {
        if (session == null) {
            preferences.edit().clear().commit()
            return
        }
        val plain = JSONObject()
            .put("email", session.email)
            .put("session_id", session.sessionId)
            .put("access_token", session.bearer)
            .put("refresh_token", session.renewal)
            .toString()
            .toByteArray(Charsets.UTF_8)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, sessionKey())
        val packed = cipher.iv + cipher.doFinal(plain)
        preferences.edit()
            .clear()
            .putString("session_blob", Base64.encodeToString(packed, Base64.NO_WRAP))
            .commit()
    }

    private fun sessionKey(): SecretKey {
        val keyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (keyStore.getKey(keyAlias, null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore").run {
            init(
                KeyGenParameterSpec.Builder(
                    keyAlias,
                    KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
                )
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .build(),
            )
            generateKey()
        }
    }
}
