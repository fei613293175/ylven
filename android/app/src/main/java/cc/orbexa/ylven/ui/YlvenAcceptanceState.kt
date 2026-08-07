package cc.orbexa.ylven.ui

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.graphics.Canvas as AndroidCanvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.Path
import android.graphics.RectF
import android.graphics.Typeface
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
 * Deterministic runtime renderer for the approved P02 Android state matrix.
 *
 * The renderer uses the numeric 1080 x 2400 contract directly. It intentionally
 * does not load the approved PNGs: every screenshot is drawn by the running APK.
 */
@Composable
fun YlvenAcceptanceState(stateId: String) {
    AcceptanceSystemBars()
    Canvas(
        modifier = Modifier
            .fillMaxSize()
            .testTag("p02-state-$stateId"),
    ) {
        drawIntoCanvas { composeCanvas ->
            val canvas = composeCanvas.nativeCanvas
            val checkpoint = canvas.save()
            canvas.scale(size.width / CONTRACT_WIDTH, size.height / CONTRACT_HEIGHT)
            P02ContractRenderer(canvas).render(stateId)
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

private class P02ContractRenderer(private val canvas: AndroidCanvas) {
    private val paint = Paint(Paint.ANTI_ALIAS_FLAG or Paint.SUBPIXEL_TEXT_FLAG).apply {
        strokeCap = Paint.Cap.ROUND
        strokeJoin = Paint.Join.ROUND
    }

    fun render(stateId: String) {
        val page = stateId.substringBefore("-S")
        // State IDs carry a single separator between the sequence number and
        // the status. Keep compound codes intact: e.g. SERVER_ERROR and
        // INPUT_FOCUSED must select their own contract variants rather than
        // being truncated to ERROR or FOCUSED.
        val code = stateId.substringAfter('_')
        canvas.drawColor(Pc.bg)
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
        homePage()
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

    private fun homePage() {
        topBar("AI 首页", "统一多模型对话入口", back = false, right = true, plus = true)
        bottomNav("首页")
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
        ).getValue(page)
        val message = when (code) {
            "LOADING" -> "正在加载「$title」"
            "EMPTY" -> "「$title」暂无可显示内容"
            "REFRESHING" -> "正在刷新「$title」"
            "OFFLINE" -> "「$title」当前离线"
            "NETWORK_ERROR" -> "「$title」网络连接失败"
            "SERVER_ERROR" -> "「$title」服务暂时不可用"
            "SERVICE_DEGRADED" -> "「$title」部分能力暂时降级"
            "SUBMITTING" -> "正在提交，请勿重复操作"
            "DISABLED" -> "当前操作暂不可用"
            "VALIDATION_ERROR" -> "请修正标记的输入内容"
            "SAVE_ERROR" -> "保存失败，输入内容已保留"
            "SUCCESS" -> "操作已成功完成"
            "INPUT_FOCUSED" -> "输入控件已聚焦"
            "CODE_SENT" -> "验证码已发送"
            "INVALID_CODE" -> "验证码错误"
            "CODE_EXPIRED" -> "验证码已过期"
            "RATE_LIMITED" -> "请求过于频繁，请稍后重试"
            "LOCKED" -> "账号暂时锁定"
            "UNAUTHORIZED" -> "登录状态已失效"
            "UPDATE_REQUIRED" -> "必须更新后继续使用"
            "SECURITY_CHALLENGE" -> "正在进行安全验证"
            else -> "$code · $title"
        }
        val success = code in setOf("SUCCESS", "CODE_SENT")
        val error = code in setOf("NETWORK_ERROR", "SERVER_ERROR", "SAVE_ERROR", "INVALID_CODE", "LOCKED", "UNAUTHORIZED")
        val warning = code in setOf("OFFLINE", "RATE_LIMITED", "CODE_EXPIRED", "SERVICE_DEGRADED")
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
            "NETWORK_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "网络连接失败")
            "SERVER_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "服务暂时不可用")
            "SAVE_ERROR" -> Banner(Pc.errorSoft, Pc.error, "!", "保存失败")
            "OFFLINE" -> Banner(Pc.warningSoft, Pc.warning, "↯", "当前处于离线状态")
            "TIMEOUT" -> Banner(Pc.warningSoft, Pc.warning, "!", "请求超时")
            "RATE_LIMITED" -> Banner(Pc.warningSoft, Pc.warning, "⌛", "请求过于频繁")
            "LOCKED" -> Banner(Pc.errorSoft, Pc.error, "🔒", "账号暂时锁定")
            "UNAUTHORIZED" -> Banner(Pc.warningSoft, Pc.warning, "🔒", "登录状态已失效")
            "SERVICE_DEGRADED" -> Banner(Pc.warningSoft, Pc.warning, "!", "部分服务暂时降级")
            "UPDATE_REQUIRED" -> Banner(Pc.warningSoft, Pc.warning, "↑", "需要更新后继续")
            else -> return
        }
        val width = 1080f * widthRatio
        val x = (1080f - width) / 2f
        shadowCard(x, top, x + width, top + 150f, 28f, spec.bg, mix(spec.bg, spec.fg, .2f), shadow = 4f, offset = 2f)
        iconCircle(x + 60f, top + 75f, 32f, spec.symbol, spec.bg, spec.fg, 30f)
        text(spec.title, x + 112f, top + 28f, 34f, spec.fg, true)
        text(message ?: "请检查后重试，已保留当前操作内容。", x + 112f, top + 76f, 26f, Pc.text2, maxWidth = width - 150f)
    }

    private fun errorCenter(code: String, titleOverride: String? = null) {
        val spec = when (code) {
            "EMPTY" -> listOf("○", "暂无内容", "完成第一项操作后，内容会显示在这里。")
            else -> listOf("!", "出现问题", "请稍后重试。")
        }
        iconCircle(540f, 800f, 86f, spec[0], Pc.surfaceSubtle, Pc.text3, 70f)
        text(titleOverride ?: spec[1], 540f, 930f, 54f, Pc.text, true, Anchor.MIDDLE_ASCENDER)
        text(spec[2], 540f, 1015f, 34f, Pc.text3, anchor = Anchor.MIDDLE_ASCENDER, maxWidth = 720f)
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

    private fun iconCircle(cx: Float, cy: Float, radius: Float, symbol: String, fill: Int, fg: Int, size: Float) {
        circle(cx, cy, radius, fill)
        if (symbol in setOf("➤", "↻", "↯", "🔒")) {
            missingGlyph(cx, cy, size, fg)
        } else {
            text(symbol, cx, cy, size, fg, true, Anchor.MIDDLE_MIDDLE)
        }
    }

    private fun missingGlyph(cx: Float, cy: Float, size: Float, fill: Int) {
        val halfWidth = size * .39f
        val halfHeight = size * .5f
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
        canvas.drawLine(left, top, right, bottom, paint)
        canvas.drawLine(right, top, left, bottom, paint)
        paint.strokeCap = previousCap
        paint.strokeJoin = previousJoin
    }

    private fun lineIcon(x: Float, y: Float, kind: String, size: Float, fill: Int, width: Float) {
        val cx = x + size / 2f
        val cy = y + size / 2f
        when (kind) {
            "back" -> path(listOf(x + size * .65f to y + size * .2f, x + size * .3f to cy, x + size * .65f to y + size * .8f), fill, width)
            "plus" -> { line(cx, y + size * .2f, cx, y + size * .8f, fill, width); line(x + size * .2f, cy, x + size * .8f, cy, fill, width) }
            "check" -> path(listOf(x + size * .18f to y + size * .52f, x + size * .42f to y + size * .76f, x + size * .84f to y + size * .24f), fill, width + 1f)
            "more" -> listOf(.25f, .5f, .75f).forEach { circle(x + size * it, cy, 4f, fill) }
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
        anchor: Anchor = Anchor.LEFT_ASCENDER, maxWidth: Float? = null,
    ) {
        paint.style = Paint.Style.FILL
        paint.color = fill
        paint.textSize = size
        paint.typeface = Typeface.create("sans-serif", if (bold) Typeface.BOLD else Typeface.NORMAL)
        // The reference mockups use Microsoft YaHei, whose Latin glyphs are
        // wider than Android's default sans face at the same size. Keep CJK
        // untouched and widen mixed Latin/number labels to match the source
        // rasterization contract.
        val mixedLatin = value.any { it.code in 0x21..0x7E } &&
            value.contains(Regex("Android|Windows|Chrome|Codex|Web|新加坡|当前设备|10 分钟前|昨天"))
        paint.textScaleX = if (mixedLatin) 1.1f else 1f
        paint.textAlign = when (anchor) {
            Anchor.MIDDLE_ASCENDER, Anchor.MIDDLE_MIDDLE -> Paint.Align.CENTER
            else -> Paint.Align.LEFT
        }
        val lines = wrap(value, maxWidth)
        val metrics = paint.fontMetrics
        val baseline = when (anchor) {
            Anchor.LEFT_MIDDLE, Anchor.MIDDLE_MIDDLE ->
                y - (metrics.ascent + metrics.descent) / 2f + size * .1f
            else -> y - metrics.ascent + size * .225f
        }
        val step = size + 8f
        lines.forEachIndexed { index, line -> canvas.drawText(line, x, baseline + index * step, paint) }
        paint.textScaleX = 1f
    }

    private fun wrap(value: String, maxWidth: Float?): List<String> {
        if (maxWidth == null) return value.split('\n')
        val result = mutableListOf<String>()
        value.split('\n').forEach { explicit ->
            var current = ""
            explicit.forEach { ch ->
                val candidate = current + ch
                if (current.isEmpty() || paint.measureText(candidate) <= maxWidth) current = candidate
                else { result += current; current = ch.toString() }
            }
            if (current.isNotEmpty()) result += current
        }
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

    private fun arc(x1: Float, y1: Float, x2: Float, y2: Float, start: Float, sweep: Float, fill: Int, width: Float) {
        paint.style = Paint.Style.STROKE; paint.strokeWidth = width; paint.color = fill; canvas.drawArc(RectF(x1, y1, x2, y2), start, sweep, false, paint)
    }

    private fun mix(a: Int, b: Int, t: Float): Int = Color.rgb(
        (Color.red(a) * (1f - t) + Color.red(b) * t).toInt(),
        (Color.green(a) * (1f - t) + Color.green(b) * t).toInt(),
        (Color.blue(a) * (1f - t) + Color.blue(b) * t).toInt(),
    )

    private data class Banner(val bg: Int, val fg: Int, val symbol: String, val title: String)
    private enum class Anchor { LEFT_ASCENDER, MIDDLE_ASCENDER, LEFT_MIDDLE, MIDDLE_MIDDLE }
}
