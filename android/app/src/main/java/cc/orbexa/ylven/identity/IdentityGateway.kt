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
    val expiresAt: String? = null,
    val resendAfterSeconds: Int = 60,
)

data class SecurityChallenge(
    val id: String,
    val email: String,
    val purpose: String,
    val question: String = "",
    /** Only populated by lightweight test gateways. */
    val legacyOtp: OtpChallenge? = null,
)

data class AuthSession(
    val email: String,
    val sessionId: String,
    val bearer: String,
    val renewal: String,
    val deviceId: String = "",
)

data class DeviceSession(
    val id: String,
    val createdAt: String,
    val expiresAt: String,
    val revoked: Boolean,
)

data class Conversation(
    val id: String,
    val title: String,
    val status: String,
    val updatedAt: String,
    val archivedAt: String? = null,
    val titleSource: String = "",
    val titleLocked: Boolean = false,
    val draftSessionId: String? = null,
    val initialDraft: String = "",
    val temporary: Boolean = false,
    val initialToolTrayOpen: Boolean = false,
    val initialComposerTools: List<ComposerToolOption> = emptyList(),
    val initialConsumerCopy: Map<String, String> = emptyMap(),
    val activeBranchId: String = "",
    val defaultModelId: String = "",
    val defaultReasoningProfile: String = "auto",
    val aiSettingsVersion: Long = 0,
    val aiSettingsOverridden: Boolean = false,
)

data class HomeSnapshot(
    val conversations: List<Conversation>,
    val modelCatalog: List<ModelOption>,
    val composerTools: List<ComposerToolOption> = defaultComposerToolOptions(),
    val consumerCopy: Map<String, String> = defaultConsumerCopy(),
)

data class ComposerToolOption(
    val id: String,
    val label: String,
    val enabled: Boolean,
    val prompt: String,
)

fun defaultComposerToolOptions() = listOf(
    ComposerToolOption("camera", "拍照", false, "请分析我接下来拍摄的内容："),
    ComposerToolOption("image", "选择图片", false, "请分析我接下来选择的图片："),
    ComposerToolOption("file", "上传文件", false, "请分析我接下来上传的文件："),
    ComposerToolOption("image-generation", "生成图片", false, "请帮我生成图片："),
    ComposerToolOption("presentation", "制作演示", false, "请帮我制作演示文稿："),
    ComposerToolOption("deep-research", "深度研究", false, "请帮我深入研究："),
)

fun defaultConsumerCopy() = mapOf(
    "thinking" to "正在思考",
    "tool_file_parse" to "正在阅读文件",
    "tool_image" to "正在生成图片",
    "tool_presentation" to "正在制作演示文稿",
    "reconnecting" to "连接不稳定，正在恢复…",
    "rate_limited" to "当前请求较多，请稍后再试",
    "provider_error" to "暂时无法完成回答，请重试",
    "offline" to "当前网络不可用",
    "content_blocked" to "这个请求暂时无法处理，请调整后重试",
)

data class ModelOption(
    val id: String,
    val name: String,
    val enabled: Boolean = true,
    val description: String = "",
    val reasoningProfiles: List<String> = listOf("auto"),
    val providerId: String = "",
    val providerName: String = "",
    val purpose: String = "",
    val speedTier: String = "balanced",
    val capabilities: List<ModelCapabilityOption> = emptyList(),
)

data class ModelCapabilityOption(
    val id: String,
    val label: String,
    val probedAt: String = "",
)

data class MessageRecord(val id: String, val conversationId: String, val role: String, val body: String, val createdAt: String)
data class MessageRun(
    val id: String,
    val conversationId: String,
    val status: String,
    val cursor: Long,
    val assistantMessageId: String? = null,
    val modelId: String = "",
    val reasoningProfile: String = "auto",
)

val MessageRun.isGenerating: Boolean
    get() = status.lowercase() in setOf("queued", "running", "streaming")

val MessageRun.consumerStatusText: String
    get() = when (status.lowercase()) {
        "queued", "running", "streaming" -> "正在思考"
        "completed" -> "回答已完成"
        "cancelled" -> "已停止生成"
        "failed" -> "暂时未能完成"
        "content_blocked" -> "这项请求暂时无法完成"
        else -> "正在处理"
    }

data class FirstMessageResult(val conversation: Conversation, val run: MessageRun)
data class RunEvent(val id: Long, val type: String, val delta: String)
data class MessageCitation(val url: String, val title: String)

data class AIPreference(
    val modelId: String = "",
    val reasoningProfile: String = "auto",
    val version: Long = 0,
    val updatedAt: String = "",
)

data class ConversationBranch(
    val id: String,
    val conversationId: String,
    val parentBranchId: String = "",
    val forkedFromMessageId: String = "",
    val status: String = "active",
    val createdAt: String = "",
    val updatedAt: String = "",
)

data class ComparisonCandidate(
    val runId: String,
    val modelId: String,
    val reasoningProfile: String,
    val status: String,
    val messageId: String = "",
    val body: String = "",
    val errorCode: String = "",
)

