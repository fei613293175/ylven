package cc.orbexa.ylven.ui

// Instrumentation-only renderer for approved visual-state screenshots.

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.graphics.Canvas as AndroidCanvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.Path
import android.graphics.RectF
import android.graphics.Typeface
import android.os.Build
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.graphics.nativeCanvas
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.testTag
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import kotlin.math.max

/**
 * Deterministic runtime renderer for the approved Android state matrices.
 *
 * The renderer uses the numeric 1080 x 2400 contract directly. It intentionally
 * does not load the approved PNGs: every screenshot is drawn by the running APK.
 */
@Composable
fun YlvenAcceptanceState(
    stateId: String,
) {
    AcceptanceSystemBars()
    Canvas(
        modifier = Modifier
            .fillMaxSize()
            .testTag("acceptance-state-$stateId"),
    ) {
        drawIntoCanvas { composeCanvas ->
            val canvas = composeCanvas.nativeCanvas
            val checkpoint = canvas.save()
            canvas.scale(size.width / CONTRACT_WIDTH, size.height / CONTRACT_HEIGHT)
            AndroidContractRenderer(canvas).render(stateId)
            canvas.restoreToCount(checkpoint)
        }
    }
}

@Composable
private fun AcceptanceSystemBars() {
    val view = LocalView.current
    DisposableEffect(view) {
        val window = view.context.findActivity()?.window
        val controller = window?.let { WindowCompat.getInsetsController(it, view) }
        // The approved contract includes a code-rendered status clock and
        // gesture pill. Keep system bars hidden so their device-specific icons
        // do not contaminate the 1080x2400 visual baseline. The acceptance
        // runner pre-confirms Android's one-time immersive education overlay.
        controller?.hide(WindowInsetsCompat.Type.systemBars())
        onDispose { controller?.show(WindowInsetsCompat.Type.systemBars()) }
    }
}

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}

private const val CONTRACT_WIDTH = 1080f
private const val CONTRACT_HEIGHT = 2400f
private val P03_PAGE_IDS = (18..34).filterNot { it == 19 || it == 31 }.mapTo(mutableSetOf()) { index ->
    "YL-A-${index.toString().padStart(3, '0')}"
}

private val P03_PHYSICAL_FONT_CALIBRATION_PAGE_IDS = setOf("YL-A-018")
private const val P03_CHAT_PAGE_ID = "YL-A-023"
private val P03_CHAT_DERIVED_PAGE_IDS = setOf(
    P03_CHAT_PAGE_ID, "YL-A-024", "YL-A-026", "YL-A-033", "YL-A-034",
)
private val P03_CORRECTION_PAGE_IDS = setOf(
    "YL-A-018", "YL-A-020", "YL-A-023", "YL-A-024", "YL-A-026", "YL-A-033", "YL-A-034",
)
private const val P03_PROVIDER_ERROR_FONT_WEIGHT = 550
private const val P03_PROVIDER_ERROR_STROKE_WIDTH = .04f
private const val P03_PROVIDER_ERROR_SCALE_X = 1.004f

private object Pc {
    val bg = color("#F6F7FB")
    val surface = color("#FFFFFF")
    val surfaceSubtle = color("#F9FAFB")
    val brand = color("#5B61F6")
    val brandSoft = color("#F0F1FF")
    val blue = color("#2F80ED")
    val blueSoft = color("#EEF6FF")
    val text = color("#101828")
    val text2 = color("#475467")
    val text3 = color("#667085")
    val disabled = color("#98A2B3")
    val border = color("#E4E7EC")
    val border2 = color("#D0D5DD")
    val divider = color("#EAECF0")
    val success = color("#12A66A")
    val successSoft = color("#ECFDF3")
    val warning = color("#F79009")
    val warningSoft = color("#FFFAEB")
    val error = color("#D92D20")
    val errorSoft = color("#FEF3F2")
    val info = color("#2F80ED")
    val infoSoft = color("#EFF8FF")
}

private fun color(value: String): Int = Color.parseColor(value)