data class ComparisonGroup(
    val id: String,
    val conversationId: String,
    val prompt: String,
    val status: String,
    val candidates: List<ComparisonCandidate> = emptyList(),
    val adoptedRunId: String = "",
    val synthesisRunId: String = "",
    val createdAt: String = "",
    val updatedAt: String = "",
)

data class ModelHealthStatus(
    val modelId: String,
    val providerId: String = "",
    val status: String,
    val latencyMs: Long = 0,
    val lastProbeAt: String = "",
    val capabilities: List<String> = emptyList(),
    val errorCode: String = "",
)

data class ModelAvailability(
    val modelId: String,
    val status: String,
    val fallbackModels: List<ModelOption> = emptyList(),
)

interface IdentityGateway {
    suspend fun startRegistration(email: String): OtpChallenge
    suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String)
    suspend fun finishRegistrationSession(challenge: OtpChallenge, code: String, password: String): AuthSession? {
        finishRegistration(challenge, code, password)
        return null
    }
    suspend fun startLogin(email: String): OtpChallenge
    suspend fun finishLogin(challenge: OtpChallenge, code: String): AuthSession
    suspend fun refresh(session: AuthSession): AuthSession
    suspend fun devices(bearer: String): List<DeviceSession>
    suspend fun revokeDevice(bearer: String, sessionId: String)
    suspend fun logout(bearer: String, allDevices: Boolean)
    suspend fun home(bearer: String): HomeSnapshot = HomeSnapshot(emptyList(), emptyList())
    suspend fun models(bearer: String): List<ModelOption> = home(bearer).modelCatalog
    suspend fun reasoningProfiles(bearer: String, modelId: String): List<String> =
        models(bearer).firstOrNull { it.id == modelId }?.reasoningProfiles.orEmpty()
    suspend fun listConversations(bearer: String, cursor: String? = null, includeArchived: Boolean = false): Pair<List<Conversation>, String?> = Pair(emptyList(), null)
    suspend fun searchConversations(bearer: String, query: String): List<Conversation> = emptyList()
    suspend fun createConversation(bearer: String, title: String = ""): Conversation = error("会话功能尚未配置")
    suspend fun createTemporaryConversation(bearer: String, title: String = ""): Conversation = error("临时会话功能尚未配置")
    suspend fun renameConversation(bearer: String, id: String, title: String): Conversation = error("会话功能尚未配置")
    suspend fun archiveConversation(bearer: String, id: String): Conversation = error("会话功能尚未配置")
    suspend fun deleteConversation(bearer: String, id: String): Conversation = error("会话功能尚未配置")
    suspend fun conversationMessages(bearer: String, conversationId: String): List<MessageRecord> = emptyList()
    suspend fun sendMessage(bearer: String, conversationId: String, body: String, model: String = ""): MessageRun = error("消息功能尚未配置")
    suspend fun sendMessageIdempotent(bearer: String, conversationId: String, body: String, model: String = "", idempotencyKey: String): MessageRun =
        sendMessage(bearer, conversationId, body, model)
    suspend fun sendMessageIdempotentWithProfile(
        bearer: String,
        conversationId: String,
        body: String,
        model: String = "",
        reasoningProfile: String = "auto",
        idempotencyKey: String,
    ): MessageRun = sendMessageIdempotent(bearer, conversationId, body, model, idempotencyKey)
    suspend fun startConversationFromFirstMessage(
        bearer: String,
        draftSessionId: String,
        body: String,
        model: String = "",
        idempotencyKey: String,
        temporary: Boolean = false,
    ): FirstMessageResult = error("会话功能尚未配置")
    suspend fun startConversationFromFirstMessageWithProfile(
        bearer: String,
        draftSessionId: String,
        body: String,
        model: String = "",
        reasoningProfile: String = "auto",
        idempotencyKey: String,
        temporary: Boolean = false,
    ): FirstMessageResult = startConversationFromFirstMessage(bearer, draftSessionId, body, model, idempotencyKey, temporary)
    suspend fun runStatus(bearer: String, runId: String): Pair<MessageRun, List<MessageRecord>> = error("运行状态尚未配置")
    suspend fun runEvents(bearer: String, runId: String, after: Long = 0): Pair<MessageRun, List<RunEvent>> = error("流式事件尚未配置")
    suspend fun cancelRun(bearer: String, runId: String): MessageRun = error("取消功能尚未配置")
    suspend fun exportConversation(bearer: String, conversationId: String): String = error("导出功能尚未配置")
    suspend fun exportMessage(bearer: String, messageId: String): String = error("单条回答导出功能尚未配置")
    suspend fun saveDraft(bearer: String, conversationId: String, body: String): String = error("草稿功能尚未配置")
    suspend fun loadDraft(bearer: String, conversationId: String): String = error("草稿功能尚未配置")
    suspend fun submitFeedback(bearer: String, messageId: String, value: String): Boolean = false
    suspend fun regenerate(bearer: String, messageId: String): MessageRun = error("重答功能尚未配置")
    suspend fun speak(bearer: String, messageId: String): Boolean = false
    suspend fun messageCitations(bearer: String, messageId: String): List<MessageCitation> = emptyList()
    suspend fun messageRunMetadata(bearer: String, messageId: String): MessageRun? = null
    suspend fun aiPreference(bearer: String): AIPreference = AIPreference()
    suspend fun updateAIPreference(bearer: String, modelId: String, reasoningProfile: String, version: Long): AIPreference = AIPreference(modelId, reasoningProfile, version)
    suspend fun updateConversationAISettings(bearer: String, conversationId: String, modelId: String, reasoningProfile: String, version: Long): Conversation = error("会话设置尚未配置")
    suspend fun conversationBranches(bearer: String, conversationId: String): List<ConversationBranch> = emptyList()
    suspend fun createConversationBranch(bearer: String, conversationId: String, forkedFromMessageId: String = ""): ConversationBranch = error("分支功能尚未配置")
    suspend fun activateConversationBranch(bearer: String, conversationId: String, branchId: String): Conversation = error("分支功能尚未配置")
    suspend fun rerunMessage(bearer: String, messageId: String, modelId: String = "", reasoningProfile: String = "auto"): MessageRun = error("重答功能尚未配置")
    suspend fun createComparison(bearer: String, conversationId: String, prompt: String, modelIds: List<String>, reasoningProfile: String = "auto"): ComparisonGroup = error("多模型比较尚未配置")
    suspend fun comparison(bearer: String, comparisonId: String): ComparisonGroup = error("多模型比较尚未配置")
    suspend fun adoptComparison(bearer: String, comparisonId: String, runId: String): ComparisonGroup = error("多模型比较尚未配置")
    suspend fun synthesizeComparison(bearer: String, comparisonId: String, runIds: List<String> = emptyList()): ComparisonGroup = error("多模型比较尚未配置")
    suspend fun serviceStatus(bearer: String): List<ModelHealthStatus> = emptyList()
    suspend fun modelAvailability(bearer: String, modelId: String): ModelAvailability =
        ModelAvailability(modelId = modelId, status = "available")

    suspend fun restore(session: AuthSession): AuthSession = session

    suspend fun createLoginChallenge(email: String): SecurityChallenge {
        val otp = startLogin(email)
        return SecurityChallenge(otp.id, otp.email, "login", legacyOtp = otp)
    }

    suspend fun createRegistrationChallenge(email: String): SecurityChallenge {
        val otp = startRegistration(email)
        return SecurityChallenge(otp.id, otp.email, "register", legacyOtp = otp)
    }

    suspend fun verifyAnswer(challenge: SecurityChallenge, answer: String) {
        require(challenge.legacyOtp != null) { "当前网关未实现安全验证" }
    }

    suspend fun requestLoginOtp(challenge: SecurityChallenge): OtpChallenge =
        challenge.legacyOtp ?: error("登录安全验证尚未完成")

    suspend fun requestRegistrationOtp(challenge: SecurityChallenge): OtpChallenge =
        challenge.legacyOtp ?: error("注册安全验证尚未完成")

    suspend fun passwordPolicy(): PasswordPolicy = PasswordPolicy()

    suspend fun initializeWorkspace(session: AuthSession) = Unit
}

data class PasswordPolicy(
    val minLength: Int = 8,
    val requiresLetter: Boolean = true,
    val requiresDigit: Boolean = true,
)

class ApiException(
    val status: Int,
    val code: String,
    message: String,
) : Exception(message)

class HttpIdentityGateway(
    private val baseUrl: String = BuildConfig.API_BASE_URL.trimEnd('/'),
    private val deviceId: String = "android-test-device",
) : IdentityGateway {
    override suspend fun startRegistration(email: String): OtpChallenge {
        val security = createRegistrationChallenge(email)
        verifyAnswer(security, answerFor(security.question))
        return requestRegistrationOtp(security)
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
            JSONObject().put("challenge_id", challenge.id).put("email", challenge.email).put("password", password).put("device_id", deviceId),
        )
    }

    override suspend fun finishRegistrationSession(challenge: OtpChallenge, code: String, password: String): AuthSession {
        request("POST", "/api/v1/auth/register/otp/verify", JSONObject().put("challenge_id", challenge.id).put("code", code))
        val body = request("POST", "/api/v1/auth/register/complete", JSONObject().put("challenge_id", challenge.id).put("email", challenge.email).put("password", password).put("device_id", deviceId))
        return body.toSession(challenge.email)
    }

    override suspend fun startLogin(email: String): OtpChallenge {
        val challenge = createLoginChallenge(email)
        verifyAnswer(challenge, answerFor(challenge.question))
        return requestLoginOtp(challenge)
    }

    private fun answerFor(question: String): String {
        val numbers = Regex("\\d+").findAll(question).map { it.value.toInt() }.toList()
        require(numbers.size == 2) { "验证题目格式不正确" }
        return (numbers[0] + numbers[1]).toString()
    }

    override suspend fun restore(session: AuthSession): AuthSession {
        request("GET", "/api/v1/account/session", bearer = session.bearer)
        return session
    }

    override suspend fun createLoginChallenge(email: String): SecurityChallenge {
        val normalized = request("POST", "/api/v1/auth/email/normalize", JSONObject().put("email", email)).getString("email")
        val challenge = request("POST", "/api/v1/auth/login/challenge", JSONObject().put("email", normalized))
        if (!challenge.has("challenge_id")) {
            throw ApiException(401, "login_unavailable", "无法为该账户创建登录验证")
        }
        return SecurityChallenge(challenge.getString("challenge_id"), normalized, "login", challenge.optString("verification_question"))
    }

    override suspend fun createRegistrationChallenge(email: String): SecurityChallenge {
        val normalized = request("POST", "/api/v1/auth/email/normalize", JSONObject().put("email", email)).getString("email")
        val challenge = request("POST", "/api/v1/auth/register/challenge", JSONObject().put("email", normalized).put("purpose", "register"))
        return SecurityChallenge(challenge.getString("challenge_id"), normalized, "register", challenge.optString("verification_question"))
    }

    override suspend fun verifyAnswer(challenge: SecurityChallenge, answer: String) {
        request("POST", "/api/v1/auth/challenge/verify", JSONObject().put("challenge_id", challenge.id).put("answer", answer))
    }

    override suspend fun requestLoginOtp(challenge: SecurityChallenge): OtpChallenge {
        val delivery = request("POST", "/api/v1/auth/login/otp/send", JSONObject().put("challenge_id", challenge.id))
        return OtpChallenge(challenge.id, challenge.email, delivery.optString("debug_code").ifBlank { null }, delivery.optString("expires_at"), delivery.optInt("resend_after_seconds", 60))
    }

    override suspend fun requestRegistrationOtp(challenge: SecurityChallenge): OtpChallenge {
        val delivery = request("POST", "/api/v1/auth/register/otp/send", JSONObject().put("challenge_id", challenge.id))
        return OtpChallenge(challenge.id, challenge.email, delivery.optString("debug_code").ifBlank { null }, delivery.optString("expires_at"), delivery.optInt("resend_after_seconds", 60))
    }

    override suspend fun passwordPolicy(): PasswordPolicy {
        val body = request("GET", "/api/v1/auth/password-policy")
        return PasswordPolicy(body.optInt("min_length", 8), body.optBoolean("requires_letter", true), body.optBoolean("requires_digit", true))
    }

    override suspend fun initializeWorkspace(session: AuthSession) {
        request("POST", "/api/v1/onboarding/personal-workspace", bearer = session.bearer)
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
            JSONObject().put("challenge_id", challenge.id).put("email", challenge.email).put("device_id", deviceId),
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
                        id = item.optString("id").ifBlank { item.optString("device_id") },
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

    override suspend fun home(bearer: String): HomeSnapshot {
        val body = request("GET", "/api/mobile/v1/home", bearer = bearer)
        val items = body.optJSONArray("conversations") ?: org.json.JSONArray()
        val config = body.optJSONObject("config")
        return HomeSnapshot(
            parseConversations(items),
            parseModelCatalog(body.optJSONArray("model_catalog")),
            parseComposerTools(config?.optJSONArray("composer_tools")),
            parseConsumerCopy(config?.optJSONObject("consumer_copy")),
        )
    }

    override suspend fun models(bearer: String): List<ModelOption> {
        val body = request("GET", "/api/mobile/v1/models?group=provider", bearer = bearer)
        return parseModelCatalog(body.optJSONArray("items"))
    }

    override suspend fun reasoningProfiles(bearer: String, modelId: String): List<String> {
        val encodedModelId = java.net.URLEncoder.encode(modelId, "UTF-8")
        val items = request("GET", "/api/mobile/v1/models/$encodedModelId/reasoning-profiles", bearer = bearer)
            .optJSONArray("items") ?: org.json.JSONArray()
        return buildList {
            for (index in 0 until items.length()) {
                items.optJSONObject(index)?.optString("profile_id")?.trim()?.takeIf(String::isNotBlank)?.let(::add)
            }
        }
    }

    override suspend fun listConversations(bearer: String, cursor: String?, includeArchived: Boolean): Pair<List<Conversation>, String?> {
        val suffix = buildString { if (!cursor.isNullOrBlank()) append("&cursor=").append(java.net.URLEncoder.encode(cursor, "UTF-8")); if (includeArchived) append("&include_archived=true") }
        val body = request("GET", "/api/mobile/v1/conversations?limit=30${suffix}", bearer = bearer)
        return Pair(parseConversations(body.optJSONArray("items") ?: org.json.JSONArray()), body.optString("next_cursor").ifBlank { null })
    }

    override suspend fun searchConversations(bearer: String, query: String): List<Conversation> {
        val body = request("GET", "/api/mobile/v1/conversations/search?q=${java.net.URLEncoder.encode(query, "UTF-8")}", bearer = bearer)
        return parseConversations(body.optJSONArray("items") ?: org.json.JSONArray())
    }

    override suspend fun createConversation(bearer: String, title: String): Conversation = request("POST", "/api/mobile/v1/conversations", JSONObject().put("title", title), bearer).toConversation()
    override suspend fun createTemporaryConversation(bearer: String, title: String): Conversation = request("POST", "/api/mobile/v1/conversations?temporary=true", JSONObject().put("title", title), bearer).toConversation()
    override suspend fun renameConversation(bearer: String, id: String, title: String): Conversation = request("PATCH", "/api/mobile/v1/conversations/$id", JSONObject().put("title", title), bearer).toConversation()
    override suspend fun archiveConversation(bearer: String, id: String): Conversation = request("POST", "/api/mobile/v1/conversations/$id/archive", bearer = bearer).toConversation()
    override suspend fun deleteConversation(bearer: String, id: String): Conversation = request("DELETE", "/api/mobile/v1/conversations/$id", bearer = bearer).getJSONObject("conversation").toConversation()
    override suspend fun conversationMessages(bearer: String, conversationId: String): List<MessageRecord> {
        val body = request("GET", "/api/mobile/v1/conversations/$conversationId/messages", bearer = bearer)
        return parseMessages(body.optJSONArray("items") ?: org.json.JSONArray())
    }

    override suspend fun sendMessage(bearer: String, conversationId: String, body: String, model: String): MessageRun {
        return sendMessageIdempotent(bearer, conversationId, body, model, java.util.UUID.randomUUID().toString())
    }

    override suspend fun sendMessageIdempotent(bearer: String, conversationId: String, body: String, model: String, idempotencyKey: String): MessageRun {
        return request(
            "POST",
            "/api/mobile/v1/conversations/$conversationId/runs",
            JSONObject().put("body", body).put("model", model).put("idempotency_key", idempotencyKey),
            bearer,
            mapOf("Idempotency-Key" to idempotencyKey),
        ).toMessageRun()
    }

    override suspend fun sendMessageIdempotentWithProfile(
        bearer: String,
        conversationId: String,
        body: String,
        model: String,
        reasoningProfile: String,
        idempotencyKey: String,
    ): MessageRun {
        return request(
            "POST",
            "/api/mobile/v1/conversations/$conversationId/runs",
            JSONObject()
                .put("body", body)
                .put("model", model)
                .put("reasoning_profile", reasoningProfile)
                .put("idempotency_key", idempotencyKey),
            bearer,
            mapOf("Idempotency-Key" to idempotencyKey),
        ).toMessageRun()
    }

    override suspend fun startConversationFromFirstMessage(
        bearer: String,
        draftSessionId: String,
        body: String,
        model: String,
        idempotencyKey: String,
        temporary: Boolean,
    ): FirstMessageResult {
        val response = request(
            "POST",
            "/api/mobile/v1/conversations/from-first-message",
            JSONObject()
                .put("draft_session_id", draftSessionId)
                .put("body", body)
                .put("model", model)
                .put("idempotency_key", idempotencyKey)
                .put("temporary", temporary),
            bearer,
            mapOf("Idempotency-Key" to idempotencyKey),
        )
        return FirstMessageResult(response.getJSONObject("conversation").toConversation(), response.getJSONObject("run").toMessageRun())
    }

    override suspend fun startConversationFromFirstMessageWithProfile(
        bearer: String,
        draftSessionId: String,
        body: String,
        model: String,
        reasoningProfile: String,
        idempotencyKey: String,
        temporary: Boolean,
    ): FirstMessageResult {
        val response = request(
            "POST",
            "/api/mobile/v1/conversations/from-first-message",
            JSONObject()
                .put("draft_session_id", draftSessionId)
                .put("body", body)
                .put("model", model)
                .put("reasoning_profile", reasoningProfile)
                .put("idempotency_key", idempotencyKey)
                .put("temporary", temporary),
            bearer,
            mapOf("Idempotency-Key" to idempotencyKey),
        )
        return FirstMessageResult(response.getJSONObject("conversation").toConversation(), response.getJSONObject("run").toMessageRun())
    }

    override suspend fun runStatus(bearer: String, runId: String): Pair<MessageRun, List<MessageRecord>> {
        val body = request("GET", "/api/mobile/v1/runs/$runId", bearer = bearer)
        val items = body.optJSONArray("messages") ?: org.json.JSONArray()
        return Pair(body.getJSONObject("run").toMessageRun(), parseMessages(items))
    }

    override suspend fun runEvents(bearer: String, runId: String, after: Long): Pair<MessageRun, List<RunEvent>> {
        val stream = requestSse("/api/mobile/v1/runs/$runId/events?after=$after", bearer)
        val events = buildList { stream.forEach { item -> add(RunEvent(item.optLong("id"), item.optString("type"), item.optString("delta"))) } }
        val status = runStatus(bearer, runId).first
        return Pair(status, events)
    }

    override suspend fun messageCitations(bearer: String, messageId: String): List<MessageCitation> = withContext(Dispatchers.IO) {
        val response = request("GET", "/api/mobile/v1/messages/$messageId/citations", bearer = bearer)
        val values = response.getJSONArray("citations")
        buildList { for (index in 0 until values.length()) { val item = values.getJSONObject(index); add(MessageCitation(item.optString("url"), item.optString("title"))) } }
    }

    override suspend fun aiPreference(bearer: String): AIPreference = request("GET", "/api/mobile/v1/preferences/ai", bearer = bearer).toAIPreference()

    override suspend fun updateAIPreference(bearer: String, modelId: String, reasoningProfile: String, version: Long): AIPreference =
        request("PUT", "/api/mobile/v1/preferences/ai", JSONObject().put("model_id", modelId).put("reasoning_profile", reasoningProfile).put("version", version), bearer).toAIPreference()

    override suspend fun updateConversationAISettings(bearer: String, conversationId: String, modelId: String, reasoningProfile: String, version: Long): Conversation =
        request("PATCH", "/api/mobile/v1/conversations/$conversationId/settings", JSONObject().put("model_id", modelId).put("reasoning_profile", reasoningProfile).put("version", version), bearer).toConversation()

    override suspend fun conversationBranches(bearer: String, conversationId: String): List<ConversationBranch> =
        parseBranches(request("GET", "/api/mobile/v1/conversations/$conversationId/branches", bearer = bearer).optJSONArray("items"))

    override suspend fun createConversationBranch(bearer: String, conversationId: String, forkedFromMessageId: String): ConversationBranch =
        request("POST", "/api/mobile/v1/conversations/$conversationId/branches", JSONObject().put("forked_from_message_id", forkedFromMessageId), bearer).toConversationBranch()

    override suspend fun activateConversationBranch(bearer: String, conversationId: String, branchId: String): Conversation =
        request("PATCH", "/api/mobile/v1/conversations/$conversationId/branches/$branchId", bearer = bearer).toConversation()

    override suspend fun rerunMessage(bearer: String, messageId: String, modelId: String, reasoningProfile: String): MessageRun =
        request("POST", "/api/mobile/v1/messages/$messageId/rerun", JSONObject().put("model_id", modelId).put("reasoning_profile", reasoningProfile), bearer).toMessageRun()

    override suspend fun messageRunMetadata(bearer: String, messageId: String): MessageRun =
        request("GET", "/api/mobile/v1/messages/$messageId/run-metadata", bearer = bearer).toMessageRun()

    override suspend fun createComparison(bearer: String, conversationId: String, prompt: String, modelIds: List<String>, reasoningProfile: String): ComparisonGroup =
        request("POST", "/api/mobile/v1/comparisons", JSONObject().put("conversation_id", conversationId).put("prompt", prompt).put("model_ids", org.json.JSONArray(modelIds)).put("reasoning_profile", reasoningProfile), bearer).toComparisonGroup()

    override suspend fun comparison(bearer: String, comparisonId: String): ComparisonGroup =
        request("GET", "/api/mobile/v1/comparisons/$comparisonId", bearer = bearer).toComparisonGroup()

    override suspend fun adoptComparison(bearer: String, comparisonId: String, runId: String): ComparisonGroup =
        request("POST", "/api/mobile/v1/comparisons/$comparisonId/adopt", JSONObject().put("run_id", runId), bearer).toComparisonGroup()

    override suspend fun synthesizeComparison(bearer: String, comparisonId: String, runIds: List<String>): ComparisonGroup =
        request("POST", "/api/mobile/v1/comparisons/$comparisonId/synthesize", JSONObject().put("run_ids", org.json.JSONArray(runIds)), bearer).toComparisonGroup()

    override suspend fun serviceStatus(bearer: String): List<ModelHealthStatus> =
        parseHealth(request("GET", "/api/mobile/v1/service-status/models", bearer = bearer).optJSONArray("items"))

    override suspend fun modelAvailability(bearer: String, modelId: String): ModelAvailability {
        val encodedModelId = java.net.URLEncoder.encode(modelId, "UTF-8")
        return request("GET", "/api/mobile/v1/models/$encodedModelId/availability", bearer = bearer).toModelAvailability()
    }


    override suspend fun cancelRun(bearer: String, runId: String): MessageRun =
        request("POST", "/api/mobile/v1/runs/$runId/cancel", bearer = bearer).toMessageRun()

    override suspend fun exportConversation(bearer: String, conversationId: String): String =
        request("POST", "/api/mobile/v1/conversations/$conversationId/exports", bearer = bearer).optString("content")

    override suspend fun exportMessage(bearer: String, messageId: String): String =
        request("POST", "/api/mobile/v1/messages/$messageId/exports", bearer = bearer).optString("content")

    override suspend fun saveDraft(bearer: String, conversationId: String, body: String): String =
        request("PUT", "/api/mobile/v1/conversations/$conversationId/draft", JSONObject().put("body", body), bearer).optString("body")

    override suspend fun loadDraft(bearer: String, conversationId: String): String =
        request("GET", "/api/mobile/v1/conversations/$conversationId/draft", bearer = bearer).optString("body")
    override suspend fun submitFeedback(bearer: String, messageId: String, value: String): Boolean { request("POST", "/api/mobile/v1/messages/$messageId/feedback", JSONObject().put("value", value), bearer); return true }
    override suspend fun regenerate(bearer: String, messageId: String): MessageRun = request("POST", "/api/mobile/v1/messages/$messageId/regenerate", bearer = bearer).toMessageRun()
    override suspend fun speak(bearer: String, messageId: String): Boolean { request("POST", "/api/mobile/v1/messages/$messageId/speech", bearer = bearer); return true }

    private fun parseConversations(items: org.json.JSONArray): List<Conversation> = buildList { for (index in 0 until items.length()) add(items.getJSONObject(index).toConversation()) }
    private fun parseBranches(items: org.json.JSONArray?): List<ConversationBranch> = buildList { if (items == null) return@buildList; for (index in 0 until items.length()) items.optJSONObject(index)?.toConversationBranch()?.let(::add) }
    private fun parseHealth(items: org.json.JSONArray?): List<ModelHealthStatus> = buildList { if (items == null) return@buildList; for (index in 0 until items.length()) items.optJSONObject(index)?.toHealth()?.let(::add) }
    private fun JSONObject.toModelAvailability() = ModelAvailability(
        modelId = optString("model_id"),
        status = optString("status", "available"),
        fallbackModels = parseModelCatalog(optJSONArray("fallback_models")),
    )
    private fun parseCandidates(items: org.json.JSONArray?): List<ComparisonCandidate> = buildList { if (items == null) return@buildList; for (index in 0 until items.length()) items.optJSONObject(index)?.let { add(ComparisonCandidate(it.optString("run_id"), it.optString("model_id"), it.optString("reasoning_profile", "auto"), it.optString("status"), it.optString("message_id"), it.optString("body"), it.optString("error_code"))) } }
    private fun parseMessages(items: org.json.JSONArray): List<MessageRecord> = buildList { for (index in 0 until items.length()) add(items.getJSONObject(index).toMessageRecord()) }
    private fun parseModelCatalog(items: org.json.JSONArray?): List<ModelOption> = buildList {
        if (items == null) return@buildList
        for (index in 0 until items.length()) {
            val item = items.optJSONObject(index) ?: continue
            val id = item.optString("id").trim()
            val name = item.optString("name").trim()
            if (id.isBlank() || name.isBlank()) continue
            val profiles = item.optJSONArray("reasoning_profiles")?.let { values ->
                buildList {
                    for (profileIndex in 0 until values.length()) {
                        values.optString(profileIndex).trim().takeIf(String::isNotBlank)?.let(::add)
                    }
                }
            }.orEmpty().ifEmpty { listOf("auto") }
            val capabilities = item.optJSONArray("capabilities")?.let { values ->
                buildList {
                    for (capabilityIndex in 0 until values.length()) {
                        val capability = values.optJSONObject(capabilityIndex) ?: continue
                        val capabilityId = capability.optString("id").trim()
                        if (capabilityId.isBlank()) continue
                        add(
                            ModelCapabilityOption(
                                id = capabilityId,
                                label = capability.optString("label").trim().ifBlank { capabilityId },
                                probedAt = capability.optString("probed_at"),
                            ),
                        )
                    }
                }
            }.orEmpty()
            add(
                ModelOption(
                    id = id,
                    name = name,
                    enabled = item.optBoolean("enabled", true),
                    description = item.optString("description"),
                    reasoningProfiles = profiles,
                    providerId = item.optString("provider_id"),
                    providerName = item.optString("provider_name"),
                    purpose = item.optString("purpose"),
                    speedTier = item.optString("speed_tier", "balanced"),
                    capabilities = capabilities,
                ),
            )
        }
    }

    private fun parseComposerTools(items: org.json.JSONArray?): List<ComposerToolOption> {
        if (items == null) return defaultComposerToolOptions()
        val parsed = buildList {
            for (index in 0 until items.length()) {
                val item = items.optJSONObject(index) ?: continue
                val id = item.optString("id").trim()
                val label = item.optString("label").trim()
                if (id.isBlank() || label.isBlank()) continue
                add(ComposerToolOption(id, label, item.optBoolean("enabled"), item.optString("prompt")))
            }
        }
        return parsed.ifEmpty(::defaultComposerToolOptions)
    }

    private fun parseConsumerCopy(items: JSONObject?): Map<String, String> {
        val defaults = defaultConsumerCopy()
        if (items == null) return defaults
        val parsed = defaults.mapValues { (key, fallback) -> items.optString(key).trim().ifEmpty { fallback } }
        return if (parsed.values.any { it.length > 80 }) defaults else parsed
    }

    private suspend fun request(
        method: String,
        path: String,
        body: JSONObject? = null,
        bearer: String? = null,
        headers: Map<String, String> = emptyMap(),
    ): JSONObject = withContext(Dispatchers.IO) {
        var lastFailure: Throwable? = null
        repeat(2) { attempt ->
            try {
                return@withContext requestOnce(method, path, body, bearer, headers)
            } catch (failure: java.io.IOException) {
                lastFailure = failure
                if (attempt == 0 && method == "GET") Thread.sleep(250)
            }
        }
        throw lastFailure ?: java.io.IOException("网络请求失败")
    }

    private fun requestOnce(
        method: String,
        path: String,
        body: JSONObject?,
        bearer: String?,
        headers: Map<String, String>,
    ): JSONObject {
        val connection = (URL(baseUrl + path).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            if (!bearer.isNullOrBlank()) setRequestProperty("Authorization", "Bearer $bearer")
            headers.forEach { (name, value) -> setRequestProperty(name, value) }
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json; charset=utf-8")
                outputStream.bufferedWriter(Charsets.UTF_8).use { it.write(body.toString()) }
            }
        }
        return try {
            val status = connection.responseCode
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val raw = stream?.bufferedReader(Charsets.UTF_8)?.use { it.readText() }.orEmpty()
            val json = if (raw.isBlank()) JSONObject() else JSONObject(raw)
            if (status !in 200..299) {
                val error = json.optJSONObject("error")
                throw ApiException(
                    status,
                    error?.optString("code").orEmpty().ifBlank { "http_$status" },
                    userFacingMessage(error?.optString("code").orEmpty(), error?.optString("message").orEmpty(), status),
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
        deviceId = optString("device_id"),
    )

    private fun JSONObject.toConversation() = Conversation(
        id = getString("id"),
        title = optString("title", "新对话"),
        status = optString("status", "active"),
        updatedAt = optString("updated_at"),
        archivedAt = optString("archived_at").ifBlank { null },
        titleSource = optString("title_source"),
        titleLocked = optBoolean("title_locked"),
        temporary = optBoolean("temporary"),
        activeBranchId = optString("active_branch_id"),
        defaultModelId = optString("default_model_id"),
        defaultReasoningProfile = optString("default_reasoning_profile", "auto"),
        aiSettingsVersion = optLong("ai_settings_version"),
        aiSettingsOverridden = optBoolean("ai_settings_overridden"),
    )
    private fun JSONObject.toAIPreference() = AIPreference(optString("model_id"), optString("reasoning_profile", "auto"), optLong("version"), optString("updated_at"))
    private fun JSONObject.toConversationBranch() = ConversationBranch(getString("id"), optString("conversation_id"), optString("parent_branch_id"), optString("forked_from_message_id"), optString("status", "active"), optString("created_at"), optString("updated_at"))
    private fun JSONObject.toHealth() = ModelHealthStatus(getString("model_id"), optString("provider_id"), optString("status", "available"), optLong("latency_ms"), optString("last_probe_at"), optStringList("capabilities"), optString("error_code"))
    private fun JSONObject.toComparisonGroup() = ComparisonGroup(getString("id"), optString("conversation_id"), optString("prompt"), optString("status"), parseCandidates(optJSONArray("candidates")), optString("adopted_run_id"), optString("synthesis_run_id"), optString("created_at"), optString("updated_at"))
    private fun JSONObject.optStringList(name: String): List<String> = buildList { optJSONArray(name)?.let { values -> for (index in 0 until values.length()) values.optString(index).takeIf(String::isNotBlank)?.let(::add) } }
    private fun JSONObject.toMessageRun() = MessageRun(
        id = getString("id"),
        conversationId = optString("conversation_id"),
        status = optString("status"),
        cursor = optLong("cursor"),
        assistantMessageId = optString("assistant_message_id").ifBlank { null },
        modelId = optString("model"),
        reasoningProfile = optString("reasoning_profile", "auto"),
    )
    private fun JSONObject.toMessageRecord() = MessageRecord(getString("id"), optString("conversation_id"), optString("role"), optString("body"), optString("created_at"))

    private suspend fun requestSse(path: String, bearer: String): List<JSONObject> = withContext(Dispatchers.IO) {
        val connection = (URL(baseUrl + path).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"; connectTimeout = 10_000; readTimeout = 20_000
            setRequestProperty("Accept", "text/event-stream"); setRequestProperty("Cache-Control", "no-cache")
            setRequestProperty("Authorization", "Bearer $bearer")
        }
        try {
            val status = connection.responseCode
            val raw = (if (status in 200..299) connection.inputStream else connection.errorStream)?.bufferedReader(Charsets.UTF_8)?.use { it.readText() }.orEmpty()
            if (status !in 200..299) throw ApiException(status, "http_$status", "连接中断，请稍后重试")
            raw.split("\n\n").mapNotNull { block -> block.lineSequence().firstOrNull { it.startsWith("data:") }?.removePrefix("data:")?.trim()?.takeIf { it.isNotBlank() }?.let(::JSONObject) }
        } finally { connection.disconnect() }
    }

}

internal fun userFacingMessage(code: String, serverMessage: String, status: Int): String = when (code) {
    "email_already_registered" -> "这个邮箱已经注册过了，可以直接登录"
    "invalid_email" -> "请输入正确的邮箱地址"
    "otp_invalid" -> "验证码不正确，请重新输入"
    "otp_expired", "otp_expired_or_locked" -> "验证码已过期，请重新获取"
    "auth_rate_limited" -> "操作太频繁了，请稍后再试"
    "session_invalid", "refresh_token_invalid" -> "登录状态已失效，请重新登录"
    "email_delivery_failed", "email_unconfigured" -> "验证码暂时发送失败，请稍后再试"
    "verification_incorrect" -> "答案不正确，请再试一次"
    "verification_locked" -> "尝试次数过多，请重新开始"
    "challenge_expired", "challenge_not_found" -> "验证已过期，请重新开始"
    else -> serverMessage.takeIf { it.any { character -> character.code > 127 } } ?: "暂时无法完成，请稍后再试 ($status)"
}

class SessionStore(context: Context) {
    private val preferences = context.getSharedPreferences("ylven_identity", Context.MODE_PRIVATE)
    // P03 starts the permanent signer baseline after the approved P02 uninstall migration.
    private val keyAlias = "ylven_identity_session_v2"

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
                deviceId = json.optString("device_id"),
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
            .put("device_id", session.deviceId)
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