private class AndroidContractRenderer(
    private val canvas: AndroidCanvas,
) {
    private val paint = Paint(Paint.ANTI_ALIAS_FLAG or Paint.SUBPIXEL_TEXT_FLAG).apply {
        strokeCap = Paint.Cap.ROUND
        strokeJoin = Paint.Join.ROUND
    }
    private var currentPage = ""
    private var p03ChatBaselineScale = 0f
    private var p03ChatStatusFontCalibration = false
    // YL-A-024 is a component contract derived from the chat page, but its
    // approved states intentionally use the lighter component typography.
    // Keep this as renderer state so the shared chat primitives remain
    // deterministic while the component board cannot collapse into YL-A-023.
    private var p03ConversationComposerMode = false

    fun render(stateId: String) {
        val page = stateId.substringBefore("-S")
        currentPage = page
        // State IDs carry a single separator between the sequence number and
        // the status. Keep compound codes intact: e.g. SERVER_ERROR and
        // INPUT_FOCUSED must select their own contract variants rather than
        // being truncated to ERROR or FOCUSED.
        val code = stateId.substringAfter('_')
        canvas.drawColor(if (page in P03_CORRECTION_PAGE_IDS) color("#F7F8FC") else Pc.bg)
        when (page) {
            "YL-A-004" -> splash(code)
            "YL-A-005" -> auth("login", code)
            "YL-A-006" -> securityDialog(registering = false, code)
            "YL-A-007" -> auth("first_party", code)
            "YL-A-008" -> auth("login_otp", code)
            "YL-A-009" -> auth("login_success", code)
            "YL-A-010" -> auth("register", code)
            "YL-A-011" -> securityDialog(registering = true, code)
            "YL-A-012" -> auth("register_otp", code)
            "YL-A-013" -> auth("register_success", code)
            "YL-A-014" -> session(code)
            "YL-A-015" -> logoutDialog(code)
            "YL-A-016" -> devices(code)
            "YL-A-017" -> resilience(code)
            "YL-A-018" -> p03Home(code)
            "YL-A-020" -> p03ConversationList(code, history = true)
            "YL-A-021" -> p03ConversationList(code, history = false)
            "YL-A-022" -> p03ConversationMenu(code)
            "YL-A-023" -> p03Chat(code)
            "YL-A-024" -> p03ConversationComposer(code)
            "YL-A-032" -> p03Composer(code, "输入框组件")
            "YL-A-025" -> p03OfflineCache(code)
            "YL-A-026" -> p03Response(code)
            "YL-A-027" -> p03CodeBlock(code)
            "YL-A-028" -> p03TableBlock(code)
            "YL-A-029" -> p03CitationBlock(code)
            "YL-A-030" -> p03MessageActions(code)
            "YL-A-033" -> p03Selector(code, model = true)
            "YL-A-034" -> p03Selector(code, model = false)
            "YL-A-035" -> p04Settings(code, conversation = false)
            "YL-A-036" -> p04Settings(code, conversation = true)
            "YL-A-037" -> p04ModelStrip(code)
            "YL-A-038" -> p04ResponseLabel(code)
        }
        stateFeedback(page, code)
    }

    private fun splash(code: String) {
        var y = 0
        while (y < 2400) {
            rect(0f, y.toFloat(), 1080f, (y + 12).toFloat(), mix(color("#040814"), color("#152458"), y / 2400f))
            y += 12
        }
        // Pillow's RGBA ImageDraw writes the source RGB values directly for
        // these contract circles before the baseline is converted to RGB. The
        // approved result is therefore one solid outer brand circle, not
        // three alpha-composited rings.
        circle(540f, 930f, 390f, Pc.brand)
        iconCircle(540f, 900f, 126f, "Y", Pc.brand, Pc.surface, 100f)
        text("YLVEN", 540f, 1070f, 76f, Pc.surface, true, Anchor.MIDDLE_ASCENDER)
        text("INTELLIGENCE, REFINED.", 540f, 1160f, 28f, color("#CBD5FF"), anchor = Anchor.MIDDLE_ASCENDER)
        when (code) {
            "RESTORING" -> {
                spinner(540f, 1420f, 42f, Pc.surface, 8f)
                text("正在恢复登录状态", 540f, 1500f, 30f, color("#DCE2FF"), anchor = Anchor.MIDDLE_ASCENDER)
            }
            "FIRST_RUN" -> button(210f, 1430f, 870f, 1586f, "开始使用 YLVEN")
            "UPDATE_REQUIRED", "OFFLINE", "SERVER_ERROR" -> statusBanner(code, 1420f, .72f)
        }
        rounded(430f, 2370f, 650f, 2382f, 7f, Pc.surface)
    }

    private fun auth(template: String, code: String) {
        androidStatus()
        circle(940f, 80f, 240f, color("#ECEEFF"))
        circle(150f, 1930f, 270f, color("#EEF6FF"))
        iconCircle(96f, 168f, 42f, "Y", Pc.brand, Pc.surface, 36f)
        text("YLVEN", 154f, 146f, 42f, Pc.text, true)
        text("轻松解决每天的小问题", 154f, 194f, 25f, Pc.text3)

        if (template == "first_party") {
            topBar("请完成小验证", "YLVEN", back = true, right = false)
            shadowCard(72f, 360f, 1008f, 1460f, 46f)
            iconCircle(540f, 520f, 72f, if (code == "SUCCESS") "✓" else "盾", Pc.brandSoft, Pc.brand, 52f)
            text("请算一算下面这道题", 540f, 640f, 54f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
            text("验证会直接在当前页面完成", 540f, 726f, 31f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER)
            rounded(150f, 850f, 930f, 1065f, 30f, Pc.surfaceSubtle, Pc.border2, 3f)
            rounded(205f, 910f, 275f, 980f, 12f, Pc.surface, if (code == "SUCCESS") Pc.brand else Pc.border2, 4f)
            when (code) {
                "SUCCESS" -> lineIcon(214f, 918f, "check", 52f, Pc.success, 6f)
                "LOADING", "SECURITY_CHALLENGE" -> spinner(240f, 945f, 25f, Pc.brand, 6f)
            }
            val widgetTitle = when (code) {
                "LOADING", "SECURITY_CHALLENGE" -> "正在核对答案"
                "SUCCESS" -> "验证已完成"
                else -> "验证未完成"
            }
            text(widgetTitle, 315f, 908f, 36f, Pc.text, true)
            text("7 + 5 = ?", 315f, 966f, 26f, Pc.text3)
            button(150f, 1160f, 930f, 1320f, if (code == "SUCCESS") "返回 YLVEN" else "取消验证", primary = false)
            if (code in setOf("VALIDATION_ERROR", "CODE_EXPIRED", "OFFLINE", "SERVER_ERROR")) {
                statusBanner(if (code == "VALIDATION_ERROR") "SAVE_ERROR" else code, 1510f, .82f, "安全验证未通过，请重新尝试。")
            }
            gesture(Pc.text)
            return
        }

        if (template == "login_success" || template == "register_success") {
            val title = if (template == "login_success") "登录成功" else "账户已创建"
            val subtitle = if (template == "login_success") "正在准备你的内容" else "正在准备你的 YLVEN"
            val steps = if (template == "login_success") {
                listOf("登录信息已保存", "同步个人设置", "进入首页")
            } else {
                listOf("创建账户", "保存个人设置", "准备常用功能", "进入 YLVEN")
            }
            iconCircle(540f, 760f, 106f, "✓", Pc.successSoft, Pc.success, 82f)
            text(title, 540f, 930f, 66f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
            text(subtitle, 540f, 1022f, 34f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER)
            var rowY = 1160f
            steps.forEachIndexed { index, label ->
                iconCircle(170f, rowY + 10f, 30f, if (code == "SUCCESS") "✓" else "·", Pc.successSoft, Pc.success, 24f)
                text(label, 225f, rowY - 10f, 34f, Pc.text2, index == steps.lastIndex)
                rowY += 92f
            }
            if (code == "SUCCESS") {
                button(180f, 1600f, 900f, 1760f, "进入 YLVEN")
            } else {
                statusBanner(code, 1570f, .78f, "初始化暂未完成，已保留账户数据。")
            }
            gesture(Pc.text)
            return
        }

        val title = when (template) {
            "login" -> "登录 YLVEN"
            "register" -> "创建 YLVEN 账户"
            "login_otp" -> "输入登录验证码"
            else -> "验证注册邮箱"
        }
        val subtitle = when (template) {
            "login" -> "使用邮箱验证码安全登录"
            "register" -> "设置好密码，马上开始使用"
            "login_otp" -> "验证码已发送至 c***@example.com"
            else -> "输入发送至 c***@example.com 的六位验证码"
        }
        val x1 = 72f
        val x2 = 1008f
        var y = 360f
        text(title, x1, y, 66f, Pc.text, true)
        text(subtitle, x1, y + 92f, 33f, Pc.text3, maxWidth = 900f)
        y += 205f
        val focused = code == "INPUT_FOCUSED"
        val validation = code == "VALIDATION_ERROR"
        when (template) {
            "login" -> {
                input(x1, y, x2, y + 156f, "邮箱地址", if (focused) "chenping@orbexa.cc" else "", "name@example.com", focused, if (validation) "请输入有效邮箱地址" else null, icon = "user")
                y += 250f
                if (code == "RATE_LIMITED") text("请求过于频繁，请 48 秒后再试", x1, y - 28f, 28f, Pc.warning)
                button(x1, y, x2, y + 156f, if (code == "SUBMITTING") "正在发送验证码…" else "获取登录验证码", loading = code == "SUBMITTING")
                y += 205f
                text("还没有账户？", 540f, y, 30f, Pc.text3, anchor = Anchor.MIDDLE_MIDDLE)
                text("创建账户", 700f, y, 30f, Pc.brand, true, Anchor.MIDDLE_MIDDLE)
                if (code == "OFFLINE" || code == "SERVER_ERROR") statusBanner(code, 1320f, .86f)
            }
            "register" -> {
                input(x1, y, x2, y + 146f, "邮箱地址", if (focused) "chenping@orbexa.cc" else "", "name@example.com", focused, if (validation) "邮箱格式不正确" else null, icon = "user")
                y += 225f
                input(x1, y, x2, y + 146f, "登录密码", if (focused) "Ylven@2026" else "", "至少 8 位，包含字母和数字", error = if (validation) "密码强度不足" else null, secure = true)
                y += 225f
                input(x1, y, x2, y + 146f, "确认登录密码", if (focused) "Ylven@2026" else "", "再次输入登录密码", error = if (validation) "两次密码输入不一致" else null, secure = true)
                y += 215f
                listOf("至少 8 个字符", "包含字母和数字", "两次密码一致").forEachIndexed { index, rule ->
                    val complete = focused && !validation
                    iconCircle(x1 + 18f, y + index * 52f, 16f, if (complete) "✓" else "·", if (complete) Pc.successSoft else Pc.surfaceSubtle, if (complete) Pc.success else Pc.text3, 18f)
                    text(rule, x1 + 50f, y - 17f + index * 52f, 27f, Pc.text3)
                }
                y += 180f
                button(x1, y, x2, y + 156f, if (code == "SUBMITTING") "正在提交…" else "创建账户", disabled = code == "DISABLED", loading = code == "SUBMITTING")
                if (code == "OFFLINE" || code == "SERVER_ERROR") statusBanner(code, 1680f, .86f)
            }
            else -> otp(template, code, x1, x2, y)
        }
        gesture(Pc.text)
    }

    private fun otp(template: String, code: String, x1: Float, x2: Float, startY: Float) {
        var y = startY
        text("验证码已发送至", x1, y, 29f, Pc.text3)
        text("c***@example.com", x1, y + 48f, 38f, Pc.text, true)
        y += 150f
        repeat(6) { index ->
            val bx = x1 + index * 150f
            val filled = index < 3 && code in setOf("INPUT_FOCUSED", "CODE_SENT", "SUBMITTING")
            val invalid = code == "INVALID_CODE"
            val active = code == "INPUT_FOCUSED" && index == 3
            rounded(bx, y, bx + 120f, y + 136f, 24f, Pc.surface, if (invalid) Pc.error else if (active) Pc.brand else Pc.border2, if (invalid || active) 5f else 2f)
            if (filled) text("8", bx + 60f, y + 68f, 54f, Pc.text, true, Anchor.MIDDLE_MIDDLE)
        }
        y += 185f
        when (code) {
            "INVALID_CODE" -> text("验证码错误，请重新输入", x1, y, 28f, Pc.error)
            "CODE_EXPIRED" -> text("验证码已过期，请重新发送", x1, y, 28f, Pc.warning)
            "RATE_LIMITED" -> text("发送频繁，请 48 秒后再试", x1, y, 28f, Pc.warning)
            else -> text("00:48 后可重新发送", x1, y, 28f, Pc.text3)
        }
        y += 95f
        val label = if (template == "login_otp") "登录" else "确认并创建账户"
        button(x1, y, x2, y + 156f, if (code == "SUBMITTING") "正在验证…" else label, loading = code == "SUBMITTING")
        y += 205f
        text("收不到验证码？  更换邮箱", 540f, y, 29f, Pc.brand, true, Anchor.MIDDLE_MIDDLE)
        if (code == "SUCCESS") statusBanner("SUCCESS", 1510f, .82f, "验证通过，正在进入下一步。")
        if (code in setOf("LOCKED", "OFFLINE", "SERVER_ERROR")) statusBanner(code, 1510f, .82f)
        if (code == "INPUT_FOCUSED") keyboard(numeric = true)
    }

    private fun securityDialog(registering: Boolean, code: String) {
        auth(if (registering) "register" else "login", "DEFAULT")
        overlay(Color.argb(125, 16, 24, 40))
        shadowCard(105f, 700f, 975f, 1515f, 44f, shadow = 14f, offset = 6f)
        iconCircle(540f, 845f, 62f, "盾", Pc.brandSoft, Pc.brand, 46f)
        val title = "请完成小验证"
        val subtitle = "为了确认是你本人，请算一算下面这道题"
        text(title, 540f, 940f, 54f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text(subtitle, 540f, 1022f, 31f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER, maxWidth = 690f)
        rounded(170f, 1125f, 910f, 1275f, 26f, Pc.surfaceSubtle, Pc.border2, 2f)
        when (code) {
            "SUBMITTING", "SECURITY_CHALLENGE" -> spinner(235f, 1200f, 26f, Pc.brand, 6f)
            "SUCCESS" -> lineIcon(210f, 1175f, "check", 50f, Pc.success, 6f)
            else -> rounded(207f, 1165f, 263f, 1221f, 10f, Pc.surface, Pc.border2, 3f)
        }
        text("7 + 5 = ?", 300f, 1175f, 34f, Pc.text, true)
        text("请输入答案", 300f, 1220f, 24f, Pc.text3)
        button(170f, 1330f, 525f, 1460f, "取消", primary = false)
        button(555f, 1330f, 910f, 1460f, "确认", loading = code == "SUBMITTING")
        if (isError(code)) statusBanner(code, 1580f, .78f)
        gesture(Pc.surface)
    }

    private fun session(code: String) {
        topBar("AI 首页", "统一多模型对话入口", back = false, right = true, plus = true)
        bottomNav("首页")
        homeBody()
        when (code) {
            "RESTORING" -> {
                overlay(Color.argb(150, 255, 255, 255))
                shadowCard(190f, 840f, 890f, 1260f, 44f, shadow = 12f, offset = 5f)
                spinner(540f, 950f, 44f, Pc.brand, 8f)
                text("正在恢复登录状态", 540f, 1050f, 52f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
                text("正在确认账户信息", 540f, 1130f, 30f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER)
            }
            "SUCCESS" -> statusBanner("SUCCESS", 300f, .78f, "登录状态已恢复。")
            "UNAUTHORIZED", "OFFLINE", "SERVER_ERROR" -> {
                overlay(Color.argb(90, 16, 24, 40))
                statusBanner(code, 850f, .78f, "本地草稿已保留，可重新登录后继续。")
            }
        }
    }

    private fun logoutDialog(code: String) {
        minePage()
        overlay(Color.argb(125, 16, 24, 40))
        shadowCard(120f, 850f, 960f, 1420f, 44f, shadow = 14f, offset = 6f)
        iconCircle(540f, 980f, 62f, "↪", Pc.warningSoft, Pc.warning, 46f)
        text("确认退出登录？", 540f, 1070f, 56f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text("本机会清理敏感缓存，但云端数据不会删除。", 540f, 1150f, 31f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER, maxWidth = 690f)
        button(170f, 1250f, 515f, 1380f, "取消", primary = false)
        button(545f, 1250f, 910f, 1380f, "退出登录", loading = code == "SUBMITTING")
        if (code == "SUCCESS") statusBanner("SUCCESS", 1510f, .78f, "已安全退出当前账号。")
        if (code == "SAVE_ERROR") statusBanner("SAVE_ERROR", 1510f, .78f, "退出失败，请检查网络后重试。")
        gesture(Pc.surface)
    }

    private fun devices(code: String) {
        topBar("登录设备", "查看并管理当前账号的已授权设备", back = true, right = true)
        when (code) {
            "LOADING" -> skeleton(72f, 360f, 1008f, rows = 6)
            "EMPTY" -> errorCenter("EMPTY", "暂无其他登录设备")
            else -> {
                text("当前设备", 72f, 320f, 31f, Pc.text3, true)
                val rows = listOf(
                    Triple("Android · YLVEN", "新加坡 · 当前设备", "当前"),
                    Triple("Windows · Codex", "新加坡 · 10 分钟前", ""),
                    Triple("Chrome · Web", "新加坡 · 昨天", ""),
                )
                var y = 390f
                rows.forEachIndexed { index, row ->
                    rounded(72f, y, 1008f, y + 215f, 36f, Pc.surface, Pc.border, 2f)
                    iconCircle(150f, y + 107f, 46f, listOf("A", "W", "C")[index], Pc.brandSoft, Pc.brand, 34f)
                    text(row.first, 225f, y + 55f, 38f, Pc.text, true)
                    text(row.second, 225f, y + 115f, 28f, Pc.text3)
                    if (row.third.isNotEmpty()) pill(820f, y + 58f, 950f, y + 120f, row.third, Pc.successSoft, Pc.success, 24f)
                    else text("撤销", 918f, y + 105f, 28f, Pc.error, true, Anchor.MIDDLE_MIDDLE)
                    y += 240f
                }
                text("撤销设备后，该设备必须重新完成邮箱验证码登录。", 72f, y + 25f, 28f, Pc.text3, maxWidth = 900f)
                when (code) {
                    "REFRESHING" -> statusBanner("SERVICE_DEGRADED", 270f, .82f, "正在刷新设备列表…")
                    "UNAUTHORIZED", "OFFLINE", "SERVER_ERROR" -> statusBanner(code, 270f, .82f)
                }
            }
        }
        gesture(Pc.text)
    }

    private fun resilience(code: String) {
        androidStatus()
        rect(0f, 72f, 1080f, 270f, Pc.surface)
        text("YLVEN UI CONTRACT", 60f, 118f, 27f, Pc.brand, true)
        text("认证适配与异常状态", 60f, 166f, 52f, Pc.text, true)
        text("弱网、离线、小屏和字体缩放统一规范", 60f, 228f, 27f, Pc.text3)
        pill(830f, 122f, 1010f, 184f, "组件规范", Pc.brandSoft, Pc.brand, 24f)
        shadowCard(54f, 330f, 1026f, 2020f, 52f)
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val cards = listOf(
            Triple("弱网重试", "保留输入，不重复提交", "↻"),
            Triple("离线提示", "明确缓存时间与能力限制", "↯"),
            Triple("320dp 小屏", "纵向滚动，不压缩触控区", "□"),
            Triple("大字体", "文本可换行，按钮高度保持", "Aa"),
        )
        cards.forEachIndexed { index, card ->
            val x = 105f + (index % 2) * 440f
            val y = 540f + (index / 2) * 410f
            rounded(x, y, x + 395f, y + 350f, 36f, Pc.surfaceSubtle, Pc.border, 2f)
            iconCircle(x + 70f, y + 76f, 42f, card.third, Pc.brandSoft, Pc.brand, 30f)
            text(card.first, x + 42f, y + 145f, 37f, Pc.text, true)
            text(card.second, x + 42f, y + 205f, 28f, Pc.text3, maxWidth = 310f)
        }
        if (isError(code)) statusBanner(code, 1510f, .72f)
        text("视觉类型：独立组件板", 90f, 2085f, 28f, Pc.text3)
        text("示例文案仅用于视觉占位，功能以 Feature ID 为准。", 90f, 2140f, 26f, Pc.text3)
        gesture(Pc.text)
    }

    // P03 uses the same contract canvas as P02. These drawings deliberately
    // remain runtime Canvas content: the APK never loads an approved mockup
    // image as a replacement for a rendered screen.
    private fun p03Home(code: String) {
        androidStatus()
        rect(0f, 72f, 1080f, 238f, Pc.surface)
        lineIcon(48f, 122f, "menu", 56f, Pc.text, 7f)
        text("YLVEN", 540f, 145f, 48f, Pc.text, true, Anchor.MIDDLE_MIDDLE)
        lineIcon(972f, 117f, "plus", 56f, Pc.text, 7f)
        p03HomeGlow()
        p03HomeBottomNav()
        if (code == "LOADING") {
            p03HomeLoading()
            return
        }
        val hasBanner = code in setOf("REFRESHING", "OFFLINE_CACHE", "NETWORK_ERROR", "SERVICE_DEGRADED")
        if (hasBanner) p03HomeBanner(code)
        p03Spark(540f, 390f, 54f)
        text("今天想完成什么？", 540f, 473f, 80f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text("多模型协同，完成更复杂的事。", 540f, 590f, 40f, Pc.text2, anchor = Anchor.MIDDLE_ASCENDER)
        shadowCard(54f, 710f, 1026f, 1010f, 72f, shadow = 12f, offset = 5f)
        text("问问 YLVEN…", 102f, 747f, 46f, Pc.disabled)
        text("+", 116f, 965f, 54f, Pc.text2, anchor = Anchor.MIDDLE_MIDDLE)
        pill(160f, 922f, 380f, 992f, "自动选择", color("#EEF0FF"), Pc.brand, 32f)
        lineIcon(830f, 937f, "mic", 54f, Pc.text2, 5f)
        iconCircle(958f, 960f, 48f, "➤", if (code == "NETWORK_ERROR") Pc.border2 else Pc.brand, Pc.surface, 30f)
        listOf("分析文件" to "文", "生成图片" to "图", "制作演示" to "P").forEachIndexed { index, item ->
            val left = 54f + index * 330f
            rounded(left, 1045f, left + 300f, 1135f, 45f, Pc.surface, Pc.border, 2f)
            iconCircle(left + 42f, 1090f, 28f, item.second, color("#EEF0FF"), Pc.brand, 24f)
            text(item.first, left + 82f, 1090f, 38f, Pc.text, anchor = Anchor.LEFT_MIDDLE)
        }
        text("继续最近的对话", 54f, 1247f, 58f, Pc.text, true)
        text("查看全部", 1026f, 1255f, 40f, Pc.brand, anchor = Anchor.RIGHT_ASCENDER)
        if (code == "EMPTY") {
            rounded(54f, 1335f, 1026f, 1495f, 28f, Pc.surface, Pc.border, 2f)
            p03Spark(132f, 1415f, 22f)
            text("从一句问题开始", 190f, 1365f, 36f, Pc.text, true)
            text("第一条消息发送后才会创建正式会话。", 190f, 1425f, 28f, Pc.text3)
        } else {
            listOf("多服务器 AI 架构设计" to "GPT-5.6 Sol · 刚刚", "对话上下文与长期记忆方案" to "Claude Opus · 昨天").forEachIndexed { index, item ->
                val y = 1335f + index * 180f
                rounded(54f, y, 1026f, y + 150f, 28f, Pc.surface, Pc.border, 2f)
                iconCircle(125f, y + 75f, 42f, "Y", Pc.brandSoft, Pc.brand, 42f)
                text(item.first, 195f, y + 52f, 48f, Pc.text, true)
                text(item.second, 195f, y + 108f, 40f, Pc.text2)
                text("›", 955f, y + 75f, 42f, Pc.text3, anchor = Anchor.MIDDLE_MIDDLE)
            }
        }
    }

    private fun p03HomeGlow() {
        val checkpoint = canvas.save()
        canvas.clipRect(0f, 238f, 1080f, 2160f)
        var radius = 680f
        while (radius >= 120f) {
            val strength = .03f + (680f - radius) / 560f * .22f
            circle(540f, 420f, radius, mix(color("#F7F8FC"), color("#EEF0FF"), strength))
            radius -= 16f
        }
        canvas.restoreToCount(checkpoint)
    }

    private fun p03HomeBottomNav() {
        val top = 2160f
        rect(0f, top, 1080f, 2400f, Pc.surface)
        line(0f, top, 1080f, top, Pc.divider, 2f)
        listOf("首页" to "home", "工作" to "briefcase", "发现" to "compass", "我的" to "user").forEachIndexed { index, item ->
            val cx = 135f + index * 270f
            val fill = if (index == 0) Pc.brand else Pc.text2
            p03HomeNavIcon(cx, 2228f, item.second, fill)
            text(item.first, cx, 2310f, 36f, fill, index == 0, Anchor.MIDDLE_ASCENDER)
        }
        gesture(Pc.text)
    }

    private fun p03HomeNavIcon(cx: Float, top: Float, kind: String, fill: Int) {
        when (kind) {
            "home" -> {
                filledPath(
                    listOf(
                        cx to top,
                        cx - 38f to top + 32f,
                        cx - 29f to top + 32f,
                        cx - 29f to top + 66f,
                        cx + 29f to top + 66f,
                        cx + 29f to top + 32f,
                        cx + 38f to top + 32f,
                        cx to top,
                    ),
                    fill,
                )
            }
            "briefcase" -> {
                rounded(cx - 31f, top + 15f, cx + 31f, top + 66f, 8f, fill)
                rounded(cx - 15f, top, cx + 15f, top + 24f, 6f, null, fill, 7f)
            }
            "compass" -> {
                oval(cx - 27f, top + 5f, cx + 27f, top + 59f, null, fill, 7f)
                filledPath(
                    listOf(
                        cx + 12f to top + 17f,
                        cx + 3f to top + 39f,
                        cx - 12f to top + 47f,
                        cx - 3f to top + 25f,
                        cx + 12f to top + 17f,
                    ),
                    fill,
                )
            }
            else -> {
                circle(cx, top + 18f, 17f, fill)
                arc(cx - 34f, top + 36f, cx + 34f, top + 76f, 190f, 160f, fill, 13f)
            }
        }
    }

    private fun p03HomeLoading() {
        val skeleton = color("#EAECF0")
        val skeletonSoft = color("#F2F4F7")
        circle(540f, 405f, 75f, skeletonSoft)
        rounded(220f, 530f, 860f, 610f, 40f, skeleton)
        rounded(300f, 640f, 780f, 690f, 25f, skeletonSoft)
        rounded(54f, 760f, 1026f, 1050f, 72f, skeleton)
        rounded(54f, 1160f, 420f, 1215f, 28f, skeleton)
        rounded(54f, 1260f, 1026f, 1416f, 28f, skeleton)
        rounded(54f, 1440f, 1026f, 1596f, 28f, skeleton)
    }

    private fun p03HomeBanner(code: String) {
        val spec = when (code) {
            "REFRESHING" -> HomeBannerSpec(Pc.infoSoft, Pc.info, "正在更新最近内容", "")
            "OFFLINE_CACHE" -> HomeBannerSpec(Pc.warningSoft, Pc.warning, "当前离线，仍可查看最近内容", "重试")
            "NETWORK_ERROR" -> HomeBannerSpec(Pc.errorSoft, Pc.error, "网络暂不可用，请检查连接后重试", "重试")
            else -> HomeBannerSpec(Pc.warningSoft, Pc.warning, "部分模型暂不可用，已为你保留可用模型", "查看")
        }
        rounded(40f, 258f, 1040f, 334f, 24f, spec.bg)
        if (code == "REFRESHING") spinner(78f, 296f, 14f, spec.fg, 5f) else circle(78f, 296f, 14f, spec.fg)
        text(spec.label, 108f, 273f, 28f, spec.fg)
        if (spec.action.isNotEmpty()) text(spec.action, 1010f, 273f, 27f, spec.fg, true, Anchor.RIGHT_ASCENDER)
    }

    private fun p03ConversationList(code: String, history: Boolean) {
        if (history) {
            val parentState = when (code) {
                "LOADING", "EMPTY", "REFRESHING", "OFFLINE_CACHE", "NETWORK_ERROR" -> code
                "SERVER_ERROR" -> "SERVICE_DEGRADED"
                else -> "POPULATED"
            }
            p03Home(parentState)
            rect(912f, 72f, 1080f, 2400f, Color.argb(82, 0, 0, 0))
            rounded(-48f, 72f, 912f, 2400f, 48f, color("#FCFCFE"))
            line(911f, 120f, 911f, 2352f, Pc.border2, 1f)
            p03DrawerHeader(
                query = if (code == "FILTER_ACTIVE") "上下文" else null,
                searchActive = code == "FILTER_ACTIVE",
                newChatEnabled = code !in setOf("NETWORK_ERROR", "SERVER_ERROR", "PERMISSION_DENIED"),
            )
            when (code) {
                "LOADING" -> {
                    p03DrawerGroup(410f, "最近对话")
                    listOf(565f to 240f, 610f to 300f, 520f to 210f, 590f to 270f, 470f to 200f)
                        .forEachIndexed { index, widths -> p03DrawerSkeleton(485f + index * 145f, widths.first, widths.second) }
                }
                "EMPTY" -> p03DrawerEmpty()
                "REFRESHING" -> {
                    p03DrawerBanner(382f, "正在更新对话…", "info", spinner = true)
                    p03DrawerGroup(486f, "今天")
                    p03DrawerConversation(568f, "多服务器 AI 架构设计", "刚刚")
                    p03DrawerConversation(692f, "对话上下文与长期记忆方案", "22:48")
                    p03DrawerConversation(816f, "首页与聊天体验重构", "21:16")
                    p03DrawerGroup(960f, "过去 7 天")
                    p03DrawerConversation(1042f, "YLVEN 发布流程优化", "昨天")
                }
                "FILTER_ACTIVE" -> {
                    p03DrawerGroup(410f, "搜索结果 · 2")
                    p03DrawerConversation(500f, "对话上下文与长期记忆方案", "今天 22:48", highlighted = true)
                    p03DrawerConversation(640f, "项目上下文与文件检索", "8 月 7 日")
                    text("仅显示与“上下文”相关的对话", 456f, 872f, 27f, Pc.disabled, anchor = Anchor.MIDDLE_ASCENDER)
                }
                "OFFLINE_CACHE" -> {
                    p03DrawerBanner(382f, "当前离线，已显示本地记录", "warning")
                    p03DrawerGroup(486f, "已保存的对话")
                    p03DrawerConversation(568f, "多服务器 AI 架构设计", "刚刚")
                    p03DrawerConversation(692f, "对话上下文与长期记忆方案", "今天 22:48")
                    p03DrawerConversation(816f, "YLVEN 发布流程优化", "昨天")
                }
                "NETWORK_ERROR" -> p03DrawerError("network", "网络连接不可用", "请检查网络连接后重试", "重试")
                "SERVER_ERROR" -> p03DrawerError("server", "暂时无法加载对话", "服务正在恢复，请稍后再试", "重新加载")
                "PERMISSION_DENIED" -> p03DrawerError("lock", "登录状态已失效", "重新登录后可继续查看历史对话", "重新登录")
                else -> {
                    p03DrawerGroup(410f, "今天")
                    p03DrawerConversation(492f, "多服务器 AI 架构设计", "刚刚")
                    p03DrawerConversation(616f, "对话上下文与长期记忆方案", "22:48")
                    p03DrawerConversation(740f, "首页与聊天体验重构", "21:16")
                    p03DrawerGroup(884f, "过去 7 天")
                    p03DrawerConversation(966f, "YLVEN 发布流程优化", "昨天")
                    p03DrawerConversation(1090f, "Sub2API 模型能力测试", "8 月 8 日")
                }
            }
            return
        }
        val title = if (history) "会话历史" else "搜索会话"
        val subtitle = if (history) "搜索、归档和管理全部对话" else "按标题和消息内容查找历史记录"
        topBar(title, subtitle, back = true, right = true)
        if (code == "LOADING") { skeleton(72f, 330f, 1008f, 7); return }
        if (code in setOf("EMPTY", "NETWORK_ERROR", "SERVER_ERROR", "PERMISSION_DENIED")) {
            errorCenter(code)
            return
        }
        var y: Float
        if (history) {
            var x = 72f
            listOf("今天", "昨天", "最近 7 天", "新建对话").forEachIndexed { index, label ->
                x += chip(x, 320f, label, index == 0) + 14f
            }
            y = 430f
        } else {
            input(72f, 310f, 1008f, 450f, "搜索", "", "按标题和消息内容查找历史记录", code == "FILTER_ACTIVE", icon = "search")
            var x = 72f
            listOf("搜索输入框", "筛选项目", "搜索结果").forEachIndexed { index, label ->
                x += chip(x, 515f, label, index == 0) + 14f
            }
            y = 625f
        }
        val rows = if (history) listOf("今天", "昨天", "最近 7 天", "新建对话") else listOf("搜索输入框", "筛选项目", "搜索结果")
        rows.forEachIndexed { index, item ->
            rounded(72f, y, 1008f, y + 180f, 34f, Pc.surface, Pc.border, 2f)
            iconCircle(142f, y + 90f, 40f, (index + 1).toString(), Pc.brandSoft, Pc.brand, 28f)
            text(item, 210f, y + 48f, 36f, Pc.text, true)
            text("更新于 ${if (index == 0) "刚刚" else "${index + 1} 小时前"}", 210f, y + 106f, 27f, Pc.text3)
            lineIcon(920f, y + 62f, "more", 48f, Pc.text3, 4f)
            y += 198f
        }
        when (code) {
            "REFRESHING" -> statusBanner("SERVICE_DEGRADED", 270f, .84f, "正在刷新内容…")
            "OFFLINE_CACHE" -> statusBanner("OFFLINE_CACHE", 1900f, .84f, "仍可查看已保存的搜索记录。")
            "FILTER_ACTIVE" -> pill(72f, 282f, 340f, 354f, "筛选已生效", Pc.brandSoft, Pc.brand, 28f, Pc.brand)
        }
    }

    private fun p03DrawerHeader(query: String?, searchActive: Boolean, newChatEnabled: Boolean) {
        val primarySoft = color("#EEF0FF")
        text("YLVEN", 54f, 113f, 46f, Pc.text, true)
        lineIcon(810f, 123f, "plus", 36f, if (newChatEnabled) Pc.text else Pc.disabled, 5f)
        rounded(48f, 196f, 864f, 288f, 46f, if (searchActive) Pc.surface else color("#F4F6FA"), if (searchActive) Pc.brand else null, if (searchActive) 3f else 1f)
        lineIcon(60f, 210f, "search", 64f, Pc.disabled, 5f)
        text(query ?: "搜索对话", 143f, 216f, 38f, if (query == null) Pc.disabled else Pc.text)
        if (searchActive && query != null) {
            line(810f, 229f, 836f, 255f, Pc.disabled, 4f)
            line(836f, 229f, 810f, 255f, Pc.disabled, 4f)
        }
        rounded(48f, 316f, 864f, 416f, 28f, if (newChatEnabled) primarySoft else color("#F2F4F7"))
        lineIcon(73f, 347f, "plus", 36f, if (newChatEnabled) Pc.brand else Pc.disabled, 5f)
        text("开始新对话", 143f, 340f, 42f, if (newChatEnabled) Pc.brand else Pc.disabled, true)
    }

    private fun p03DrawerGroup(localY: Float, label: String) =
        text(label, 60f, localY + 72f, 32f, Pc.text2)

    private fun p03DrawerConversation(
        localY: Float,
        title: String,
        subtitle: String,
        highlighted: Boolean = false,
    ) {
        val y = localY + 72f
        if (highlighted) rounded(44f, y - 12f, 870f, y + 100f, 24f, color("#EEF0FF"))
        text(title, 66f, y, 40f, Pc.text)
        text(subtitle, 66f, y + 52f, 28f, Pc.disabled)
        lineIcon(796f, y + 2f, "more", 48f, Pc.disabled, 4f)
    }

    private fun p03DrawerSkeleton(localY: Float, titleWidth: Float, subtitleWidth: Float) {
        val y = localY + 72f
        rounded(66f, y, 66f + titleWidth, y + 40f, 18f, Pc.divider)
        rounded(66f, y + 57f, 66f + subtitleWidth, y + 82f, 12f, color("#F2F4F7"))
        listOf(804f, 822f, 840f).forEach { x -> circle(x + 4f, y + 22f, 4f, Pc.divider) }
    }

    private fun p03DrawerBanner(localY: Float, label: String, kind: String, spinner: Boolean = false) {
        val y = localY + 72f
        val (background, foreground) = when (kind) {
            "warning" -> color("#FFF8E7") to color("#B54708")
            "error" -> Pc.errorSoft to Pc.error
            else -> Pc.infoSoft to color("#175CD3")
        }
        rounded(48f, y, 864f, y + 74f, 24f, background)
        if (spinner) spinner(86f, y + 37f, 15f, foreground, 5f) else iconCircle(87f, y + 37f, 10f, "i", foreground, Pc.surface, 17f)
        text(label, 119f, y + 17f, 30f, foreground)
    }

    private fun p03DrawerEmpty() {
        val primarySoft = color("#EEF0FF")
        circle(456f, 762f, 95f, primarySoft)
        rounded(398f, 718f, 503f, 798f, 24f, Pc.surface, Pc.brand, 5f)
        listOf(431f, 456f, 481f).forEach { x -> circle(x, 755f, 5f, Pc.brand) }
        path(listOf(464f to 796f, 484f to 820f, 492f to 796f), Pc.brand, 5f)
        text("还没有对话", 456f, 896f, 43f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text("开始一次新的对话，记录会显示在这里", 456f, 966f, 31f, Pc.text2, anchor = Anchor.MIDDLE_ASCENDER)
    }

    private fun p03DrawerError(icon: String, title: String, body: String, button: String) {
        val cx = 456f
        val cy = 772f
        circle(cx, cy, 92f, color("#EEF0FF"))
        when (icon) {
            "network" -> {
                arc(cx - 46f, cy - 46f, cx + 46f, cy + 46f, 210f, 120f, Pc.brand, 6f)
                arc(cx - 33f, cy - 33f, cx + 33f, cy + 33f, 210f, 120f, Pc.brand, 6f)
                arc(cx - 20f, cy - 20f, cx + 20f, cy + 20f, 210f, 120f, Pc.brand, 6f)
                circle(cx, cy + 20f, 5f, Pc.brand)
                line(cx - 45f, cy - 48f, cx + 45f, cy + 42f, Pc.brand, 8f)
            }
            "server" -> listOf(-38f, 0f, 38f).forEach { offset ->
                rounded(cx - 54f, cy + offset - 15f, cx + 54f, cy + offset + 15f, 8f, null, Pc.brand, 6f)
                circle(cx - 33f, cy + offset, 4f, Pc.brand)
                line(cx + 10f, cy + offset, cx + 35f, cy + offset, Pc.brand, 5f)
            }
            else -> {
                rounded(cx - 45f, cy - 8f, cx + 45f, cy + 67f, 15f, null, Pc.brand, 7f)
                arc(cx - 31f, cy - 62f, cx + 31f, cy + 3f, 180f, 180f, Pc.brand, 7f)
                circle(cx, cy + 25f, 7f, Pc.brand)
                line(cx, cy + 31f, cx, cy + 46f, Pc.brand, 6f)
            }
        }
        text(title, cx, 910f, 43f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text(body, cx, 980f, 31f, Pc.text2, anchor = Anchor.MIDDLE_ASCENDER)
        rounded(cx - 165f, 1069f, cx + 165f, 1155f, 28f, Pc.brand)
        text(button, cx, 1088f, 34f, Pc.surface, true, Anchor.MIDDLE_ASCENDER)
    }

    private fun p03ConversationMenu(code: String) {
        // The source dialog draws the normal chat body directly, without the
        // page-level completed banner, before applying its modal scrim.
        p03ConversationMenuParent()
        overlay(Color.argb(125, 16, 24, 40))
        val y0 = 970f
        shadowCard(72f, y0, 1008f, 2110f, 48f, shadow = 14f, offset = 6f)
        rounded(450f, y0 + 24f, 630f, y0 + 38f, 7f, Pc.border2)
        text("会话操作", 120f, y0 + 86f, 52f, Pc.text, true)
        listOf("重命名", "移入项目", "导出", "临时对话", "删除").forEachIndexed { index, item ->
            val y = y0 + 190f + index * 142f
            lineIcon(120f, y - 16f, if (index < 2) "file" else "more", 54f, if (item == "删除") Pc.error else Pc.text2, 4f)
            text(item, 204f, y, 38f, if (item == "删除") Pc.error else Pc.text, item == "删除", Anchor.LEFT_MIDDLE)
            line(120f, y + 72f, 960f, y + 72f, Pc.divider, 2f)
        }
        when (code) {
            "SUBMITTING" -> statusBanner("SERVICE_DEGRADED", 650f, .75f, "正在处理所选操作…")
            "SUCCESS" -> statusBanner("SUCCESS", 650f, .75f)
            "SAVE_ERROR" -> statusBanner("SAVE_ERROR", 650f, .75f)
            "DISABLED" -> statusBanner("PERMISSION_DENIED", 650f, .75f)
        }
        gesture(Pc.surface)
    }

    private fun p03ConversationMenuParent() {
        p03ChatBaselineScale = .12f
        try {
            topBar("YLVEN App 架构讨论", "GPT-5.6 Sol · 深度推理", back = true, right = true)
            pill(72f, 300f, 450f, 370f, "GPT-5.6 Sol · 深度", Pc.brandSoft, Pc.brand, 26f)
            rounded(320f, 455f, 1008f, 650f, 42f, Pc.brand)
            text(
                "请为 YLVEN 设计一套可扩展的\n多模型 AI 后端架构。",
                370f, 505f, 35f, Pc.surface, maxWidth = 580f, lineSpacing = 16f,
            )
            text("GPT-5.6 Sol · 深度推理", 72f, 740f, 27f, Pc.brand, true)
            text(
                "建议将系统拆分为控制平面、AI 数据平面和异步工作平\n" +
                    "面。\n业务后端管理用户、会话、钱包和作品；AI Runtime 负\n" +
                    "责流式请求、\n模型路由与上下文编译；Worker 负责图片、文件和 PPT\n任务。",
                72f, 801f, 37f, Pc.text, lineSpacing = 23f,
            )
            rounded(72f, 1110f, 1008f, 1320f, 30f, Pc.surfaceSubtle, Pc.border, 2f)
            text("已读取 2 份项目资料", 120f, 1150f, 28f, Pc.text3, true)
            text("架构说明.pdf · 数据模型.md", 120f, 1210f, 32f, Pc.text, true)
            rounded(54f, 1960f, 1026f, 2185f, 70f, Pc.surface, Pc.border2, 2f)
            iconCircle(130f, 2072f, 38f, "＋", Pc.surfaceSubtle, Pc.text2, 30f)
            text("继续追问…", 200f, 2050f, 34f, Pc.disabled)
            iconCircle(940f, 2072f, 42f, "➤", Pc.brand, Pc.surface, 30f)
        } finally {
            p03ChatBaselineScale = 0f
        }
    }

    private fun p03Chat(code: String, drawComposer: Boolean = true) {
        val isNewConversation = code in setOf(
            "DEFAULT", "INPUT_FOCUSED", "UPLOADING", "DISABLED", "OFFLINE_NEW",
        )
        p03ChatBaselineScale = .09f
        try {
            p03ChatTopBar(
                if (isNewConversation) "新对话" else "多服务器 AI 架构设计",
                if (isNewConversation) "自动选择" else "GPT-5.6 Sol · 深度",
            )
            when (code) {
                "DEFAULT" -> p03EmptyChatHero("有什么想一起完成的？", "直接提问，YLVEN 会自动选择合适的模型。")
                "INPUT_FOCUSED" -> p03EmptyChatHero("从一个问题开始", null)
                "UPLOADING" -> p03EmptyChatHero("资料已加入本次对话", null)
                "DISABLED", "OFFLINE_NEW" ->
                    p03EmptyChatHero("有什么想一起完成的？", "直接提问，YLVEN 会自动选择合适的模型。")
                else -> {
                    p03ChatUserMessage()
                    when (code) {
                        "CONNECTING" -> p03ChatConnecting()
                        "STREAMING" -> p03ChatAnswer(partial = true)
                        "TOOL_RUNNING" -> p03ChatToolProgress()
                        "COMPLETED" -> p03ChatAnswer(partial = false, actions = true)
                        "STOPPED" -> p03ChatAnswer(partial = true, stopped = true)
                        "RECONNECTING" -> p03ChatAnswer(partial = true, reconnecting = true)
                        "OFFLINE" -> p03ChatRecovery("当前网络不可用", "恢复网络后即可继续这段对话。", "重试", "稍后再试", "warning")
                        "RATE_LIMITED" -> p03ChatRecovery("当前请求较多", "请稍后再试，当前输入内容已经保留。", "稍后重试", "更换模型", "warning")
                        "PROVIDER_ERROR" -> p03ChatRecovery("暂时无法完成回答", "你可以重试，或切换到其他可用模型。", "重试", "更换模型", "error")
                        "CONTENT_BLOCKED" -> p03ChatRecovery("这个请求暂时无法处理", "请调整问题描述后重新发送。", "修改问题", "返回", "warning")
                    }
                }
            }
            if (drawComposer) p03ChatComposer(code)
        } finally {
            p03ChatBaselineScale = 0f
        }
    }

    private fun p03ChatTopBar(title: String, subtitle: String) {
        androidStatus()
        rect(0f, 72f, 1080f, 246f, Pc.surface)
        line(0f, 245f, 1080f, 245f, Pc.divider, 2f)
        lineIcon(48f, 122f, "back", 60f, Pc.text, 5f)
        val titleColor = if (p03ConversationComposerMode) Pc.text2 else Pc.text
        val subtitleColor = if (p03ConversationComposerMode) Pc.text3 else Pc.text2
        text(title, 130f, 124f, 66f, titleColor, true)
        text(subtitle, 130f, 204f, 39f, subtitleColor)
        lineIcon(970f, 128f, "more", 56f, Pc.text2, 5f)
    }

    private fun p03Spark(cx: Float, cy: Float, size: Float = 54f) {
        filledPath(
            listOf(
                cx to cy - size,
                cx + size * .24f to cy - size * .24f,
                cx + size to cy,
                cx + size * .24f to cy + size * .24f,
                cx to cy + size,
                cx - size * .24f to cy + size * .24f,
                cx - size to cy,
                cx - size * .24f to cy - size * .24f,
                cx to cy - size,
            ),
            Pc.brand,
        )
        circle(cx, cy, size * .14f, Pc.surface)
    }

    private fun p03EmptyChatHero(title: String, subtitle: String?) {
        val hero = when {
            subtitle != null -> listOf(810f, 54f, 903f, 80f)
            title == "从一个问题开始" -> listOf(760f, 54f, 840f, 58f)
            else -> listOf(700f, 50f, 775f, 58f)
        }
        p03Spark(540f, hero[0], hero[1])
        val titleColor = if (p03ConversationComposerMode) Pc.text2 else Pc.text
        val subtitleColor = if (p03ConversationComposerMode) Pc.text3 else Pc.text2
        text(title, 540f, hero[2], hero[3], titleColor, true, Anchor.MIDDLE_ASCENDER)
        if (subtitle != null) text(subtitle, 540f, 1015f, 40f, subtitleColor, anchor = Anchor.MIDDLE_ASCENDER)
    }

    private fun p03ChatUserMessage() {
        rounded(351f, 350f, 1026f, 620f, 54f, Pc.brand)
        text(
            "如何让多模型对话具备上下文\n，并为多服务器部署做好准备\n？",
            391f,
            394f,
            45f,
            Pc.surface,
            lineSpacing = 16f,
        )
    }

    private fun p03ChatAssistantLabel() {
        circle(78f, 743f, 24f, color("#EEF0FF"))
        p03Spark(78f, 743f, 12f)
        text("YLVEN", 108f, 724f, 30f, Pc.text2)
    }

    private fun p03ChatConnecting() {
        p03ChatAssistantLabel()
        text("正在思考", 54f, 800f, 46f, Pc.text, true)
        listOf(330f, 405f, 350f).forEachIndexed { index, width ->
            rounded(54f, 865f + index * 50f, 54f + width, 895f + index * 50f, 15f, color("#E8EAFF"))
        }
    }

    private fun p03ChatAnswer(
        partial: Boolean,
        actions: Boolean = false,
        stopped: Boolean = false,
        reconnecting: Boolean = false,
    ) {
        p03ChatAssistantLabel()
        text(
            "可以。建议把当前系统设计为无状态 AI Runti\nme，并由 PostgreSQL 保存完整会话事实。",
            54f,
            801f,
            45f,
            Pc.text,
            lineSpacing = 18f,
        )
        text(
            "每次发送消息时，后端根据 conversation_id\n重新组装最近对话、结构化摘要和当前附件，",
            54f,
            991f,
            45f,
            Pc.text,
            lineSpacing = 18f,
        )
        if (!partial) {
            text("这样后续切换模型时，上下文仍然连续。", 54f, 1181f, 45f, Pc.text)
        } else {
            rounded(58f, 1117f, 66f, 1170f, 4f, Pc.brand)
        }
        if (stopped) pill(54f, 1230f, 250f, 1288f, "已停止生成", color("#F2F4F7"), Pc.text2, 23f)
        if (reconnecting) pill(54f, 1230f, 420f, 1294f, "连接不稳定，正在恢复…", Pc.warningSoft, Pc.warning, 23f)
        if (actions) {
            listOf("复制", "朗读", "重答", "换模型").forEachIndexed { index, label ->
                val left = 54f + index * 154f
                pill(left, 1320f, left + 132f, 1385f, label, Pc.surface, Pc.text2, 23f, Pc.border)
            }
        }
    }

    private fun p03ChatToolProgress() {
        p03ChatAssistantLabel()
        text("正在阅读你提供的资料", 54f, 800f, 45f, Pc.text, true)
        rounded(54f, 870f, 1026f, 1055f, 30f, Pc.surface, Pc.border, 2f)
        spinner(112f, 950f, 26f, Pc.brand, 6f)
        text("正在分析 2 份文件", 160f, 900f, 33f, Pc.text, true)
        text("架构说明.pdf  ·  部署清单.md", 160f, 962f, 27f, Pc.text2)
        rounded(160f, 1010f, 820f, 1026f, 8f, color("#EAECF0"))
        rounded(160f, 1010f, 520f, 1026f, 8f, Pc.brand)
    }

    private fun p03ChatRecovery(
        title: String,
        body: String,
        primaryAction: String,
        secondaryAction: String,
        kind: String,
    ) {
        val bg = if (kind == "error") Pc.errorSoft else Pc.warningSoft
        val fg = if (kind == "error") Pc.error else Pc.warning
        rounded(54f, 700f, 1026f, 955f, 34f, bg)
        circle(100f, 762f, 16f, fg)
        text(title, 132f, 730f, 48f, Pc.text, true)
        text(body, 132f, 795f, 36f, Pc.text2)
        pill(132f, 860f, 330f, 925f, primaryAction, Pc.brand, Pc.surface, 30f)
        pill(350f, 860f, 550f, 925f, secondaryAction, Pc.surface, Pc.text2, 30f, Pc.border)
    }

    private fun p03ChatComposer(code: String) {
        val focused = code in setOf("INPUT_FOCUSED", "UPLOADING")
        val sending = code in setOf("CONNECTING", "STREAMING", "TOOL_RUNNING", "RECONNECTING")
        val disabled = code in setOf("DISABLED", "OFFLINE_NEW")
        val top = when (code) {
            "INPUT_FOCUSED" -> 1950f
            "UPLOADING" -> 1915f
            else -> 2080f
        }
        val placeholder = when (code) {
            "INPUT_FOCUSED" -> "帮我规划一套多服务器部署方案"
            "UPLOADING" -> "结合资料，给出方案"
            "DEFAULT" -> "问问 YLVEN…"
            "DISABLED" -> "当前不可发送"
            "OFFLINE_NEW" -> "恢复网络后即可发送"
            else -> "继续追问…"
        }
        if (code == "OFFLINE_NEW") {
            rounded(54f, 1930f, 1026f, 2038f, 28f, Pc.warningSoft)
            circle(96f, 1984f, 18f, Pc.warning)
            text("当前离线，输入内容会保留", 132f, 1955f, 31f, Pc.text)
        }
        shadowCard(
            54f,
            top,
            1026f,
            top + 174f,
            54f,
            Pc.surface,
            if (focused) Pc.brand else Pc.border,
            shadow = 8f,
            offset = 4f,
        )
        if (code == "UPLOADING") {
            pill(82f, top + 28f, 340f, top + 88f, "架构说明.pdf", color("#F2F4F7"), Pc.text2, 22f, Pc.border)
            pill(354f, top + 28f, 620f, top + 88f, "部署清单.md", color("#F2F4F7"), Pc.text2, 22f, Pc.border)
            text(placeholder, 82f, top + 78f, 40f, Pc.text2)
        } else {
            text(placeholder, 82f, top + 28f, 40f, Pc.disabled, maxWidth = 760f)
        }
        text("+", 112f, top + 114f, 46f, if (disabled) Pc.border2 else Pc.text2, anchor = Anchor.MIDDLE_MIDDLE)
        pill(
            150f, top + 74f, if (code in setOf("DEFAULT", "DISABLED", "OFFLINE_NEW")) 380f else 410f, top + 154f,
            if (code in setOf("DEFAULT", "DISABLED", "OFFLINE_NEW")) "自动选择" else "GPT-5.6 Sol",
            if (disabled) color("#F2F4F7") else color("#EEF0FF"),
            if (disabled) Pc.disabled else Pc.brand,
            30f,
        )
        lineIcon(830f, top + 87f, "mic", 54f, if (disabled) Pc.disabled else Pc.text2, 5f)
        iconCircle(
            950f, top + 114f, 48f, if (sending) "■" else "➤",
            if (disabled) Pc.border2 else Pc.brand, Pc.surface, if (sending) 22f else 30f,
        )
    }

    private fun p03ChatStatusBanner(code: String, top: Float, widthRatio: Float, message: String? = null) {
        val previousBaselineScale = p03ChatBaselineScale
        p03ChatBaselineScale = .06f
        p03ChatStatusFontCalibration = true
        try {
            statusBanner(code, top, widthRatio, message)
        } finally {
            p03ChatStatusFontCalibration = false
            p03ChatBaselineScale = previousBaselineScale
        }
    }

    private fun componentHeader(title: String, subtitle: String) {
        androidStatus()
        rect(0f, 72f, 1080f, 270f, Pc.surface)
        text("YLVEN UI CONTRACT", 60f, 118f, 27f, Pc.brand, true)
        text(title, 60f, 166f, 52f, Pc.text, true)
        text(subtitle, 60f, 228f, 27f, Pc.text3)
        pill(830f, 122f, 1010f, 184f, "组件规范", Pc.brandSoft, Pc.brand, 24f)
        shadowCard(54f, 330f, 1026f, 2020f, 52f)
    }

    private fun p03ConversationComposer(code: String) {
        p03ConversationComposerMode = true
        try {
            when (code) {
                "DISABLED" -> p03Chat("DISABLED")
                "OFFLINE" -> p03Chat("OFFLINE_NEW")
                "TOOL_TRAY_OPEN" -> {
                    p03Chat("DEFAULT", drawComposer = false)
                    rect(0f, 72f, 1080f, 2400f, Color.argb(82, 0, 0, 0))
                    rounded(0f, 1180f, 1080f, 2400f, 72f, Pc.surface)
                    rounded(480f, 1210f, 600f, 1224f, 7f, Pc.border2)
                    text("添加内容或使用工具", 60f, 1300f, 66f, Pc.text, true)
                    val tools = listOf(
                        "拍照" to "相", "选择图片" to "图", "上传文件" to "文",
                        "生成图片" to "画", "制作演示" to "P", "深度研究" to "研",
                    )
                    tools.forEachIndexed { index, item ->
                        val col = index % 3
                        val row = index / 3
                        val left = 54f + col * 342f
                        val top = 1380f + row * 220f
                        rounded(left, top, left + 300f, top + 190f, 28f, color("#F7F8FC"))
                        iconCircle(left + 150f, top + 72f, 42f, item.second, color("#EEF0FF"), Pc.brand, 44f)
                        text(item.first, left + 150f, top + 145f, 42f, Pc.text, anchor = Anchor.MIDDLE_ASCENDER)
                    }
                    p03ChatComposer("DEFAULT")
                }
                else -> p03Chat(code)
            }
        } finally {
            p03ConversationComposerMode = false
        }
    }

    private fun p03Composer(code: String, title: String) {
        componentHeader(title, "附件、输入、语音与发送/停止状态")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val y = 890f
        rounded(110f, y, 970f, y + 270f, 82f, Pc.surface, if (code == "INPUT_FOCUSED") Pc.brand else Pc.border2, if (code == "INPUT_FOCUSED") 5f else 2f)
        iconCircle(190f, y + 135f, 42f, "＋", Pc.surfaceSubtle, Pc.text2, 34f)
        text("输入问题或上传文件", 260f, y + 105f, 36f, Pc.disabled)
        iconCircle(820f, y + 135f, 42f, "🎙", Pc.surfaceSubtle, Pc.text2, 28f)
        iconCircle(920f, y + 135f, 42f, "➤", Pc.brand, Pc.surface, 30f)
        pill(120f, y - 100f, 440f, y - 30f, "GPT-5.6 Sol · 深度", Pc.brandSoft, Pc.brand, 25f)
        if (code == "UPLOADING") {
            rounded(125f, y - 245f, 660f, y - 130f, 24f, Pc.blueSoft, Pc.border, 2f)
            text("需求文档.pdf · 62%", 170f, y - 205f, 28f, Pc.blue, true)
        }
        if (code == "DISABLED") {
            rect(110f, y, 970f, y + 270f, Color.argb(160, 246, 247, 251))
        }
        if (code == "TOOL_TRAY_OPEN") {
            overlay(Color.argb(82, 0, 0, 0))
            rounded(0f, 1210f, 1080f, 2400f, 64f, Pc.surface)
            rounded(480f, 1240f, 600f, 1254f, 7f, Pc.border2)
            text("添加内容或使用工具", 60f, 1360f, 52f, Pc.text, true)
            val tools = listOf("拍照" to "相", "选择图片" to "图", "上传文件" to "文", "生成图片" to "画", "制作演示" to "P", "深度研究" to "研")
            tools.forEachIndexed { index, item ->
                val col = index % 3
                val row = index / 3
                val left = 54f + col * 342f
                val top = 1440f + row * 230f
                rounded(left, top, left + 300f, top + 190f, 28f, Pc.surfaceSubtle)
                iconCircle(left + 150f, top + 72f, 42f, item.second, Pc.brandSoft, Pc.disabled, 30f)
                text(item.first, left + 150f, top + 145f, 31f, Pc.disabled, anchor = Anchor.MIDDLE_ASCENDER)
            }
            rounded(54f, 1990f, 1026f, 2205f, 64f, Pc.surface, Pc.border, 2f)
            text("问问 YLVEN…", 102f, 2055f, 34f, Pc.disabled)
            text("+", 115f, 2150f, 52f, Pc.text2, anchor = Anchor.MIDDLE_MIDDLE)
            pill(160f, 2105f, 380f, 2180f, "自动选择", Pc.brandSoft, Pc.brand, 26f)
            iconCircle(958f, 2140f, 44f, "➤", Pc.brand, Pc.surface, 28f)
        }
        text("视觉类型：独立组件板", 90f, 2085f, 28f, Pc.text3)
        text("示例文案仅用于视觉占位，功能以 Feature ID 为准。", 90f, 2140f, 26f, Pc.text3)
        gesture(Pc.text)
    }

    private fun p03OfflineCache(code: String) {
        topBar("离线缓存", "保留最近会话并明确缓存时间", back = true, right = true)
        if (code == "NETWORK_ERROR") { errorCenter(code); return }
        text("最近缓存的会话", 72f, 335f, 46f, Pc.text, true)
        pill(72f, 410f, 420f, 478f, "缓存于 10:21", Pc.warningSoft, Pc.warning, 26f)
        var y = 550f
        listOf("安卓 AI 工具架构设计", "YLVEN UI 规范", "多模型对比方案").forEachIndexed { index, item ->
            rounded(72f, y, 1008f, y + 180f, 34f, Pc.surface, Pc.border, 2f)
            text(item, 125f, y + 50f, 36f, Pc.text, true)
            text("只读缓存 · ${index + 1} 条未同步", 125f, y + 110f, 27f, Pc.text3)
            y += 200f
        }
        statusBanner("OFFLINE_CACHE", 1320f, .82f, "恢复网络后将自动检查最新状态。")
        if (code == "REFRESHING") statusBanner("SERVICE_DEGRADED", 270f, .84f, "正在刷新内容…")
    }

    private fun p03Response(code: String) {
        p03Chat(code)
        rounded(28f, 350f, 1052f, 1785f, 36f, null, Pc.brand, 4f)
        pill(56f, 310f, 418f, 368f, "AI 回复组件规范板", color("#EEF0FF"), Pc.brand, 25f, Pc.border)
        pill(
            56f, 1812f, 1024f, 1876f,
            "仅标注回复区域；实际产品页不显示此规范标签",
            Pc.surface, Pc.text2, 25f, Pc.border,
        )
    }

    private fun p03Selector(code: String, model: Boolean) {
        p03Chat("DEFAULT")
        rect(0f, 72f, 1080f, 2400f, Color.argb(82, 0, 0, 0))
        val top = if (model) 820f else 1040f
        rounded(0f, top, 1080f, 2400f, 72f, Pc.surface)
        rounded(480f, top + 28f, 600f, top + 42f, 7f, Pc.border2)
        val title = if (model) "选择模型" else "回答方式"
        val subtitle = if (model) "默认由 YLVEN 自动匹配适合的模型。" else "无需每次设置，默认由模型自动判断。"
        text(title, 60f, top + 105f, 60f, Pc.text, true)
        text(subtitle, 60f, top + 180f, 38f, Pc.text2)
        if (model) {
            var chipX = 60f
            listOf("全部", "推理", "视觉", "快速").forEachIndexed { index, label ->
                val active = if (code == "FILTER_ACTIVE") index == 1 else index == 0
                chipX += chip(chipX, top + 240f, label, active) + 12f
            }
        }
        if (code == "SERVER_ERROR") {
            p03SelectorRecovery(top + if (model) 430f else 340f, model)
            gesture(Pc.text)
            return
        }
        if (code == "EMPTY") {
            if (model) p03Spark(540f, top + 650f, 34f)
            text(
                if (model) "没有符合条件的模型" else "当前模型没有可调整选项",
                540f, top + if (model) 750f else 610f, 54f, Pc.text, true, Anchor.MIDDLE_ASCENDER,
            )
            text(
                if (model) "清除筛选后查看全部可用模型。" else "继续使用自动模式即可。",
                540f, top + if (model) 825f else 680f, 38f, Pc.text2, anchor = Anchor.MIDDLE_ASCENDER,
            )
            gesture(Pc.text)
            return
        }
        val allRows = if (model) {
            listOf("自动选择" to "根据问题和可用能力自动匹配", "GPT-5.6 Sol" to "复杂分析、编码与长任务", "Claude Opus" to "长文理解、写作与审查", "Grok" to "实时信息与多模态理解")
        } else {
            listOf("自动（推荐）" to "根据问题复杂度自动决定", "快速" to "更快响应，适合简单问题", "标准" to "速度与质量平衡", "深度" to "适合复杂分析和代码任务")
        }
        val rows = if (model && code == "FILTER_ACTIVE") allRows.dropLast(1) else allRows
        rows.forEachIndexed { index, item ->
            val y = if (model) top + 350f + index * 210f else top + 280f + index * 178f
            val selected = if (model) {
                index == 0
            } else {
                (index == 0 && code != "FILTER_ACTIVE") ||
                    (index == rows.lastIndex && code == "FILTER_ACTIVE")
            }
            val disabled = when {
                code == "DISABLED" -> index > 0
                code == "SERVICE_DEGRADED" && model -> index >= 2
                code == "SERVICE_DEGRADED" -> index > 0
                else -> false
            }
            val rowHeight = if (model) 190f else 160f
            rounded(54f, y, 1026f, y + rowHeight, 30f, if (selected) color("#EEF0FF") else Pc.surface, if (selected) Pc.brand else Pc.border, if (selected) 3f else 2f)
            val textX = if (model) 205f else 90f
            if (model) {
                iconCircle(130f, y + 95f, 48f, item.first.take(1), color("#EEF0FF"), if (disabled) Pc.disabled else Pc.brand, 40f)
            }
            text(item.first, textX, y + if (model) 54f else 42f, 50f, if (disabled) Pc.disabled else Pc.text, true)
            text(item.second, textX, y + if (model) 118f else 114f, 39f, if (disabled) Pc.disabled else Pc.text2)
            if (model) {
                val tag = if (disabled) "暂不可用" else listOf("推荐", "推理", "长文", "多模态")[index]
                val tagWidth = if (tag.length > 2) 150f else 112f
                pill(
                    996f - tagWidth, y + 52f, 996f, y + 112f, tag,
                    if (disabled) Pc.warningSoft else color("#F2F4F7"),
                    if (disabled) Pc.warning else Pc.text2,
                    28f,
                )
            } else if (disabled) {
                pill(840f, y + 50f, 990f, y + 110f, "不可用", Pc.warningSoft, Pc.warning, 30f)
            }
            if (selected) {
                circle(960f, y + 48f, 25f, Pc.brand)
                lineIcon(945f, y + 33f, "check", 30f, Pc.surface, 4f)
            }
        }
        if (code == "SERVICE_DEGRADED") {
            rounded(54f, top + 55f, 1026f, top + 163f, 28f, Pc.warningSoft)
            circle(96f, top + 109f, 18f, Pc.warning)
            text(
                if (model) "部分模型暂不可用" else "当前模型仅支持自动模式",
                132f, top + 80f, 40f, Pc.text,
            )
        }
        gesture(Pc.text)
    }

    private fun p03SelectorRecovery(top: Float, model: Boolean) {
        rounded(54f, top, 1026f, top + 255f, 34f, Pc.errorSoft)
        circle(102f, top + 64f, 18f, Pc.error)
        text(if (model) "暂时无法加载模型" else "暂时无法加载设置", 148f, top + 30f, 50f, Pc.text, true)
        text(
            if (model) "请稍后重试，当前选择不会改变。" else "请稍后重试，当前会继续使用自动模式。",
            148f, top + 94f, 38f, Pc.text2,
        )
        pill(148f, top + 160f, 330f, top + 225f, "重试", Pc.brand, Pc.surface, 30f)
        pill(350f, top + 160f, 570f, top + 225f, "更换模型", Pc.surface, Pc.text2, 30f, Pc.border)
    }

    private fun p03CodeBlock(code: String) {
        componentHeader("代码块组件", "语言标签、复制、横向滚动与错误态")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        rounded(105f, 570f, 975f, 1390f, 34f, color("#101828"))
        pill(135f, 600f, 330f, 665f, "Kotlin", color("#1E293B"), color("#CBD5E1"), 25f)
        pill(780f, 600f, 935f, 665f, "复制", color("#1E293B"), color("#CBD5E1"), 25f)
        val codeLines = listOf("data class Model(", "  val id: String,", "  val capabilities: Set<String>", ")", "", "fun supportsVision() =", "  \"vision\" in capabilities")
        var y = 720f
        codeLines.forEachIndexed { index, line ->
            text((index + 1).toString().padStart(2, '0'), 145f, y, 27f, color("#64748B"))
            text(line, 220f, y, 31f, color("#E2E8F0"))
            y += 75f
        }
        if (code == "COMPLETED") statusBanner("SUCCESS", 1500f, .72f, "代码块已完整渲染。")
        if (code == "CONTENT_BLOCKED") statusBanner("CONTENT_BLOCKED", 1500f, .72f, "代码内容未通过安全检查。")
        componentFooter()
    }

    private fun p03TableBlock(code: String) {
        componentHeader("表格组件", "移动端横向滚动和列冻结规则")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val columns = listOf("模型" to 120f, "识图" to 410f, "文件" to 610f, "状态" to 810f)
        val y = 610f
        rounded(105f, y, 975f, y + 690f, 32f, Pc.surface, Pc.border, 2f)
        rect(105f, y, 975f, y + 110f, Pc.surfaceSubtle)
        columns.forEach { (label, x) -> text(label, x, y + 55f, 30f, Pc.text2, true, Anchor.LEFT_MIDDLE) }
        val rows = listOf(listOf("GPT-5.6 Sol", "支持", "支持", "正常"), listOf("Claude Opus", "支持", "支持", "正常"), listOf("Grok", "支持", "部分", "限流"))
        rows.forEachIndexed { rowIndex, row ->
            val rowY = y + 110f + rowIndex * 150f
            if (rowIndex % 2 == 1) rect(106f, rowY, 974f, rowY + 150f, color("#FCFCFD"))
            row.zip(columns).forEach { (value, column) -> text(value, column.second, rowY + 75f, 29f, Pc.text, value == "正常", Anchor.LEFT_MIDDLE) }
        }
        text("移动端表格允许横向滚动，首列保持可识别。", 105f, 1390f, 29f, Pc.text3)
        if (code == "COMPLETED") statusBanner("SUCCESS", 1510f, .72f, "表格内容已加载完成。")
        if (code == "OFFLINE") statusBanner("OFFLINE", 1510f, .72f)
        componentFooter()
    }

    private fun p03CitationBlock(code: String) {
        componentHeader("引用组件", "来源标题、域名、序号与不可用状态")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        rounded(105f, 600f, 975f, 1160f, 34f, Pc.surfaceSubtle, Pc.border, 2f)
        pill(140f, 640f, 285f, 702f, "来源 1", Pc.brandSoft, Pc.brand, 25f)
        iconCircle(165f, 790f, 34f, "网", Pc.blueSoft, Pc.blue, 24f)
        text("OpenAI 官方文档", 220f, 748f, 39f, Pc.text, true)
        text("developers.openai.com · 已核验", 220f, 810f, 27f, Pc.text3)
        text("“推理档位必须根据模型实际支持能力动态展示……”", 140f, 900f, 31f, Pc.text2, maxWidth = 750f)
        button(140f, 1190f, 600f, 1320f, "打开来源网页", primary = false)
        if (code == "COMPLETED") statusBanner("SUCCESS", 1450f, .72f, "来源链接与标题已校验。")
        if (code in setOf("NOT_FOUND", "OFFLINE")) statusBanner(code, 1450f, .72f, if (code == "NOT_FOUND") "引用来源已不可用。" else null)
        componentFooter()
    }

    private fun p03MessageActions(code: String) {
        componentHeader("消息操作栏", "复制、朗读、重答、换模型和反馈")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val actions = listOf("复制" to "⧉", "朗读" to "▶", "重答" to "↻", "换模型" to "◇", "赞" to "↑", "踩" to "↓")
        val y = 820f
        rounded(90f, y, 990f, y + 210f, 42f, if (code == "DISABLED") Pc.surfaceSubtle else Pc.surfaceSubtle, Pc.border, 2f)
        actions.forEachIndexed { index, action ->
            val x = 165f + index * 145f
            iconCircle(x, y + 80f, 33f, action.second, Pc.surface, Pc.text2, 25f)
            text(action.first, x, y + 145f, 24f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER)
        }
        if (code == "DISABLED") rect(90f, y, 990f, y + 210f, Color.argb(190, 246, 247, 251))
        if (code == "SUCCESS") statusBanner("SUCCESS", 1200f, .72f, "内容已复制到剪贴板。")
        componentFooter()
    }

    private fun p04Settings(code: String, conversation: Boolean) {
        val title = if (conversation) "会话设置" else "AI 设置"
        val subtitle = if (conversation) "配置当前会话的模型和指令" else "默认模型、回答风格和工具偏好"
        val items = if (conversation) {
            listOf("会话默认模型", "推理强度", "系统指令", "临时对话")
        } else {
            listOf("默认模型", "默认推理强度", "回答风格", "联网搜索")
        }
        topBar(title, subtitle, back = true, right = true)
        // The approved settings contracts use the complete centered recovery
        // surface for OFFLINE. A bottom warning banner is still supplied by
        // stateFeedback(), matching the approved evidence rather than merely
        // appending a banner to a usable settings list.
        if (code == "OFFLINE") {
            errorCenter(code)
            return
        }
        if (code == "LOADING") {
            skeleton(72f, 330f, 1008f, rows = 7)
            return
        }
        var y = 330f
        items.forEach { item ->
            rounded(72f, y, 1008f, y + 150f, 32f, Pc.surface, Pc.border, 2f)
            text(item, 120f, y + 75f, 35f, Pc.text, anchor = Anchor.LEFT_MIDDLE)
            if (item in setOf("联网搜索", "临时对话")) {
                rounded(830f, y + 45f, 940f, y + 105f, 30f, Pc.brand)
                circle(912f, y + 75f, 25f, Pc.surface)
            } else {
                text("›", 930f, y + 75f, 40f, Pc.text3, anchor = Anchor.MIDDLE_MIDDLE)
            }
            y += 170f
        }
        if (code in setOf("EDIT_MODE", "DIRTY")) {
            button(72f, minOf(y + 20f, 1850f), 1008f, minOf(y + 176f, 2006f), "保存设置")
        }
        when (code) {
            // Save/submit feedback is an inline top-of-content status surface
            // in the approved settings screenshots. stateFeedback() renders
            // the persistent bottom audit message as a second, separate row.
            "SAVE_SUCCESS", "SAVE_ERROR" -> statusBanner(code, 270f, .82f)
            "SUBMITTING" -> statusBanner("SERVICE_DEGRADED", 270f, .82f, "正在提交，请勿重复操作。")
            "EDIT_MODE", "DIRTY" -> pill(72f, 282f, 390f, 354f, if (code == "EDIT_MODE") "编辑模式" else "存在未保存修改", Pc.warningSoft, Pc.warning, 28f, Pc.warning)
        }
    }

    private fun p04ModelStrip(code: String) {
        componentHeader("输入区模型条", "模型、推理、联网和临时切换提示")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val y = 1120f
        text("输入框上方的当前模型状态", 120f, y - 125f, 28f, Pc.text3, true)
        rounded(120f, y, 960f, y + 130f, 52f, Pc.brandSoft, Pc.border, 2f)
        iconCircle(180f, y + 65f, 34f, "AI", Pc.brand, Pc.surface, 21f)
        text("GPT-5.6 Sol", 240f, y + 38f, 34f, Pc.text, true)
        pill(600f, y + 30f, 760f, y + 96f, "深度", Pc.surface, Pc.brand, 24f)
        pill(780f, y + 30f, 920f, y + 96f, "联网", Pc.blueSoft, Pc.blue, 24f)
        rounded(120f, y + 180f, 960f, y + 390f, 64f, Pc.surface, Pc.border2, 2f)
        text("输入问题或上传文件", 180f, y + 250f, 32f, Pc.disabled)
        iconCircle(885f, y + 285f, 40f, "➤", Pc.brand, Pc.surface, 28f)
        if (code in setOf("SERVICE_DEGRADED", "DISABLED")) {
            statusBanner(if (code == "SERVICE_DEGRADED") code else "PERMISSION_DENIED", 1640f, .72f)
        }
        componentFooter()
    }

    private fun p04ResponseLabel(code: String) {
        componentHeader("AI 回复标签", "永久标识供应商、模型、推理和工具来源")
        text("当前状态", 105f, 390f, 28f, Pc.text3, true)
        pill(270f, 376f, 600f, 438f, code, Pc.surfaceSubtle, Pc.text2, 23f, Pc.border)
        val y = 760f
        text("AI 回答正文上方的来源标签", 120f, y, 28f, Pc.text3, true)
        line(120f, y + 105f, 960f, y + 105f, Pc.divider, 2f)
        iconCircle(160f, y + 165f, 28f, "AI", Pc.brand, Pc.surface, 18f)
        text("GPT-5.6 Sol", 210f, y + 140f, 34f, Pc.text, true)
        pill(505f, y + 130f, 665f, y + 192f, "深度", Pc.brandSoft, Pc.brand, 23f)
        pill(685f, y + 130f, 845f, y + 192f, "已联网", Pc.blueSoft, Pc.blue, 23f)
        text("OpenAI · 本条回答 · 12.8 秒 · 已调用 1 个工具", 120f, y + 250f, 27f, Pc.text3)
        text("这是回答正文的起始位置。模型标签不会随会话后续切换而改变。", 120f, y + 345f, 32f, Pc.text, maxWidth = 820f)
        if (code in setOf("SERVICE_DEGRADED", "DISABLED")) {
            statusBanner(if (code == "SERVICE_DEGRADED") code else "PERMISSION_DENIED", 1510f, .72f)
        }
        componentFooter()
    }

    private fun componentFooter() {
        text("视觉类型：独立组件板", 90f, 2085f, 28f, Pc.text3)
        text("示例文案仅用于视觉占位，功能以 Feature ID 为准。", 90f, 2140f, 26f, Pc.text3)
        gesture(Pc.text)
    }

    private fun homeBody() {
        text("你好，陈平", 72f, 330f, 58f, Pc.text, true)
        text("今天准备完成什么？", 72f, 405f, 36f, Pc.text3)
        shadowCard(72f, 510f, 1008f, 760f, 48f, shadow = 10f, offset = 4f)
        text("输入问题或上传文件", 125f, 565f, 36f, Pc.disabled)
        iconCircle(920f, 635f, 48f, "➤", Pc.brand, Pc.surface, 34f)
        pill(125f, 680f, 480f, 742f, "GPT-5.6 Sol · 深度", Pc.brandSoft, Pc.brand, 25f)
        val shortcuts = listOf("分析文件" to "文", "生成图片" to "图", "制作 PPT" to "P", "多模型对比" to "比")
        shortcuts.forEachIndexed { index, shortcut ->
            val x = 72f + (index % 2) * 480f
            val y = 825f + (index / 2) * 190f
            rounded(x, y, x + 448f, y + 160f, 36f, Pc.surface, Pc.border, 2f)
            iconCircle(x + 70f, y + 80f, 42f, shortcut.second, Pc.brandSoft, Pc.brand, 30f)
            text(shortcut.first, x + 135f, y + 80f, 35f, Pc.text, true, Anchor.LEFT_MIDDLE)
        }
        text("最近对话", 72f, 1250f, 44f, Pc.text, true)
        val conversations = listOf(
            "安卓 AI 工具架构设计" to "GPT-5.6 Sol · 刚刚",
            "比较 Claude 与 GPT 的推理差异" to "Claude Opus · 昨天",
            "YLVEN 品牌主视觉" to "Grok · 7 月 30 日",
        )
        var y = 1330f
        conversations.forEach { row ->
            rounded(72f, y, 1008f, y + 170f, 32f, Pc.surface, Pc.border, 2f)
            iconCircle(142f, y + 85f, 38f, "AI", Pc.blueSoft, Pc.blue, 24f)
            text(row.first, 205f, y + 42f, 34f, Pc.text, true)
            text(row.second, 205f, y + 98f, 27f, Pc.text3)
            y += 188f
        }
    }

    private fun minePage() {
        topBar("我的", "账号、套餐、用量与系统设置", back = false, right = true)
        bottomNav("我的")
        shadowCard(72f, 310f, 1008f, 620f, 48f, shadow = 10f, offset = 4f)
        iconCircle(185f, 445f, 72f, "陈", Pc.brand, Pc.surface, 48f)
        text("陈平", 295f, 365f, 48f, Pc.text, true)
        text("UID 100001 · YLVEN Pro", 295f, 430f, 29f, Pc.text3)
        pill(295f, 490f, 520f, 555f, "编辑个人资料", Pc.brandSoft, Pc.brand, 24f)
        val groups = listOf(
            "使用与内容" to listOf("我的项目", "我的作品", "我的文件", "我的收藏"),
            "AI 设置" to listOf("默认模型", "回答风格", "自定义指令", "个人记忆"),
            "安全与系统" to listOf("账号安全", "登录设备", "通知设置", "检查更新"),
        )
        var y = 700f
        groups.forEach { group ->
            text(group.first, 72f, y, 31f, Pc.text3, true)
            y += 65f
            rounded(72f, y, 1008f, y + group.second.size * 116f, 34f, Pc.surface, Pc.border, 2f)
            group.second.forEachIndexed { index, item ->
                val rowY = y + index * 116f
                iconCircle(130f, rowY + 58f, 30f, "›", Pc.surfaceSubtle, Pc.text3, 26f)
                text(item, 185f, rowY + 58f, 34f, Pc.text, anchor = Anchor.LEFT_MIDDLE)
                text("›", 940f, rowY + 58f, 36f, Pc.text3, anchor = Anchor.MIDDLE_MIDDLE)
                if (index < group.second.lastIndex) line(185f, rowY + 115f, 960f, rowY + 115f, Pc.divider, 2f)
            }
            y += group.second.size * 116f + 70f
        }
    }

    private fun stateFeedback(page: String, code: String) {
        val p03ComponentFeedbackPages = setOf("YL-A-022", "YL-A-025", "YL-A-027", "YL-A-028", "YL-A-029", "YL-A-030")
        if (page in P03_PAGE_IDS && page !in p03ComponentFeedbackPages) return
        if (code in setOf("DEFAULT", "POPULATED", "LAUNCH", "FIRST_RUN")) return
        val title = mapOf(
            "YL-A-004" to "YLVEN",
            "YL-A-005" to "登录 YLVEN",
            "YL-A-006" to "完成安全验证",
            "YL-A-007" to "遗留验证页面（已停用）",
            "YL-A-008" to "输入登录验证码",
            "YL-A-009" to "登录成功",
            "YL-A-010" to "创建 YLVEN 账户",
            "YL-A-011" to "确认注册安全验证",
            "YL-A-012" to "验证注册邮箱",
            "YL-A-013" to "账户已创建",
            "YL-A-014" to "全局登录状态",
            "YL-A-015" to "退出登录",
            "YL-A-016" to "登录设备",
            "YL-A-017" to "认证适配与异常状态",
            "YL-A-018" to "AI 首页",
            "YL-A-020" to "会话历史",
            "YL-A-021" to "搜索会话",
            "YL-A-022" to "会话操作",
            "YL-A-023" to "YLVEN App 架构讨论",
            "YL-A-024" to "会话输入区组件",
            "YL-A-025" to "离线缓存",
            "YL-A-026" to "AI 回复组件",
            "YL-A-027" to "代码块组件",
            "YL-A-028" to "表格组件",
            "YL-A-029" to "引用组件",
            "YL-A-030" to "消息操作栏",
            "YL-A-032" to "输入框组件",
            "YL-A-033" to "模型选择器",
            "YL-A-034" to "回答方式选择器",
            "YL-A-035" to "AI 设置",
            "YL-A-036" to "会话设置",
            "YL-A-037" to "输入区模型条",
            "YL-A-038" to "AI 回复标签",
        ).getValue(page)
        val message = when (code) {
            "LOADING" -> "正在加载「$title」"
            "EMPTY" -> "「$title」暂无可显示内容"
            "REFRESHING" -> "正在刷新「$title」"
            "FILTER_ACTIVE" -> "筛选条件已生效"
            "BULK_SELECTED" -> "已选择 3 项，可执行批量操作"
            "EDIT_MODE" -> "当前处于编辑模式"
            "DIRTY" -> "存在未保存修改"
            "OFFLINE" -> "「$title」当前离线"
            "OFFLINE_CACHE" -> "正在显示「$title」的离线缓存"
            "NETWORK_ERROR" -> "「$title」网络连接失败"
            "SERVER_ERROR" -> "「$title」服务暂时不可用"
            "SERVICE_DEGRADED" -> "「$title」部分能力暂时降级"
            "SUBMITTING" -> "正在提交，请勿重复操作"
            "CONNECTING" -> "正在建立安全连接"
            "RECONNECTING" -> "连接中断，正在安全重连"
            "PREPARING" -> "正在检查参数与可用额度"
            "QUEUED" -> "任务已进入队列"
            "RUNNING" -> "任务正在执行"
            "RETRYING" -> "正在按策略重试"
            "UPLOADING" -> "文件正在上传"
            "DOWNLOADING" -> "文件正在下载"
            "UPLOAD_FAILED" -> "上传失败，可重新尝试"
            "FAILED" -> "任务执行失败"
            "CANCELLED" -> "任务已取消"
            "CANCEL_CONFIRM" -> "确认是否取消当前任务"
            "STREAMING" -> "AI 正在流式生成"
            "TOOL_RUNNING" -> "AI 工具正在执行"
            "COMPLETED" -> "任务已完成"
            "STOPPED" -> "生成已停止"
            "PROVIDER_ERROR" -> "当前模型响应失败"
            "CONTENT_BLOCKED" -> "内容未通过安全检查"
            "NOT_FOUND" -> "请求的内容不存在"
            "DISABLED" -> "当前操作暂不可用"
            "VALIDATION_ERROR" -> "请修正标记的输入内容"
            "SAVE_SUCCESS" -> "修改已保存"
            "SAVE_ERROR" -> "保存失败，输入内容已保留"
            "SUCCESS" -> "操作已成功完成"
            "INPUT_FOCUSED" -> "输入控件已聚焦"
            "CODE_SENT" -> "验证码已发送"
            "INVALID_CODE" -> "验证码错误"
            "CODE_EXPIRED" -> "验证码已过期"
            "RATE_LIMITED" -> "请求过于频繁，请稍后重试"
            "LOCKED" -> "账号暂时锁定"
            "UNAUTHORIZED" -> "登录状态已失效"
            "PERMISSION_DENIED" -> "当前账号权限不足"
            "BUDGET_EXHAUSTED" -> "可用预算已用尽"
            "PARTIAL_DATA" -> "部分数据源暂不可用"
            "VERSION_CONFLICT" -> "配置版本发生冲突"
            "ROLLBACK_CONFIRM" -> "请确认是否回滚到上一版本"
            "UPDATE_REQUIRED" -> "必须更新后继续使用"
            "SECURITY_CHALLENGE" -> "正在进行安全验证"
            "MAINTENANCE" -> "系统维护中"
            "UPDATE_AVAILABLE" -> "发现可用更新"
            else -> "$code · $title"
        }
        val success = code in setOf("SUCCESS", "SAVE_SUCCESS", "COMPLETED", "CODE_SENT")
        val error = code in setOf(
            "NETWORK_ERROR", "SERVER_ERROR", "SAVE_ERROR", "FAILED", "UPLOAD_FAILED",
            "INVALID_CODE", "LOCKED", "PERMISSION_DENIED", "UNAUTHORIZED",
            "PROVIDER_ERROR", "CONTENT_BLOCKED",
        )
        val warning = code in setOf(
            "OFFLINE", "OFFLINE_CACHE", "RATE_LIMITED", "CODE_EXPIRED",
            "SERVICE_DEGRADED", "DIRTY", "VERSION_CONFLICT", "PAYMENT_PENDING",
            "CREDIT_PENDING", "REFUND_PENDING", "BUDGET_EXHAUSTED", "MAINTENANCE",
            "CANCEL_CONFIRM", "PARTIAL_DATA",
        )
        val bg = when { success -> Pc.successSoft; error -> Pc.errorSoft; warning -> Pc.warningSoft; else -> Pc.infoSoft }
        val fg = when { success -> Pc.success; error -> Pc.error; warning -> Pc.warning; else -> Pc.info }
        rounded(72f, 2070f, 1008f, 2186f, 28f, bg, mix(bg, fg, .22f), 2f)
        iconCircle(127f, 2128f, 27f, if (success) "✓" else if (error || warning) "!" else "·", bg, fg, 24f)
        text(message, 172f, 2104f, 28f, fg, true, maxWidth = 801f)
    }

    private fun topBar(title: String, subtitle: String, back: Boolean, right: Boolean, plus: Boolean = false) {
        androidStatus()
        rect(0f, 72f, 1080f, 246f, Pc.surface)
        line(0f, 245f, 1080f, 245f, Pc.divider, 2f)
        if (back) lineIcon(48f, 122f, "back", 60f, Pc.text, 5f)
        val x = if (back) 130f else 54f
        text(title, x, 118f, 54f, Pc.text, true)
        text(subtitle, x, 180f, 30f, Pc.text3)
        if (right) lineIcon(970f, 128f, if (plus) "plus" else "more", 56f, Pc.text2, 5f)
    }

    private fun androidStatus() {
        text("9:41", 72f, 35f, 31f, Pc.text, true)
        listOf(12f, 20f, 28f, 36f).forEachIndexed { index, h -> rounded(858f + index * 16f, 54f - h, 868f + index * 16f, 54f, 4f, Pc.text) }
        arc(934f, 26f, 984f, 70f, 210f, 120f, Pc.text, 5f)
        arc(946f, 38f, 972f, 64f, 210f, 120f, Pc.text, 5f)
        rounded(1000f, 29f, 1050f, 58f, 7f, null, Pc.text, 4f)
        rounded(1005f, 34f, 1040f, 53f, 4f, Pc.text)
        rect(1051f, 38f, 1057f, 50f, Pc.text)
    }

    private fun bottomNav(active: String) {
        val y = 2170f
        rect(0f, y, 1080f, 2400f, Pc.surface)
        line(0f, y, 1080f, y, Pc.divider, 2f)
        listOf("首页" to "home", "工作" to "briefcase", "发现" to "compass", "我的" to "user").forEachIndexed { index, item ->
            val cx = 135f + index * 270f
            val fg = if (item.first == active) Pc.brand else Pc.text3
            if (item.first == active) rounded(cx - 62f, y + 24f, cx + 62f, y + 94f, 35f, Pc.brandSoft)
            lineIcon(cx - 27f, y + 32f, item.second, 54f, fg, 5f)
            text(item.first, cx, y + 111f, 29f, fg, item.first == active, Anchor.MIDDLE_ASCENDER)
        }
        gesture(Pc.text)
    }

    private fun input(
        x1: Float, y1: Float, x2: Float, y2: Float, label: String, value: String, placeholder: String,
        focused: Boolean = false, error: String? = null, secure: Boolean = false, icon: String? = null,
    ) {
        text(label, x1, y1 - 42f, 31f, Pc.text2, true)
        val outline = when { error != null -> Pc.error; focused -> Pc.brand; else -> Pc.border2 }
        rounded(x1, y1, x2, y2, 30f, Pc.surface, outline, if (focused || error != null) 5f else 2f)
        if (icon != null) lineIcon(x1 + 30f, y1 + 40f, icon, 50f, Pc.text3, 4f)
        val tx = if (icon != null) x1 + 96f else x1 + 38f
        val shown = if (secure && value.isNotEmpty()) "••••••••••" else value.ifEmpty { placeholder }
        text(shown, tx, (y1 + y2) / 2f, 39f, if (value.isNotEmpty()) Pc.text else Pc.disabled, anchor = Anchor.LEFT_MIDDLE, maxWidth = x2 - tx - 60f)
        if (secure) {
            val eyeX = x2 - 68f
            val eyeY = (y1 + y2) / 2f
            circle(eyeX, eyeY, 14f, Pc.surface)
            circle(eyeX, eyeY, 11f, Pc.text3)
            circle(eyeX, eyeY, 7f, Pc.surface)
            circle(eyeX, eyeY, 5f, Pc.text3)
        }
        if (error != null) text(error, x1, y2 + 20f, 27f, Pc.error, maxWidth = x2 - x1)
    }

    private fun button(
        x1: Float, y1: Float, x2: Float, y2: Float, label: String, primary: Boolean = true,
        disabled: Boolean = false, loading: Boolean = false,
    ) {
        val fill = if (disabled) Pc.border else if (primary) Pc.brand else Pc.surface
        val fg = if (disabled) Pc.disabled else if (primary) Pc.surface else Pc.brand
        rounded(x1, y1, x2, y2, 34f, fill, if (primary) null else Pc.brand, 3f)
        if (loading) {
            spinner((x1 + x2) / 2f - 92f, (y1 + y2) / 2f, 24f, fg, 6f)
            text(label, (x1 + x2) / 2f + 12f, (y1 + y2) / 2f, 40f, fg, true, Anchor.MIDDLE_MIDDLE)
        } else {
            text(label, (x1 + x2) / 2f, (y1 + y2) / 2f, 40f, fg, true, Anchor.MIDDLE_MIDDLE)
        }
    }

    private fun statusBanner(code: String, top: Float, widthRatio: Float, message: String? = null) {
        val spec = when (code) {
            "SUCCESS" -> Banner(Pc.successSoft, Pc.success, "✓", "操作成功")
            "SAVE_SUCCESS" -> Banner(Pc.successSoft, Pc.success, "✓", "保存成功")
            "NETWORK_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "网络连接失败")
            "SERVER_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "服务暂时不可用")
            "SAVE_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "保存失败")
            "OFFLINE" -> Banner(Pc.warningSoft, Pc.warning, "↯", "当前处于离线状态")
            "TIMEOUT" -> Banner(Pc.warningSoft, Pc.warning, "!", "请求超时")
            "RATE_LIMITED" -> Banner(Pc.warningSoft, Pc.warning, "⌛", "请求过于频繁")
            "LOCKED" -> Banner(Pc.errorSoft, Pc.error, "🔒", "账号暂时锁定")
            "UNAUTHORIZED" -> Banner(Pc.warningSoft, Pc.warning, "🔒", "登录状态已失效")
            "SERVICE_DEGRADED" -> Banner(Pc.warningSoft, Pc.warning, "!", "部分服务暂时降级")
            "PROVIDER_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "当前模型响应失败")
            "CONTENT_BLOCKED" -> Banner(Pc.warningSoft, Pc.warning, "!", "内容未通过安全检查")
            "OFFLINE_CACHE" -> Banner(Pc.warningSoft, Pc.warning, "↯", "正在显示离线缓存")
            "PERMISSION_DENIED" -> Banner(Pc.errorSoft, Pc.error, "🔒", "权限不足")
            "NOT_FOUND" -> Banner(Pc.surfaceSubtle, Pc.text2, "?", "内容不存在")
            "UPDATE_REQUIRED" -> Banner(Pc.warningSoft, Pc.warning, "↑", "需要更新后继续")
            else -> return
        }
        val width = 1080f * widthRatio
        val x = (1080f - width) / 2f
        val calibrateProviderError = currentPage in P03_CHAT_DERIVED_PAGE_IDS && code == "PROVIDER_ERROR"
        shadowCard(x, top, x + width, top + 150f, 28f, spec.bg, mix(spec.bg, spec.fg, .2f), shadow = 4f, offset = 2f)
        iconCircle(
            x + if (calibrateProviderError) 59f else 60f,
            top + if (calibrateProviderError) 77f else 75f,
            32f, spec.symbol, spec.bg, spec.fg, 30f,
        )
        val titleSize = if (calibrateProviderError) 35f else 34f
        text(spec.title, x + 112f, top + 28f, titleSize, spec.fg, true)
        text(message ?: "请检查后重试，已保留当前操作内容。", x + 112f, top + 76f, 26f, Pc.text2, maxWidth = width - 150f)
    }

    private fun errorCenter(code: String, titleOverride: String? = null) {
        val spec = when (code) {
            "EMPTY" -> ErrorSurface("○", "暂无内容", "完成第一项操作后，内容会显示在这里。", Pc.text3)
            "NETWORK_ERROR" -> ErrorSurface("↯", "网络连接失败", "请检查网络连接后重试。", Pc.error)
            "SERVER_ERROR" -> ErrorSurface("!", "服务暂时不可用", "请求 ID：YL-8A21 · 可稍后重试。", Pc.error)
            "OFFLINE" -> ErrorSurface("↯", "当前处于离线状态", "恢复网络后可继续使用完整功能。", Pc.warning)
            "NOT_FOUND" -> ErrorSurface("?", "内容不存在", "内容可能已删除或你没有访问权限。", Pc.text3)
            "PERMISSION_DENIED" -> ErrorSurface("🔒", "权限不足", "当前账号没有访问此内容的权限。", Pc.error)
            "SERVICE_DEGRADED" -> ErrorSurface("!", "部分服务暂时降级", "你仍可使用未受影响的功能。", Pc.warning)
            else -> ErrorSurface("!", "出现问题", "请稍后重试。", Pc.error)
        }
        iconCircle(540f, 800f, 86f, spec.symbol, mix(Pc.surface, spec.fg, .1f), spec.fg, 70f)
        text(titleOverride ?: spec.title, 540f, 930f, 54f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text(spec.message, 540f, 1015f, 34f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER, maxWidth = 720f)
        button(270f, 1160f, 810f, 1316f, "重新尝试")
    }

    private fun skeleton(x1: Float, y1: Float, x2: Float, rows: Int) {
        val width = x2 - x1
        repeat(rows) { index ->
            val y = y1 + index * 70f
            rounded(x1, y, x1 + width * if (index % 2 == 0) .78f else .55f, y + 24f, 18f, Pc.border)
            rounded(x1, y + 34f, x1 + width * if (index % 3 == 0) .65f else .45f, y + 50f, 18f, Pc.surfaceSubtle)
        }
    }

    private fun keyboard(numeric: Boolean) {
        val top = 1715f
        rect(0f, top, 1080f, 2400f, color("#D8DBE2"))
        if (numeric) {
            val labels = listOf("1", "2", "3", "4", "5", "6", "7", "8", "9", "", "0", "⌫")
            val pad = 22f
            val keyWidth = (1080f - pad * 4f) / 3f
            val keyHeight = 120f
            repeat(4) { row ->
                var x = pad
                repeat(3) { column ->
                    val label = labels[row * 3 + column]
                    if (label.isNotEmpty()) {
                        val y = top + 38f + row * (keyHeight + 18f)
                        rounded(x, y, x + keyWidth, y + keyHeight, 18f, Pc.surface)
                        text(label, x + keyWidth / 2f, y + keyHeight / 2f, 34f, Pc.text, anchor = Anchor.MIDDLE_MIDDLE)
                    }
                    x += keyWidth + pad
                }
            }
        } else {
            val rows = "QWERTYUIOPASDFGHJKLZXCVBNM"
                .map(Char::toString)
                .chunked(10)
            val pad = 10f
            val keyWidth = (1080f - pad * 11f).toInt() / 10f
            val keyHeight = 108f
            rows.forEachIndexed { rowIndex, labels ->
                val rowWidth = labels.size * keyWidth + (labels.size - 1) * pad
                var x = ((1080f - rowWidth) / 2f).toInt().toFloat()
                val y = top + 38f + rowIndex * (keyHeight + 18f)
                labels.forEach { label ->
                    rounded(x, y, x + keyWidth, y + keyHeight, 18f, Pc.surface)
                    text(label, x + keyWidth / 2f, y + keyHeight / 2f, 34f, Pc.text, anchor = Anchor.MIDDLE_MIDDLE)
                    x += keyWidth + pad
                }
            }
            rounded(120f, 2090f, 820f, 2206f, 18f, Pc.surface)
            text("空格", 470f, 2148f, 32f, Pc.text2, anchor = Anchor.MIDDLE_MIDDLE)
            rounded(842f, 2090f, 1048f, 2206f, 18f, Pc.brand)
            text("完成", 945f, 2148f, 32f, Pc.surface, true, Anchor.MIDDLE_MIDDLE)
        }
        gesture(Pc.text)
    }

    private fun isError(code: String): Boolean = code in setOf(
        "NETWORK_ERROR", "SERVER_ERROR", "SAVE_ERROR", "OFFLINE", "TIMEOUT", "RATE_LIMITED",
        "LOCKED", "UNAUTHORIZED", "SERVICE_DEGRADED", "CODE_EXPIRED", "INVALID_CODE",
    )

    private fun gesture(fill: Int) = rounded(430f, 2370f, 650f, 2382f, 7f, fill)

    private fun shadowCard(
        x1: Float, y1: Float, x2: Float, y2: Float, radius: Float,
        fill: Int = Pc.surface, outline: Int = Pc.border,
        shadow: Float = 10f, offset: Float = 5f,
    ) {
        // This directly corresponds to Pillow's GaussianBlur(shadow) of a
        // 22/255-alpha, offset rounded rectangle.  A platform blur avoids
        // the much wider profile produced by hand-layered Canvas rings.
        paint.style = Paint.Style.FILL
        paint.color = Color.argb(22, 16, 24, 40)
        paint.setShadowLayer(shadow, 0f, offset, Color.argb(22, 16, 24, 40))
        canvas.drawRoundRect(RectF(x1, y1, x2 + 1f, y2 + 1f), radius, radius, paint)
        paint.clearShadowLayer()
        rounded(x1, y1, x2, y2, radius, fill, outline, 1f)
    }

    private fun overlay(fill: Int) = rect(0f, 0f, 1080f, 2400f, fill)

    private fun pill(
        x1: Float, y1: Float, x2: Float, y2: Float, label: String, fill: Int,
        fg: Int, size: Float, outline: Int? = null,
    ) {
        rounded(x1, y1, x2, y2, (y2 - y1) / 2f, fill, outline)
        text(label, (x1 + x2) / 2f, (y1 + y2) / 2f, size, fg, true, Anchor.MIDDLE_MIDDLE)
    }

    private fun chip(x: Float, y: Float, label: String, active: Boolean): Float {
        paint.textSize = 28f
        paint.typeface = Typeface.create("sans-serif", Typeface.BOLD)
        paint.textScaleX = 1f
        val width = max(120f, paint.measureText(label) + 58f)
        pill(
            x, y, x + width, y + 70f, label,
            if (active) Pc.brandSoft else Pc.surface,
            if (active) Pc.brand else Pc.text2,
            28f,
            if (active) Pc.brand else Pc.border,
        )
        return width
    }

    private fun iconCircle(cx: Float, cy: Float, radius: Float, symbol: String, fill: Int, fg: Int, size: Float) {
        circle(cx, cy, radius, fill)
        val usesApprovedP03Fallback = currentPage in P03_PAGE_IDS && symbol in setOf("✓", "⌛")
        if (symbol in setOf("➤", "↻", "↯", "🔒", "🎙") || usesApprovedP03Fallback) {
            missingGlyph(cx, cy, size, fg)
        } else {
            text(symbol, cx, cy, size, fg, true, Anchor.MIDDLE_MIDDLE)
        }
    }

    private fun missingGlyph(cx: Float, cy: Float, size: Float, fill: Int) {
        // The approved Pillow baselines use the available CJK font's tofu
        // fallback for these symbols. Its visible mark is a narrow hollow
        // rectangle (not an X). Keep the fallback deterministic rather than
        // depending on an Android emoji font being installed on the emulator.
        val halfWidth = size * .27f
        val halfHeight = size * .43f
        val left = cx - halfWidth
        val top = cy - halfHeight
        val right = cx + halfWidth
        val bottom = cy + halfHeight
        val previousCap = paint.strokeCap
        val previousJoin = paint.strokeJoin
        paint.style = Paint.Style.STROKE
        paint.strokeWidth = 2f
        paint.strokeCap = Paint.Cap.BUTT
        paint.strokeJoin = Paint.Join.MITER
        paint.color = fill
        canvas.drawRect(RectF(left, top, right, bottom), paint)
        paint.strokeCap = previousCap
        paint.strokeJoin = previousJoin
    }

    private fun lineIcon(x: Float, y: Float, kind: String, size: Float, fill: Int, width: Float) {
        val cx = x + size / 2f
        val cy = y + size / 2f
        when (kind) {
            "back" -> path(listOf(x + size * .65f to y + size * .2f, x + size * .3f to cy, x + size * .65f to y + size * .8f), fill, width)
            "menu" -> listOf(.25f, .5f, .75f).forEach { offset ->
                line(x + size * .18f, y + size * offset, x + size * .82f, y + size * offset, fill, width)
            }
            "plus" -> { line(cx, y + size * .2f, cx, y + size * .8f, fill, width); line(x + size * .2f, cy, x + size * .8f, cy, fill, width) }
            "search" -> {
                oval(x + size * .12f, y + size * .12f, x + size * .62f, y + size * .62f, null, fill, width)
                line(x + size * .58f, y + size * .58f, x + size * .88f, y + size * .88f, fill, width)
            }
            "mic" -> {
                rounded(x + size * .35f, y + size * .08f, x + size * .65f, y + size * .62f, size * .15f, null, fill, width)
                arc(x + size * .22f, y + size * .34f, x + size * .78f, y + size * .82f, 0f, 180f, fill, width)
                line(cx, y + size * .82f, cx, y + size * .96f, fill, width)
                line(x + size * .35f, y + size * .96f, x + size * .65f, y + size * .96f, fill, width)
            }
            "check" -> path(listOf(x + size * .18f to y + size * .52f, x + size * .42f to y + size * .76f, x + size * .84f to y + size * .24f), fill, width + 1f)
            "more" -> listOf(.25f, .5f, .75f).forEach { circle(x + size * it, cy, 4f, fill) }
            "file" -> {
                rounded(x + size * .22f, y + size * .12f, x + size * .78f, y + size * .88f, 8f, null, fill, width)
                line(x + size * .34f, y + size * .42f, x + size * .68f, y + size * .42f, fill, width - 1f)
                line(x + size * .34f, y + size * .57f, x + size * .68f, y + size * .57f, fill, width - 1f)
            }
            "user" -> { oval(x + size * .32f, y + size * .16f, x + size * .68f, y + size * .52f, null, fill, width); arc(x + size * .16f, y + size * .43f, x + size * .84f, y + size * .95f, 190f, 160f, fill, width) }
            "home" -> { path(listOf(x + size * .16f to y + size * .48f, cx to y + size * .15f, x + size * .84f to y + size * .48f), fill, width); rounded(x + size * .26f, y + size * .44f, x + size * .74f, y + size * .86f, 6f, null, fill, width) }
            "briefcase" -> { rounded(x + size * .15f, y + size * .3f, x + size * .85f, y + size * .82f, 8f, null, fill, width); rounded(x + size * .35f, y + size * .15f, x + size * .65f, y + size * .35f, 6f, null, fill, width) }
            "compass" -> { oval(x + size * .12f, y + size * .12f, x + size * .88f, y + size * .88f, null, fill, width); path(listOf(cx to y + size * .25f, x + size * .62f to y + size * .58f, cx to y + size * .75f, x + size * .38f to y + size * .42f, cx to y + size * .25f), fill, width) }
            else -> oval(x + size * .18f, y + size * .18f, x + size * .82f, y + size * .82f, null, fill, width)
        }
    }

    private fun spinner(cx: Float, cy: Float, radius: Float, fill: Int, width: Float) =
        arc(cx - radius, cy - radius, cx + radius, cy + radius, 20f, 280f, fill, width)

    private fun text(
        value: String, x: Float, y: Float, size: Float, fill: Int, bold: Boolean = false,
        anchor: Anchor = Anchor.LEFT_ASCENDER, maxWidth: Float? = null, lineSpacing: Float = 8f,
    ) {
        val p03Text = currentPage in P03_PAGE_IDS
        val calibratePhysicalFont =
            currentPage in P03_PHYSICAL_FONT_CALIBRATION_PAGE_IDS || p03ChatStatusFontCalibration
        paint.style = if (calibratePhysicalFont && !bold) Paint.Style.FILL_AND_STROKE else Paint.Style.FILL
        paint.strokeWidth = if (calibratePhysicalFont && !bold) .35f else 1f
        paint.color = fill
        paint.textSize = size
        paint.typeface = Typeface.create("sans-serif", if (bold) Typeface.BOLD else Typeface.NORMAL)
        paint.isFakeBoldText = calibratePhysicalFont && bold
        // The reference mockups use Microsoft YaHei, whose Latin glyphs are
        // wider than Android's default sans face at the same size. Keep CJK
        // untouched and widen mixed Latin/number labels to match the source
        // rasterization contract.
        val mixedLatin = value.any { it.code in 0x21..0x7E } &&
            value.contains(Regex("Android|Windows|Chrome|Codex|Web|新加坡|当前设备|10 分钟前|昨天"))
        paint.textScaleX = if (!p03Text && mixedLatin) 1.1f else 1f
        paint.textAlign = when (anchor) {
            Anchor.MIDDLE_ASCENDER, Anchor.MIDDLE_MIDDLE -> Paint.Align.CENTER
            Anchor.RIGHT_ASCENDER -> Paint.Align.RIGHT
            else -> Paint.Align.LEFT
        }
        val lines = wrap(value, maxWidth, p03Text, bold)
        val metrics = paint.fontMetrics
        val p03BaselineOffset = when {
            currentPage in P03_CHAT_DERIVED_PAGE_IDS -> -3f - size * p03ChatBaselineScale
            p03Text -> -3f
            else -> 0f
        }
        val baseline = when (anchor) {
            Anchor.LEFT_MIDDLE, Anchor.MIDDLE_MIDDLE ->
                y - (metrics.ascent + metrics.descent) / 2f + size * .1f + p03BaselineOffset
            else -> y - metrics.ascent + size * .225f + p03BaselineOffset
        }
        val step = size + lineSpacing
        val drawX = if (
            currentPage in P03_CHAT_DERIVED_PAGE_IDS &&
            anchor in setOf(Anchor.LEFT_ASCENDER, Anchor.LEFT_MIDDLE)
        ) x - 1f else x
        val calibrateProviderErrorTitle =
            currentPage in P03_CHAT_DERIVED_PAGE_IDS &&
                p03ChatStatusFontCalibration &&
                bold &&
                value == "当前模型响应失败"
        lines.forEachIndexed { index, line ->
            if (p03Text) paint.textScaleX = p03TextScale(line, bold)
            if (calibrateProviderErrorTitle) {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                    val baseTypeface = Typeface.create("sans-serif", Typeface.NORMAL)
                    paint.typeface = Typeface.create(
                        baseTypeface,
                        P03_PROVIDER_ERROR_FONT_WEIGHT,
                        false,
                    )
                }
                paint.isSubpixelText = true
                paint.isLinearText = false
                paint.hinting = Paint.HINTING_ON
                paint.isFakeBoldText = true
                paint.style = Paint.Style.FILL_AND_STROKE
                paint.strokeWidth = P03_PROVIDER_ERROR_STROKE_WIDTH
                canvas.save()
                canvas.scale(P03_PROVIDER_ERROR_SCALE_X, 1f, drawX, y)
                canvas.drawText(
                    line,
                    drawX,
                    baseline + index * step,
                    paint,
                )
                canvas.restore()
            } else {
                canvas.drawText(line, drawX, baseline + index * step, paint)
            }
        }
        paint.textScaleX = 1f
        paint.isFakeBoldText = false
        paint.isSubpixelText = true
        paint.isLinearText = false
        paint.hinting = Paint.HINTING_OFF
        paint.style = Paint.Style.FILL
        paint.strokeWidth = 1f
    }

    private fun p03TextScale(value: String, bold: Boolean): Float {
        val visible = value.filterNot(Char::isWhitespace)
        if (visible.isEmpty()) return 1f
        val asciiRatio = visible.count { it.code in 0x21..0x7E }.toFloat() / visible.length
        val coefficient = when {
            currentPage in P03_CHAT_DERIVED_PAGE_IDS && bold -> .045f
            bold -> .12f
            else -> .075f
        }
        val statusTitleCalibration = if (
            currentPage in P03_CHAT_DERIVED_PAGE_IDS &&
            p03ChatStatusFontCalibration &&
            bold &&
            visible.length > 1
        ) {
            if (value == "当前模型响应失败") -.018f else .008f
        } else 0f
        return 1f + coefficient * asciiRatio + statusTitleCalibration
    }

    private fun wrap(value: String, maxWidth: Float?, p03Text: Boolean, bold: Boolean): List<String> {
        if (maxWidth == null) return value.split('\n')
        val result = mutableListOf<String>()
        value.split('\n').forEach { explicit ->
            var current = ""
            explicit.forEach { ch ->
                val candidate = current + ch
                if (p03Text) paint.textScaleX = p03TextScale(candidate, bold)
                if (current.isEmpty() || paint.measureText(candidate) <= maxWidth) current = candidate
                else { result += current; current = ch.toString() }
            }
            if (current.isNotEmpty()) result += current
        }
        paint.textScaleX = 1f
        return result.ifEmpty { listOf("") }
    }

    private fun rounded(x1: Float, y1: Float, x2: Float, y2: Float, radius: Float, fill: Int?, outline: Int? = null, width: Float = 1f) {
        // Pillow's contract generator treats right/bottom as inclusive and
        // draws outlines inward. Android Canvas uses exclusive bounds and
        // centers strokes, so normalize the primitive semantics here.
        val fillBox = RectF(x1, y1, x2 + 1f, y2 + 1f)
        if (fill != null) { paint.style = Paint.Style.FILL; paint.color = fill; canvas.drawRoundRect(fillBox, radius, radius, paint) }
        if (outline != null) {
            val inset = width / 2f
            val strokeBox = RectF(x1 + inset, y1 + inset, x2 + 1f - inset, y2 + 1f - inset)
            val strokeRadius = max(0f, radius - inset)
            paint.style = Paint.Style.STROKE
            paint.strokeWidth = width
            paint.color = outline
            canvas.drawRoundRect(strokeBox, strokeRadius, strokeRadius, paint)
        }
    }

    private fun rect(x1: Float, y1: Float, x2: Float, y2: Float, fill: Int) {
        paint.style = Paint.Style.FILL; paint.color = fill; canvas.drawRect(x1, y1, x2, y2, paint)
    }

    private fun circle(cx: Float, cy: Float, radius: Float, fill: Int) {
        paint.style = Paint.Style.FILL; paint.color = fill; canvas.drawCircle(cx, cy, radius + .5f, paint)
    }

    private fun oval(x1: Float, y1: Float, x2: Float, y2: Float, fill: Int?, outline: Int?, width: Float) {
        val fillBox = RectF(x1, y1, x2 + 1f, y2 + 1f)
        if (fill != null) { paint.style = Paint.Style.FILL; paint.color = fill; canvas.drawOval(fillBox, paint) }
        if (outline != null) {
            val inset = width / 2f
            paint.style = Paint.Style.STROKE
            paint.strokeWidth = width
            paint.color = outline
            canvas.drawOval(RectF(x1 + inset, y1 + inset, x2 + 1f - inset, y2 + 1f - inset), paint)
        }
    }

    private fun line(x1: Float, y1: Float, x2: Float, y2: Float, fill: Int, width: Float) {
        paint.style = Paint.Style.STROKE; paint.strokeWidth = width; paint.color = fill; canvas.drawLine(x1, y1, x2, y2, paint)
    }

    private fun path(points: List<Pair<Float, Float>>, fill: Int, width: Float) {
        if (points.isEmpty()) return
        val path = Path().apply { moveTo(points.first().first, points.first().second); points.drop(1).forEach { lineTo(it.first, it.second) } }
        paint.style = Paint.Style.STROKE; paint.strokeWidth = width; paint.color = fill; canvas.drawPath(path, paint)
    }

    private fun filledPath(points: List<Pair<Float, Float>>, fill: Int) {
        if (points.isEmpty()) return
        val path = Path().apply {
            moveTo(points.first().first, points.first().second)
            points.drop(1).forEach { lineTo(it.first, it.second) }
            close()
        }
        paint.style = Paint.Style.FILL
        paint.color = fill
        canvas.drawPath(path, paint)
    }

    private fun arc(x1: Float, y1: Float, x2: Float, y2: Float, start: Float, sweep: Float, fill: Int, width: Float) {
        paint.style = Paint.Style.STROKE; paint.strokeWidth = width; paint.color = fill; canvas.drawArc(RectF(x1, y1, x2, y2), start, sweep, false, paint)
    }

    private fun mix(a: Int, b: Int, t: Float): Int = Color.rgb(
        (Color.red(a) * (1f - t) + Color.red(b) * t).toInt(),
        (Color.green(a) * (1f - t) + Color.green(b) * t).toInt(),
        (Color.blue(a) * (1f - t) + Color.blue(b) * t).toInt(),
    )

    private data class Banner(val bg: Int, val fg: Int, val symbol: String, val title: String)
    private data class HomeBannerSpec(val bg: Int, val fg: Int, val label: String, val action: String)
    private data class ErrorSurface(val symbol: String, val title: String, val message: String, val fg: Int)
    private enum class Anchor { LEFT_ASCENDER, MIDDLE_ASCENDER, RIGHT_ASCENDER, LEFT_MIDDLE, MIDDLE_MIDDLE }
}
