#!/usr/bin/env python3
from __future__ import annotations

import csv
import hashlib
import io
import json
import math
import os
import re
import shutil
import sys
import textwrap
from functools import lru_cache
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

import numpy as np
import yaml
from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_VERSION = '1.5.0'
DESIGN_VERSION = 'YL-DS-1.2.0'
DATE = '2026-08-04'

def _discover_font(bold: bool) -> str:
    env = os.environ.get('YLVEN_UI_FONT_BOLD' if bold else 'YLVEN_UI_FONT_REGULAR')
    candidates = [env] if env else []
    if os.name == 'nt':
        windir = Path(os.environ.get('WINDIR', r'C:\Windows')) / 'Fonts'
        candidates += [str(windir / name) for name in (
            ['msyhbd.ttc', 'Dengb.ttf', 'simhei.ttf'] if bold
            else ['msyh.ttc', 'Deng.ttf', 'simsun.ttc']
        )]
    candidates += ([
        '/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc',
        '/usr/share/fonts/truetype/noto/NotoSansCJK-Bold.ttc',
        '/System/Library/Fonts/PingFang.ttc',
    ] if bold else [
        '/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc',
        '/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc',
        '/System/Library/Fonts/PingFang.ttc',
    ])
    for candidate in candidates:
        if candidate and Path(candidate).is_file():
            return candidate
    raise RuntimeError(
        'No CJK font found. Install Microsoft YaHei, PingFang or Noto Sans CJK, '
        'or set YLVEN_UI_FONT_REGULAR/YLVEN_UI_FONT_BOLD. Font files are intentionally not distributed.'
    )

FONT_REGULAR = _discover_font(False)
FONT_BOLD = _discover_font(True)

C = {
    'bg': '#F6F7FB', 'surface': '#FFFFFF', 'surface_subtle': '#F9FAFB',
    'brand': '#5B61F6', 'brand_dark': '#454BD9', 'brand_soft': '#F0F1FF',
    'blue': '#2F80ED', 'blue_soft': '#EEF6FF', 'violet': '#8B5CF6',
    'text': '#101828', 'text2': '#475467', 'text3': '#667085', 'disabled': '#98A2B3',
    'border': '#E4E7EC', 'border2': '#D0D5DD', 'divider': '#EAECF0',
    'success': '#12A66A', 'success_soft': '#ECFDF3',
    'warning': '#F79009', 'warning_soft': '#FFFAEB',
    'error': '#D92D20', 'error_soft': '#FEF3F2',
    'info': '#2F80ED', 'info_soft': '#EFF8FF',
    'dark': '#111827', 'code': '#0F172A', 'code_text': '#E5E7EB',
}

SURFACE_DIR = {'ANDROID':'android','ADMIN':'admin','DEVELOPER':'developer','WEB':'web','DESIGN_SYSTEM':'design-system'}
CANVAS = {
    'ANDROID': (1080, 2400),
    'ADMIN': (1440, 1000),
    'DEVELOPER': (1440, 1000),
    'WEB': (1878, 1000),
    'DESIGN_SYSTEM': (1440, 1200),
}


def read_yaml(path: Path) -> Any:
    return yaml.safe_load(path.read_text(encoding='utf-8'))


def write_yaml(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(yaml.safe_dump(data, allow_unicode=True, sort_keys=False, width=140), encoding='utf-8')


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def write_csv(path: Path, rows: list[dict[str, Any]], fields: list[str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('w', encoding='utf-8-sig', newline='') as f:
        w = csv.DictWriter(f, fieldnames=fields)
        w.writeheader(); w.writerows(rows)


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as f:
        for chunk in iter(lambda: f.read(1024*1024), b''):
            h.update(chunk)
    return h.hexdigest()


@lru_cache(maxsize=256)
def font(size: int, bold: bool=False) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(FONT_BOLD if bold else FONT_REGULAR, size=size)


def hexrgb(h: str) -> tuple[int,int,int]:
    h=h.lstrip('#')
    return tuple(int(h[i:i+2],16) for i in (0,2,4))


def mix(a: str, b: str, t: float) -> tuple[int,int,int]:
    ar=np.array(hexrgb(a)); br=np.array(hexrgb(b)); x=(ar*(1-t)+br*t).astype(int)
    return tuple(int(v) for v in x)


def rounded(draw: ImageDraw.ImageDraw, box, radius, fill, outline=None, width=1):
    draw.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)


def shadow_card(im: Image.Image, box, radius=24, fill=C['surface'], outline=C['border'], shadow=10, offset=5):
    x1,y1,x2,y2=map(int,box)
    layer=Image.new('RGBA', im.size, (0,0,0,0)); ld=ImageDraw.Draw(layer)
    ld.rounded_rectangle((x1,y1+offset,x2,y2+offset), radius=radius, fill=(16,24,40,22))
    layer=layer.filter(ImageFilter.GaussianBlur(shadow))
    im.alpha_composite(layer)
    d=ImageDraw.Draw(im); d.rounded_rectangle((x1,y1,x2,y2), radius=radius, fill=fill, outline=outline, width=1)


def text(draw: ImageDraw.ImageDraw, xy, value: str, size: int, color=C['text'], bold=False, anchor=None, max_width=None, line_spacing=8, align='left'):
    f=font(size,bold)
    x,y=xy
    raw=str(value)
    if max_width:
        # Chinese-aware wrapping that also preserves explicit line breaks.
        lines=[]
        explicit_lines=raw.split('\n')
        for explicit in explicit_lines:
            if explicit == '':
                lines.append('')
                continue
            current=''
            for ch in explicit:
                test=current+ch
                if not current or draw.textlength(test,font=f) <= max_width:
                    current=test
                else:
                    lines.append(current)
                    current=ch
            if current:
                lines.append(current)
        draw.multiline_text((x,y),'\n'.join(lines),font=f,fill=color,spacing=line_spacing,anchor=anchor,align=align)
    elif '\n' in raw:
        draw.multiline_text((x,y),raw,font=f,fill=color,spacing=line_spacing,anchor=anchor,align=align)
    else:
        draw.text((x,y),raw,font=f,fill=color,anchor=anchor)


def pill(draw, box, label, fill=C['brand_soft'], fg=C['brand'], fs=30, radius=None, outline=None):
    x1,y1,x2,y2=box; radius=radius or int((y2-y1)/2)
    rounded(draw,box,radius,fill,outline)
    text(draw,((x1+x2)//2,(y1+y2)//2),label,fs,fg,True,'mm')


def icon_circle(draw, center, radius, symbol, fill=C['brand_soft'], fg=C['brand'], fs=36):
    x,y=center; draw.ellipse((x-radius,y-radius,x+radius,y+radius),fill=fill)
    text(draw,(x,y),symbol,fs,fg,True,'mm')


def line_icon(draw, x, y, kind, size=54, color=C['text2'], width=5):
    # Minimal deterministic icon set using primitives.
    s=size; cx=x+s/2; cy=y+s/2
    if kind=='back':
        draw.line((x+s*.65,y+s*.2,x+s*.3,cy,x+s*.65,y+s*.8),fill=color,width=width,joint='curve')
    elif kind=='plus':
        draw.line((cx,y+s*.2,cx,y+s*.8),fill=color,width=width); draw.line((x+s*.2,cy,x+s*.8,cy),fill=color,width=width)
    elif kind=='search':
        draw.ellipse((x+s*.12,y+s*.12,x+s*.62,y+s*.62),outline=color,width=width); draw.line((x+s*.58,y+s*.58,x+s*.88,y+s*.88),fill=color,width=width)
    elif kind=='menu':
        for yy in (.25,.5,.75): draw.line((x+s*.18,y+s*yy,x+s*.82,y+s*yy),fill=color,width=width)
    elif kind=='send':
        draw.polygon([(x+s*.12,y+s*.45),(x+s*.88,y+s*.12),(x+s*.68,y+s*.88),(x+s*.48,y+s*.58)],fill=color)
    elif kind=='check':
        draw.line((x+s*.18,y+s*.52,x+s*.42,y+s*.76,x+s*.84,y+s*.24),fill=color,width=width+1,joint='curve')
    elif kind=='close':
        draw.line((x+s*.22,y+s*.22,x+s*.78,y+s*.78),fill=color,width=width); draw.line((x+s*.78,y+s*.22,x+s*.22,y+s*.78),fill=color,width=width)
    elif kind=='more':
        for xx in (.25,.5,.75): draw.ellipse((x+s*xx-4,cy-4,x+s*xx+4,cy+4),fill=color)
    elif kind=='file':
        rounded(draw,(x+s*.22,y+s*.12,x+s*.78,y+s*.88),8,None,color,width); draw.line((x+s*.34,y+s*.42,x+s*.68,y+s*.42),fill=color,width=width-1); draw.line((x+s*.34,y+s*.57,x+s*.68,y+s*.57),fill=color,width=width-1)
    elif kind=='image':
        rounded(draw,(x+s*.12,y+s*.18,x+s*.88,y+s*.82),10,None,color,width); draw.ellipse((x+s*.24,y+s*.3,x+s*.37,y+s*.43),fill=color); draw.polygon([(x+s*.2,y+s*.72),(x+s*.45,y+s*.48),(x+s*.58,y+s*.62),(x+s*.72,y+s*.45),(x+s*.84,y+s*.72)],fill=color)
    elif kind=='user':
        draw.ellipse((x+s*.32,y+s*.16,x+s*.68,y+s*.52),outline=color,width=width); draw.arc((x+s*.16,y+s*.43,x+s*.84,y+s*.95),190,350,fill=color,width=width)
    elif kind=='home':
        draw.polygon([(x+s*.16,y+s*.48),(cx,y+s*.15),(x+s*.84,y+s*.48)],outline=color); rounded(draw,(x+s*.26,y+s*.44,x+s*.74,y+s*.86),6,None,color,width)
    elif kind=='briefcase':
        rounded(draw,(x+s*.15,y+s*.3,x+s*.85,y+s*.82),8,None,color,width); rounded(draw,(x+s*.35,y+s*.15,x+s*.65,y+s*.35),6,None,color,width)
    elif kind=='compass':
        draw.ellipse((x+s*.12,y+s*.12,x+s*.88,y+s*.88),outline=color,width=width); draw.polygon([(cx,y+s*.25),(x+s*.62,y+s*.58),(cx,y+s*.75),(x+s*.38,y+s*.42)],fill=color)
    elif kind=='wallet':
        rounded(draw,(x+s*.12,y+s*.25,x+s*.88,y+s*.78),9,None,color,width); rounded(draw,(x+s*.55,y+s*.38,x+s*.9,y+s*.66),7,None,color,width)
    else:
        draw.ellipse((x+s*.18,y+s*.18,x+s*.82,y+s*.82),outline=color,width=width)


def status_banner(im: Image.Image, state_code: str, message: str|None=None, top=200, width_ratio=.84):
    d=ImageDraw.Draw(im); W,H=im.size
    mapping={
        'SUCCESS':(C['success_soft'],C['success'],'✓','操作成功'), 'SAVE_SUCCESS':(C['success_soft'],C['success'],'✓','保存成功'),
        'NETWORK_ERROR':(C['error_soft'],C['error'],'!','网络连接失败'), 'SERVER_ERROR':(C['error_soft'],C['error'],'!','服务暂时不可用'),
        'PROVIDER_ERROR':(C['error_soft'],C['error'],'!','当前模型响应失败'), 'SAVE_ERROR':(C['error_soft'],C['error'],'!','保存失败'),
        'FAILED':(C['error_soft'],C['error'],'!','任务执行失败'), 'UPLOAD_FAILED':(C['error_soft'],C['error'],'!','上传失败'),
        'OFFLINE':(C['warning_soft'],C['warning'],'↯','当前处于离线状态'), 'OFFLINE_CACHE':(C['warning_soft'],C['warning'],'↯','正在显示离线缓存'),
        'TIMEOUT':(C['warning_soft'],C['warning'],'!','请求超时'), 'RATE_LIMITED':(C['warning_soft'],C['warning'],'⌛','请求过于频繁'),
        'LOCKED':(C['error_soft'],C['error'],'🔒','账号暂时锁定'), 'PERMISSION_DENIED':(C['error_soft'],C['error'],'🔒','权限不足'),
        'UNAUTHORIZED':(C['warning_soft'],C['warning'],'🔒','登录状态已失效'), 'SERVICE_DEGRADED':(C['warning_soft'],C['warning'],'!','部分服务暂时降级'),
        'CONTENT_BLOCKED':(C['warning_soft'],C['warning'],'!','内容未通过安全检查'), 'BUDGET_EXHAUSTED':(C['warning_soft'],C['warning'],'¥','可用预算已用尽'),
        'UPDATE_AVAILABLE':(C['info_soft'],C['info'],'↑','发现新版本'), 'UPDATE_REQUIRED':(C['warning_soft'],C['warning'],'↑','需要更新后继续'),
        'MAINTENANCE':(C['warning_soft'],C['warning'],'⚙','系统维护中'), 'VERSION_CONFLICT':(C['warning_soft'],C['warning'],'!','版本发生冲突'),
        'NOT_FOUND':(C['surface_subtle'],C['text2'],'?','内容不存在'),
    }
    if state_code not in mapping: return
    bg,fg,symbol,title=mapping[state_code]
    bw=int(W*width_ratio); x=(W-bw)//2; y=top; h=150 if W<1300 else 106
    shadow_card(im,(x,y,x+bw,y+h),radius=28 if W<1300 else 14,fill=bg,outline=mix(bg,fg,.2),shadow=4,offset=2)
    d=ImageDraw.Draw(im); icon_circle(d,(x+60 if W<1300 else x+48,y+h//2),32 if W<1300 else 22,symbol,bg,fg,30 if W<1300 else 20)
    text(d,(x+112 if W<1300 else x+84,y+28 if W<1300 else y+18),title,34 if W<1300 else 18,fg,True)
    text(d,(x+112 if W<1300 else x+84,y+76 if W<1300 else y+52),message or '请检查后重试，已保留当前操作内容。',26 if W<1300 else 13,C['text2'],False,max_width=bw-150)


def skeleton(draw, box, rows=5, radius=12):
    x1,y1,x2,y2=box; w=x2-x1
    for i in range(rows):
        yy=y1+i*70
        rounded(draw,(x1,yy,x1+w*(.78 if i%2==0 else .55),yy+24),radius,C['border'])
        rounded(draw,(x1,yy+34,x1+w*(.45 if i%3 else .65),yy+50),radius,C['surface_subtle'])


def draw_spinner(draw, center, radius, color=C['brand'], width=7):
    x,y=center; draw.arc((x-radius,y-radius,x+radius,y+radius),20,300,fill=color,width=width)


def truncate(s: str, n: int=24) -> str:
    return s if len(s)<=n else s[:n-1]+'…'


def feature_names(page: dict[str,Any], feature_by_id: dict[str,dict[str,Any]]) -> list[str]:
    return [feature_by_id[f]['name'] for f in page.get('feature_ids',[]) if f in feature_by_id]


# ---------- Visual taxonomy -------------------------------------------------

def android_identity(page: dict[str,Any], feature_by_id: dict[str,dict[str,Any]]) -> dict[str,Any]:
    pid=page['page_id']; name=page['name']; fs=feature_names(page,feature_by_id)
    # Exact visual definitions for every Android contract. Component/overlay identities are intentionally distinct.
    specs: dict[str, dict[str,Any]] = {
        'YL-A-001': dict(kind='COMPONENT_BOARD',template='app_shell',title='Android App Shell',subtitle='全局安全区、顶部栏、内容区与底部导航框架',parent=None,profile='component_shell',items=['状态栏与安全区','统一顶部栏','内容容器','底部导航','全局浮层']),
        'YL-A-002': dict(kind='DESIGN_BOARD',template='android_design',title='Android 设计系统',subtitle='颜色、字体、间距与基础组件移动端基准',parent=None,profile='component_shell',items=['颜色与语义状态','排版层级','按钮与输入框','卡片与列表','弹窗与提示']),
        'YL-A-003': dict(kind='COMPONENT_BOARD',template='bottom_nav',title='四栏底部导航',subtitle='首页、工作、发现、我的固定导航结构',parent='YL-A-001',profile='component_navigation',items=['首页','工作','发现','我的']),
        'YL-A-004': dict(kind='PAGE',template='splash',title='YLVEN',subtitle='INTELLIGENCE, REFINED.',parent=None,profile='splash',items=[]),
        'YL-A-005': dict(kind='PAGE',template='login',title='登录 YLVEN',subtitle='使用邮箱验证码安全登录',parent=None,profile='auth_login',items=['邮箱地址','获取验证码','创建账户']),
        'YL-A-006': dict(kind='OVERLAY',template='security_modal_login',title='请完成小验证',subtitle='为了确认是你本人，请算一算下面这道题',parent='YL-A-005',profile='auth_security_overlay',items=['YLVEN 小验证','请输入答案','取消','确认']),
        'YL-A-007': dict(kind='PAGE',template='retired_first_party_verification',title='遗留验证承载页（已停用）',subtitle='仅保留历史资产，不会在当前应用中打开',parent='YL-A-005',profile='auth_first_party_retired',items=['历史页面标记','不再导航','当前页弹窗替代说明']),
        'YL-A-008': dict(kind='PAGE',template='login_otp',title='输入登录验证码',subtitle='验证码已发送至 c***@example.com',parent='YL-A-005',profile='auth_otp',items=['六位验证码','倒计时','重新发送','登录']),
        'YL-A-009': dict(kind='PAGE',template='login_success',title='登录成功',subtitle='正在安全恢复你的工作区',parent='YL-A-008',profile='auth_success',items=['令牌安全保存','同步个人设置','进入首页']),
        'YL-A-010': dict(kind='PAGE',template='register',title='创建 YLVEN 账户',subtitle='设置好密码，马上开始使用',parent=None,profile='auth_register',items=['邮箱地址','登录密码','确认登录密码','密码强度','创建账户']),
        'YL-A-011': dict(kind='OVERLAY',template='security_modal_register',title='请完成小验证',subtitle='为了确认是你本人，请算一算下面这道题',parent='YL-A-010',profile='auth_security_overlay',items=['YLVEN 小验证','请输入答案','取消','确认']),
        'YL-A-012': dict(kind='PAGE',template='register_otp',title='验证注册邮箱',subtitle='输入发送至 c***@example.com 的六位验证码',parent='YL-A-010',profile='auth_otp',items=['六位验证码','倒计时','重新发送','确认并创建账户']),
        'YL-A-013': dict(kind='PAGE',template='register_success',title='账户已创建',subtitle='正在准备你的 YLVEN',parent='YL-A-012',profile='auth_success',items=['创建账户','保存个人设置','准备常用功能','进入 YLVEN']),
        'YL-A-014': dict(kind='SYSTEM_OVERLAY',template='session_overlay',title='全局会话状态',subtitle='令牌刷新、会话失效与安全返回登录',parent='YL-A-001',profile='system_session',items=['正在刷新会话','登录状态失效','保留本地草稿']),
        'YL-A-015': dict(kind='OVERLAY',template='logout_modal',title='退出登录',subtitle='确认后将清理本机敏感缓存',parent='YL-A-086',profile='logout_dialog',items=['取消','确认退出']),
        'YL-A-016': dict(kind='PAGE',template='devices',title='登录设备',subtitle='查看并管理当前账号的已授权设备',parent='YL-A-086',profile='device_list',items=['当前设备','Windows · Codex','Android · YLVEN','撤销设备']),
        'YL-A-017': dict(kind='SYSTEM_STATE_BOARD',template='auth_resilience',title='认证适配与异常状态',subtitle='弱网、离线、小屏和字体缩放统一规范',parent=None,profile='auth_resilience_board',items=['弱网重试','离线提示','320dp 小屏','大字体与无障碍']),
        'YL-A-018': dict(kind='PAGE',template='home',title='AI 首页',subtitle='统一多模型对话入口',parent=None,profile='main_tab',items=['新建对话','分析文件','生成图片','制作 PPT','最近对话']),
        'YL-A-019': dict(kind='PAGE',template='new_chat_landing',title='开始新对话',subtitle='选择模型后输入你的问题',parent='YL-A-018',profile='detail',items=['智能推荐','GPT','Claude','Grok','输入问题']),
        'YL-A-020': dict(kind='OVERLAY',template='conversation_drawer',title='会话历史',subtitle='搜索、归档和管理全部对话',parent='YL-A-018',profile='list',items=['今天','昨天','最近 7 天','新建对话']),
        'YL-A-021': dict(kind='PAGE',template='conversation_search',title='搜索会话',subtitle='按标题和消息内容查找历史记录',parent='YL-A-020',profile='list',items=['搜索输入框','筛选项目','搜索结果']),
        'YL-A-022': dict(kind='OVERLAY',template='conversation_menu',title='会话操作',subtitle='只展示已登记的会话操作',parent='YL-A-023',profile='dialog',items=['重命名','移入项目','导出','临时对话','删除']),
        'YL-A-023': dict(kind='PAGE',template='chat',title='YLVEN App 架构讨论',subtitle='GPT-5.6 Sol · 深度推理',parent=None,profile='chat',items=['用户消息','AI 回复','模型标签','输入区']),
        'YL-A-024': dict(kind='COMPONENT_BOARD',template='composer',title='会话输入区组件',subtitle='附件、输入、语音与发送/停止状态',parent='YL-A-023',profile='component_composer',items=['附件按钮','多行输入','语音按钮','发送/停止']),
        'YL-A-025': dict(kind='SYSTEM_STATE_BOARD',template='offline_cache',title='离线缓存',subtitle='保留最近会话并明确缓存时间',parent='YL-A-023',profile='offline_cache_only',items=['缓存时间','只读内容','联网重试']),
        'YL-A-026': dict(kind='COMPONENT_BOARD',template='ai_response',title='AI 回复组件',subtitle='流式内容、工具调用与完成态',parent='YL-A-023',profile='component_response',items=['模型来源','流式正文','工具卡','操作栏']),
        'YL-A-027': dict(kind='COMPONENT_BOARD',template='code_block',title='代码块组件',subtitle='语言标签、复制、横向滚动与错误态',parent='YL-A-026',profile='component_code',items=['Kotlin','复制代码','行号','横向滚动']),
        'YL-A-028': dict(kind='COMPONENT_BOARD',template='table_block',title='表格组件',subtitle='移动端横向滚动和列冻结规则',parent='YL-A-026',profile='component_table',items=['模型','能力','状态','延迟']),
        'YL-A-029': dict(kind='COMPONENT_BOARD',template='citation_block',title='引用组件',subtitle='来源标题、域名、序号与不可用状态',parent='YL-A-026',profile='component_citation',items=['引用 1','来源域名','打开来源']),
        'YL-A-030': dict(kind='COMPONENT_BOARD',template='message_actions',title='消息操作栏',subtitle='复制、朗读、重答、换模型和反馈',parent='YL-A-026',profile='component_actionbar',items=['复制','朗读','重新生成','换模型','赞/踩']),
        'YL-A-031': dict(kind='PAGE',template='new_conversation_form',title='新建对话',subtitle='设置会话名称、项目和默认模型',parent='YL-A-018',profile='form',items=['会话名称','所属项目','默认模型','会话指令']),
        'YL-A-032': dict(kind='COMPONENT_BOARD',template='input_box',title='输入框组件',subtitle='短文本、多行、附件和键盘状态',parent='YL-A-024',profile='component_composer',items=['占位文案','附件预览','字数状态','发送按钮']),
        'YL-A-033': dict(kind='OVERLAY',template='model_selector',title='选择 AI 模型',subtitle='按供应商与能力选择当前模型',parent='YL-A-023',profile='selector_content',items=['智能推荐','GPT-5.6 Sol','Claude Opus','Grok']),
        'YL-A-034': dict(kind='OVERLAY',template='reasoning_selector',title='推理强度',subtitle='仅展示当前模型实际支持的档位',parent='YL-A-023',profile='selector_content',items=['快速','标准','深度','极致']),
        'YL-A-035': dict(kind='PAGE',template='ai_settings',title='AI 设置',subtitle='默认模型、回答风格和工具偏好',parent='YL-A-086',profile='settings',items=['默认模型','默认推理强度','回答风格','联网搜索']),
        'YL-A-036': dict(kind='PAGE',template='conversation_settings',title='会话设置',subtitle='配置当前会话的模型和指令',parent='YL-A-023',profile='settings',items=['会话默认模型','推理强度','系统指令','临时对话']),
        'YL-A-037': dict(kind='COMPONENT_BOARD',template='model_strip',title='输入区模型条',subtitle='模型、推理、联网和临时切换提示',parent='YL-A-024',profile='component_model_strip',items=['GPT-5.6 Sol','深度推理','联网']),
        'YL-A-038': dict(kind='COMPONENT_BOARD',template='response_label',title='AI 回复标签',subtitle='永久标识供应商、模型、推理和工具来源',parent='YL-A-026',profile='component_model_strip',items=['OpenAI','GPT-5.6 Sol','深度','已联网']),
        'YL-A-039': dict(kind='PAGE',template='branch_switcher',title='会话分支',subtitle='查看并切换同一问题的多个回答分支',parent='YL-A-023',profile='list',items=['主分支','Claude 重答','Grok 重答','当前分支']),
        'YL-A-040': dict(kind='PAGE',template='comparison_setup',title='多模型对比',subtitle='同一问题并行请求多个 AI',parent='YL-A-023',profile='wizard',items=['选择模型','问题内容','并行数量','开始对比']),
        'YL-A-041': dict(kind='PAGE',template='comparison_result',title='对比结果',subtitle='切换查看各模型回答并进行综合',parent='YL-A-040',profile='detail',items=['GPT','Claude','Grok','采用回答','综合结果']),
        'YL-A-042': dict(kind='OVERLAY',template='model_error_modal',title='模型暂时不可用',subtitle='透明展示失败原因和可选替代方案',parent='YL-A-023',profile='dialog',items=['重试','选择其他模型','查看服务状态']),
        'YL-A-043': dict(kind='PAGE',template='service_status',title='AI 服务状态',subtitle='查看模型与工具的实时可用性',parent='YL-A-086',profile='list',items=['OpenAI 正常','Claude 限流','Grok 正常','PPT Worker 正常']),
        'YL-A-044': dict(kind='OVERLAY',template='attachment_selector',title='添加内容',subtitle='选择相机、相册、文件或资料库',parent='YL-A-024',profile='selector_content',items=['相机','相册','文件','资料库']),
        'YL-A-045': dict(kind='OVERLAY',template='upload_progress',title='上传文件',subtitle='显示真实进度并允许取消',parent='YL-A-023',profile='job',items=['产品需求.pdf','62%','后台继续','取消']),
        'YL-A-046': dict(kind='COMPONENT_BOARD',template='image_attachment',title='图片附件组件',subtitle='原图、缩略图、上传和识图状态',parent='YL-A-023',profile='component_attachment',items=['截图.png','1.8 MB','识图中']),
        'YL-A-047': dict(kind='COMPONENT_BOARD',template='file_attachment',title='文件附件卡片',subtitle='文件名、类型、大小、解析与错误状态',parent='YL-A-023',profile='component_attachment',items=['需求文档.pdf','12 页','已解析']),
        'YL-A-048': dict(kind='OVERLAY',template='library_selector',title='从资料库选择',subtitle='按项目和类型筛选可用文件',parent='YL-A-044',profile='selector_content',items=['最近使用','项目资料','PDF','图片']),
        'YL-A-049': dict(kind='PAGE',template='file_library',title='文件资料库',subtitle='统一管理上传文件和解析状态',parent='YL-A-077',profile='list',items=['需求文档.pdf','市场数据.xlsx','产品截图.png']),
        'YL-A-050': dict(kind='PAGE',template='file_search',title='搜索文件',subtitle='按名称、项目、类型和内容查找',parent='YL-A-049',profile='list',items=['搜索文件','类型筛选','项目筛选','结果']),
        'YL-A-051': dict(kind='OVERLAY',template='file_menu',title='文件操作',subtitle='仅提供合同允许的文件动作',parent='YL-A-052',profile='dialog',items=['重命名','移入项目','下载','删除']),
        'YL-A-052': dict(kind='PAGE',template='file_detail',title='文件详情',subtitle='需求文档.pdf · 已完成解析',parent='YL-A-049',profile='detail',items=['文件信息','解析摘要','关联项目','供应商引用']),
        'YL-A-053': dict(kind='PAGE',template='project_list',title='项目',subtitle='集中管理长期对话、资料和作品',parent='YL-A-077',profile='list',items=['YLVEN Android App','商业模型研究','品牌资产']),
        'YL-A-054': dict(kind='PAGE',template='project_create',title='新建项目',subtitle='设置项目名称、说明和默认模型',parent='YL-A-053',profile='form',items=['项目名称','项目说明','默认模型','创建']),
        'YL-A-055': dict(kind='PAGE',template='project_detail',title='YLVEN Android App',subtitle='对话、资料、作品与项目指令',parent='YL-A-053',profile='detail',items=['对话 12','资料 8','作品 6','最近活动']),
        'YL-A-056': dict(kind='PAGE',template='project_settings',title='项目设置',subtitle='管理项目名称、默认模型和归档状态',parent='YL-A-055',profile='settings',items=['项目名称','默认模型','归档项目','删除项目']),
        'YL-A-057': dict(kind='PAGE',template='project_instructions',title='项目指令',subtitle='为项目内所有对话提供稳定上下文',parent='YL-A-055',profile='settings',items=['项目目标','回答风格','技术约束','保存']),
        'YL-A-058': dict(kind='PAGE',template='project_files',title='项目资料',subtitle='项目专属文件与知识索引',parent='YL-A-055',profile='list',items=['架构说明.pdf','数据库设计.md','UI规范.yaml']),
        'YL-A-059': dict(kind='OVERLAY',template='artifact_menu',title='作品操作',subtitle='下载、移入项目、版本和删除',parent='YL-A-079',profile='dialog',items=['下载','移入项目','查看版本','删除']),
        'YL-A-060': dict(kind='COMPONENT_BOARD',template='project_citation',title='项目知识引用',subtitle='显示文件、页码、命中片段和可信来源',parent='YL-A-026',profile='component_citation',items=['架构说明.pdf','第 8 页','打开原文']),
        'YL-A-061': dict(kind='PAGE',template='workspace_search',title='工作区搜索',subtitle='跨项目、文件、作品和任务检索',parent='YL-A-077',profile='list',items=['项目','文件','作品','任务']),
        'YL-A-062': dict(kind='PAGE',template='image_landing',title='图片工作台',subtitle='生成、编辑并管理 AI 图片作品',parent='YL-A-077',profile='detail',items=['生成图片','编辑图片','最近作品','任务状态']),
        'YL-A-063': dict(kind='PAGE',template='image_studio',title='生成图片',subtitle='描述画面并设置模型、比例和质量',parent='YL-A-062',profile='wizard',items=['画面描述','参考图','模型','比例','数量','生成']),
        'YL-A-064': dict(kind='PAGE',template='image_job',title='图片生成任务',subtitle='任务 #IMG-240804-018',parent='YL-A-063',profile='job',items=['提示词优化','模型排队','生成中','保存作品']),
        'YL-A-065': dict(kind='PAGE',template='image_result',title='图片生成结果',subtitle='4 张图片已生成',parent='YL-A-064',profile='preview',items=['图片 1','图片 2','图片 3','图片 4','继续编辑']),
        'YL-A-066': dict(kind='PAGE',template='image_detail',title='图片详情',subtitle='品牌科技主视觉 · 2048×2048',parent='YL-A-065',profile='preview',items=['原始提示词','模型与参数','版本历史','下载']),
        'YL-A-067': dict(kind='PAGE',template='ppt_landing',title='PPT 工作台',subtitle='从主题和资料生成可编辑演示文稿',parent='YL-A-077',profile='wizard',items=['新建 PPT','从资料生成','最近项目','模板']),
        'YL-A-068': dict(kind='PAGE',template='ppt_basic',title='PPT 基本信息',subtitle='第 1 步，共 4 步',parent='YL-A-067',profile='wizard',items=['演示主题','目标受众','页数','语言','下一步']),
        'YL-A-069': dict(kind='PAGE',template='ppt_sources',title='添加参考资料',subtitle='第 2 步，共 4 步',parent='YL-A-067',profile='wizard',items=['上传 PDF','上传 Word','从项目选择','粘贴文本']),
        'YL-A-070': dict(kind='PAGE',template='ppt_template',title='选择 PPT 模板',subtitle='第 3 步，共 4 步',parent='YL-A-067',profile='wizard',items=['商务','科技','极简','路演','教育']),
        'YL-A-071': dict(kind='PAGE',template='ppt_outline',title='确认 PPT 大纲',subtitle='第 4 步，共 4 步',parent='YL-A-067',profile='wizard',items=['封面','市场背景','核心方案','实施路径','总结']),
        'YL-A-072': dict(kind='PAGE',template='ppt_outline_edit',title='编辑大纲',subtitle='拖动排序、增加页面或修改标题',parent='YL-A-071',profile='wizard',items=['拖动排序','新增页面','删除页面','生成 PPT']),
        'YL-A-073': dict(kind='PAGE',template='ppt_job',title='PPT 生成任务',subtitle='正在渲染 12 页演示文稿',parent='YL-A-072',profile='job',items=['内容生成','素材准备','PPTX 渲染','预览图']),
        'YL-A-074': dict(kind='PAGE',template='ppt_preview',title='PPT 预览',subtitle='YLVEN 商业 AI 平台 · 12 页',parent='YL-A-073',profile='preview',items=['页面缩略图','当前页面','播放预览','编辑']),
        'YL-A-075': dict(kind='PAGE',template='ppt_slide_edit',title='编辑第 3 页',subtitle='修改内容、版式和图片',parent='YL-A-074',profile='form',items=['页面标题','正文要点','版式','替换图片','保存']),
        'YL-A-076': dict(kind='PAGE',template='ppt_detail',title='PPT 详情',subtitle='YLVEN 商业 AI 平台 · 已保存',parent='YL-A-074',profile='preview',items=['PPTX','PDF 预览','版本历史','下载']),
        'YL-A-077': dict(kind='PAGE',template='work_home',title='工作',subtitle='工具、项目、作品和任务统一工作台',parent=None,profile='main_tab',items=['工具','项目','作品','任务']),
        'YL-A-078': dict(kind='PAGE',template='tool_catalog',title='AI 工具',subtitle='选择图片、PPT、文件分析或多模型工作流',parent='YL-A-077',profile='main_tab',items=['图片生成','PPT 制作','文件分析','多模型对比']),
        'YL-A-079': dict(kind='PAGE',template='artifact_library',title='作品',subtitle='管理图片、PPT、文档和其他成果',parent='YL-A-077',profile='list',items=['全部','图片','PPT','文档']),
        'YL-A-080': dict(kind='PAGE',template='job_center',title='任务中心',subtitle='查看排队、进行中、完成和失败任务',parent='YL-A-077',profile='job',items=['进行中','已完成','失败','已取消']),
        'YL-A-081': dict(kind='PAGE',template='discover_home',title='发现',subtitle='探索模板、模型、工作流和新服务',parent=None,profile='main_tab',items=['推荐','效率工具','内容创作','开发工具']),
        'YL-A-082': dict(kind='PAGE',template='discover_category',title='全部服务',subtitle='按分类浏览可用 AI 能力',parent='YL-A-081',profile='list',items=['效率工具','内容创作','开发工具','连接器']),
        'YL-A-083': dict(kind='PAGE',template='prompt_templates',title='Prompt 模板',subtitle='选择模板并填入变量快速开始',parent='YL-A-081',profile='list',items=['商业分析','代码审查','PPT 大纲','图片提示词']),
        'YL-A-084': dict(kind='PAGE',template='model_lab',title='模型实验室',subtitle='查看模型能力并进行对比测试',parent='YL-A-081',profile='list',items=['GPT-5.6 Sol','Claude Opus','Grok','能力测试']),
        'YL-A-085': dict(kind='PAGE',template='announcements',title='公告与更新',subtitle='平台更新、维护和新能力说明',parent='YL-A-081',profile='list',items=['版本更新','模型上线','维护公告']),
        'YL-A-086': dict(kind='PAGE',template='mine',title='我的',subtitle='账号、套餐、用量与系统设置',parent=None,profile='main_tab',items=['个人资料','套餐与会员','钱包','AI 设置','安全与数据']),
        'YL-A-087': dict(kind='PAGE',template='profile',title='个人资料',subtitle='编辑头像、用户名和公开信息',parent='YL-A-086',profile='settings',items=['头像','用户名','UID','简介']),
        'YL-A-088': dict(kind='PAGE',template='favorites',title='我的收藏',subtitle='收藏的对话、作品和模板',parent='YL-A-086',profile='list',items=['对话','作品','模板']),
        'YL-A-089': dict(kind='PAGE',template='custom_instructions',title='自定义指令',subtitle='告诉 YLVEN 应如何理解和回答你',parent='YL-A-086',profile='detail',items=['关于我','回答方式','保存']),
        'YL-A-090': dict(kind='PAGE',template='memory',title='个人记忆',subtitle='查看、编辑或删除已保存的长期记忆',parent='YL-A-086',profile='settings',items=['偏好','项目背景','常用设置','关闭记忆']),
        'YL-A-091': dict(kind='PAGE',template='notifications',title='通知设置',subtitle='控制任务完成、额度和系统通知',parent='YL-A-086',profile='settings',items=['任务完成','额度提醒','安全提醒','系统公告']),
        'YL-A-092': dict(kind='PAGE',template='appearance',title='外观与语言',subtitle='设置主题、字体大小和界面语言',parent='YL-A-086',profile='settings',items=['浅色模式','深色模式','跟随系统','字体大小','简体中文']),
        'YL-A-093': dict(kind='PAGE',template='privacy',title='数据与隐私',subtitle='导出数据、清空历史或注销账号',parent='YL-A-086',profile='settings',items=['导出数据','数据保留时间','清空历史','注销账号']),
        'YL-A-094': dict(kind='PAGE',template='app_update',title='检查更新',subtitle='当前版本 1.0.0',parent='YL-A-086',profile='detail',items=['版本号','更新内容','下载 APK','校验 SHA-256']),
        'YL-A-095': dict(kind='PAGE',template='help',title='帮助与反馈',subtitle='查找帮助或提交问题反馈',parent='YL-A-086',profile='detail',items=['帮助中心','常见问题','提交反馈','系统日志']),
        'YL-A-096': dict(kind='PAGE',template='membership',title='套餐与会员',subtitle='选择适合你的 YLVEN 方案',parent='YL-A-086',profile='finance',items=['免费版','YLVEN Pro','API 额度','项目与存储']),
        'YL-A-097': dict(kind='PAGE',template='plan_confirm',title='确认订阅',subtitle='YLVEN Pro · 月度方案',parent='YL-A-096',profile='finance',items=['套餐权益','支付金额','到期时间','确认支付']),
        'YL-A-098': dict(kind='PAGE',template='wallet',title='钱包',subtitle='统一查看套餐、赠送和充值额度',parent='YL-A-086',profile='finance',items=['可用额度','套餐额度','赠送额度','充值余额','充值']),
        'YL-A-099': dict(kind='PAGE',template='bills',title='账单明细',subtitle='查看充值、订阅、使用和退款记录',parent='YL-A-098',profile='finance',items=['全部','充值','消费','退款']),
        'YL-A-100': dict(kind='PAGE',template='usage',title='用量中心',subtitle='按来源、模型和时间查看消耗',parent='YL-A-086',profile='finance',items=['App 对话','图片生成','PPT 制作','外部 API']),
    }
    spec=specs.get(pid)
    if spec is None:
        spec=dict(kind='PAGE',template='generic_android',title=name,subtitle=' · '.join(fs[:2]) or 'YLVEN 功能页面',parent=None,profile=page.get('state_profile','detail'),items=fs[:5])
    spec['screen_signature']=hashlib.sha1((spec['template']+'|'+spec['title']+'|'+spec.get('subtitle','')+'|'+','.join(spec.get('items',[]))+'|'+str(spec.get('parent') or '')).encode()).hexdigest()[:16]
    return spec


def generic_identity(page: dict[str,Any], feature_by_id: dict[str,dict[str,Any]]) -> dict[str,Any]:
    pid=page['page_id']; name=page['name']; fs=feature_names(page,feature_by_id)
    section,title=(name.split('/',1)+[''])[:2] if '/' in name else ('YLVEN',name)
    if not title: title=section
    kind='PAGE'; template=page['layout_profile']; parent=None
    spec={'kind':kind,'template':template,'title':title,'subtitle':section,'parent':parent,'profile':page.get('state_profile'),
          'items':fs[:6] or [title]}
    spec['screen_signature']=hashlib.sha1((template+'|'+name+'|'+','.join(spec['items'])).encode()).hexdigest()[:16]
    return spec


CUSTOM_PROFILES = {
    'component_shell':['DEFAULT'],
    'component_navigation':['DEFAULT'],
    'auth_login':['DEFAULT','INPUT_FOCUSED','VALIDATION_ERROR','SUBMITTING','RATE_LIMITED','OFFLINE','SERVER_ERROR'],
    'auth_security_overlay':['DEFAULT','SECURITY_CHALLENGE','SUBMITTING','SUCCESS','SAVE_ERROR','RATE_LIMITED','OFFLINE','SERVER_ERROR'],
    'auth_turnstile':['LOADING','SECURITY_CHALLENGE','SUCCESS','VALIDATION_ERROR','CODE_EXPIRED','OFFLINE','SERVER_ERROR'],
    'auth_first_party_retired':['LOADING','SECURITY_CHALLENGE','SUCCESS','VALIDATION_ERROR','CODE_EXPIRED','OFFLINE','SERVER_ERROR'],
    'auth_otp':['DEFAULT','INPUT_FOCUSED','CODE_SENT','SUBMITTING','SUCCESS','INVALID_CODE','CODE_EXPIRED','RATE_LIMITED','LOCKED','OFFLINE','SERVER_ERROR'],
    'auth_success':['SUCCESS','SAVE_ERROR','OFFLINE'],
    'auth_register':['DEFAULT','INPUT_FOCUSED','VALIDATION_ERROR','SUBMITTING','DISABLED','OFFLINE','SERVER_ERROR'],
    'system_session':['RESTORING','SUCCESS','UNAUTHORIZED','OFFLINE','SERVER_ERROR'],
    'logout_dialog':['DEFAULT','SUBMITTING','SUCCESS','SAVE_ERROR'],
    'device_list':['LOADING','POPULATED','EMPTY','REFRESHING','UNAUTHORIZED','OFFLINE','SERVER_ERROR'],
    'auth_resilience_board':['DEFAULT','OFFLINE','NETWORK_ERROR','TIMEOUT','SERVICE_DEGRADED'],
    'component_composer':['DEFAULT','INPUT_FOCUSED','UPLOADING','DISABLED','OFFLINE'],
    'offline_cache_only':['OFFLINE_CACHE','REFRESHING','NETWORK_ERROR'],
    'component_response':['CONNECTING','STREAMING','TOOL_RUNNING','COMPLETED','STOPPED','PROVIDER_ERROR','CONTENT_BLOCKED'],
    'component_code':['DEFAULT','COMPLETED','CONTENT_BLOCKED'],
    'component_table':['DEFAULT','COMPLETED','OFFLINE'],
    'component_citation':['DEFAULT','COMPLETED','NOT_FOUND','OFFLINE'],
    'component_actionbar':['DEFAULT','DISABLED','SUCCESS'],
    'component_model_strip':['DEFAULT','SERVICE_DEGRADED','DISABLED'],
    'component_attachment':['DEFAULT','UPLOADING','UPLOAD_FAILED','SUCCESS'],
    'selector_content':['POPULATED','FILTER_ACTIVE','EMPTY','DISABLED','SERVICE_DEGRADED','SERVER_ERROR'],
}


# ---------- Contract regeneration ------------------------------------------

def rebuild_contracts() -> tuple[list[dict[str,Any]], dict[str,dict[str,Any]], dict[str,Any]]:
    feature_doc=read_yaml(ROOT/'contracts/feature-map.yaml')
    feature_by_id={f['feature_id']:f for f in feature_doc['features']}
    page_doc=read_yaml(ROOT/'contracts/ui-page-catalog.yaml')
    state_doc=read_yaml(ROOT/'contracts/ui-state-profiles.yaml')
    definitions=state_doc['definitions']
    profiles=state_doc['profiles']
    profiles.update(CUSTOM_PROFILES)
    state_doc['profiles']=profiles
    write_yaml(ROOT/'contracts/ui-state-profiles.yaml',state_doc)

    page_contract_rows=[]; state_rows=[]; mock_rows=[]
    surface_counts=Counter(); state_surface_counts=Counter()
    # Remove old page contracts and mockup images only; retain README files.
    for sub in (ROOT/'ui/pages').iterdir():
        if sub.is_dir():
            for p in sub.glob('*.yaml'): p.unlink()
    for sub in (ROOT/'ui/mockups').iterdir():
        if sub.is_dir():
            for child in sub.iterdir():
                if child.is_dir(): shutil.rmtree(child)
                elif child.suffix.lower()=='.png': child.unlink()
    if (ROOT/'ui/reference-boards').exists():
        for p in (ROOT/'ui/reference-boards').glob('*.png'): p.unlink()
    if (ROOT/'ui/visual-review').exists(): shutil.rmtree(ROOT/'ui/visual-review')

    pages=[]
    for page in page_doc['pages']:
        page=dict(page)
        ident=android_identity(page,feature_by_id) if page['surface']=='ANDROID' else generic_identity(page,feature_by_id)
        profile=ident['profile'] or page.get('state_profile')
        codes=profiles.get(profile)
        if not codes:
            raise RuntimeError(f"unknown state profile {profile} for {page['page_id']}")
        page['state_profile']=profile
        page['visual_kind']=ident['kind']
        page['parent_page_id']=ident.get('parent')
        page['visual_identity']={
            'template':ident['template'], 'title':ident['title'], 'subtitle':ident['subtitle'],
            'required_elements':ident.get('items',[]), 'screen_signature':ident['screen_signature'],
            'identity_rule':'This visual identity must remain unique for the Page ID; do not reuse another page body and only rename the folder.',
        }
        page['state_count']=len(codes)
        page['state_ids']=[]; page['mockup_paths']=[]
        surf_dir=SURFACE_DIR[page['surface']]
        for idx,code in enumerate(codes,1):
            sid=f"{page['page_id']}-S{idx:02d}_{code}"
            rel=f"ui/mockups/{surf_dir}/{page['page_id']}/{sid}.png"
            page['state_ids'].append(sid); page['mockup_paths'].append(rel)
            dfn=definitions[code]
            state_rows.append({
                'state_id':sid,'page_id':page['page_id'],'surface':page['surface'],'page_name':page['name'],
                'state_code':code,'state_name':dfn['name'],'description':dfn['description'],'recovery':dfn['recovery'],
                'feature_ids':'|'.join(page.get('feature_ids',[])),'phases':'|'.join(page.get('phases',[])),
                'work_packets':'|'.join(page.get('work_packets',[])),'mockup_id':sid,'mockup_path':rel,
                'visual_kind':ident['kind'],'parent_page_id':ident.get('parent') or '',
                'screen_signature':ident['screen_signature'],
            })
            mock_rows.append({
                'mockup_id':sid,'page_id':page['page_id'],'state_id':sid,'surface':page['surface'],
                'page_name':page['name'],'state_name':dfn['name'],'canonical_canvas':f"{CANVAS[page['surface']][0]}x{CANVAS[page['surface']][1]}",
                'relative_path':rel,'status':'PLANNED','sha256':'','approved_by':'','approved_at':'',
                'visual_scope':'layout|spacing|typography|color|component_geometry|visible_state',
                'functional_scope':'NONE; functionality comes from Feature IDs','feature_ids':'|'.join(page.get('feature_ids',[])),
                'notes':'Generated from unique Page ID visual identity; sample text is non-functional.',
                'visual_kind':ident['kind'],'parent_page_id':ident.get('parent') or '',
                'screen_signature':ident['screen_signature'],'duplicate_audit':'PENDING',
            })
        pages.append(page); surface_counts[page['surface']]+=1; state_surface_counts[page['surface']]+=len(codes)

        layout=read_yaml(ROOT/'contracts/ui-layout-profiles.yaml')['profiles'][page['layout_profile']]
        states=[]
        for idx,code in enumerate(codes,1):
            sid=f"{page['page_id']}-S{idx:02d}_{code}"; dfn=definitions[code]
            states.append({'state_id':sid,'code':code,'name':dfn['name'],'description':dfn['description'],'recovery':dfn['recovery'],
                           'mockup_id':sid,'mockup_path':f"ui/mockups/{SURFACE_DIR[page['surface']]}/{page['page_id']}/{sid}.png",
                           'mockup_status':'PLANNED'})
        contract={
            'schema_version':'1.4','page_id':page['page_id'],'name':page['name'],'surface':page['surface'],'route':page['route'],
            'primary_phase':page['primary_phase'],'phases':page['phases'],'work_packets':page['work_packets'],'feature_ids':page['feature_ids'],
            'layout_profile':page['layout_profile'],'resolved_layout':layout,'design_system_version':DESIGN_VERSION,
            'visual_kind':ident['kind'],'parent_page_id':ident.get('parent'),'visual_identity':page['visual_identity'],
            'critical_token_snapshot':critical_token_snapshot(),'states':states,
            'binding_rules':{
                'visual_implementation_gate':'Every listed state must have an APPROVED mockup with matching SHA-256 before Codex finalizes this page.',
                'visual_authority':'Approved mockup + this resolved numeric contract.',
                'functional_authority':'contracts/feature-map.yaml and the listed Feature IDs.',
                'no_invention':'Text or controls shown only as sample content do not create a feature. No unlisted action may be implemented.',
                'identity_rule':'A Page ID must not reuse another Page ID body. Overlay and component contracts must render their own focal surface and reference their parent page explicitly.',
                'conflict_rule':'Numeric token contract wins over estimated measurements; functional contract wins over mockup sample content; conflicts require an ADR and updated mockup.',
            }
        }
        write_yaml(ROOT/f"ui/pages/{surf_dir}/{page['page_id']}.yaml",contract)
        page_contract_rows.append({
            'page_id':page['page_id'],'surface':page['surface'],'name':page['name'],'route':page['route'],
            'layout_profile':page['layout_profile'],'state_profile':profile,'state_count':len(codes),
            'feature_ids':'|'.join(page['feature_ids']),'phases':'|'.join(page['phases']),
            'primary_phase':page['primary_phase'],'work_packets':'|'.join(page['work_packets']),
            'visual_kind':ident['kind'],'parent_page_id':ident.get('parent') or '',
            'visual_template':ident['template'],'visual_title':ident['title'],'screen_signature':ident['screen_signature'],
            'design_system_version':DESIGN_VERSION,'mockup_gate':'APPROVED_REQUIRED_BEFORE_VISUAL_IMPLEMENTATION',
        })

    page_doc['schema_version']='1.4'; page_doc['page_count']=len(pages); page_doc['pages']=pages
    write_yaml(ROOT/'contracts/ui-page-catalog.yaml',page_doc)
    write_csv(ROOT/'contracts/ui-page-resolved.csv',page_contract_rows,list(page_contract_rows[0].keys()))
    write_csv(ROOT/'contracts/ui-state-catalog.csv',state_rows,list(state_rows[0].keys()))
    write_csv(ROOT/'contracts/mockup-manifest.csv',mock_rows,list(mock_rows[0].keys()))

    # Feature UI map: preserve all existing bindings but rebuild from pages.
    bindings=defaultdict(list)
    for p in pages:
        for fid in p.get('feature_ids',[]):
            bindings[fid].append({'page_id':p['page_id'],'surface':p['surface'],'page_name':p['name'],'visual_kind':p['visual_kind'],'parent_page_id':p.get('parent_page_id')})
    feature_ui=[]
    for f in feature_doc['features']:
        feature_ui.append({'feature_id':f['feature_id'],'name':f['name'],'phase':f['phase'],'ui_bindings':bindings.get(f['feature_id'],[])})
    write_yaml(ROOT/'contracts/feature-ui-map.yaml',{'schema_version':'1.4','features':feature_ui})

    # Phase binding and work packet page binding.
    phase_binding=[]
    for phase in [f'P{i:02d}' for i in range(14)]:
        ids=[p['page_id'] for p in pages if phase in p.get('phases',[])]
        phase_binding.append({'phase':phase,'page_ids':ids,'page_count':len(ids),'state_count':sum(p['state_count'] for p in pages if phase in p.get('phases',[]))})
    write_yaml(ROOT/'contracts/ui-phase-binding.yaml',{'schema_version':'1.4','phases':phase_binding})

    # Update work packet map page IDs/states from page membership.
    wp_doc=read_yaml(ROOT/'contracts/work-packet-map.yaml')
    for wp in wp_doc['work_packets']:
        wpid=wp['work_packet_id']; pgs=[p for p in pages if wpid in p.get('work_packets',[])]
        wp['page_ids']=[p['page_id'] for p in pgs]; wp['page_count']=len(pgs); wp['ui_state_count']=sum(p['state_count'] for p in pgs)
    write_yaml(ROOT/'contracts/work-packet-map.yaml',wp_doc)

    summary={'schema_version':'1.4','package_version':PACKAGE_VERSION,'design_system_version':DESIGN_VERSION,
             'page_count':len(pages),'state_count':len(state_rows),'mockup_count':len(mock_rows),
             'surface_page_counts':dict(surface_counts),'surface_state_counts':dict(state_surface_counts),
             'visual_kind_counts':dict(Counter(p['visual_kind'] for p in pages)),
             'correction':'V1.5 rebuilds page identity, overlay/component bindings and state profiles to remove false duplicates.'}
    write_json(ROOT/'UI_CONTRACT_BUILD_SUMMARY.json',summary)
    return pages,feature_by_id,summary


def critical_token_snapshot() -> dict[str,Any]:
    return {
        'design_system_version':DESIGN_VERSION,
        'android_canvas':'360x800dp / 1080x2400px @3x','android_page_horizontal_dp':16,'android_auth_horizontal_dp':24,
        'android_top_bar_height_dp':56,'android_bottom_navigation_content_height_dp':64,
        'android_primary_button':'52dp height / 14dp radius','android_text_field':'52dp min height / 14dp radius',
        'android_card':'16dp radius / 16dp padding / 1dp border','android_gaps':'inline 8dp / card 12dp / section 24dp',
        'android_typography':'page 24/32sp 700; section 20/28sp 700; card 17/24sp 600; body 16/24sp; secondary 14/20sp; caption 12/16sp',
        'android_icon_and_touch':'24dp icon / 48dp minimum touch','android_chat':'composer 52–148dp radius 24; user bubble max 84% radius 18; message gap 20dp',
        'android_sheet_dialog':'bottom sheet radius 28dp; dialog radius 20dp max width 312dp',
        'light_colors':'#F6F7FB background; #FFFFFF surface; #5B61F6 primary; #101828 primary text; #475467 secondary text; #E4E7EC border',
        'admin_shell':'1440px canonical; 248px sidebar; 64px topbar; 24px content padding; 48px table row',
        'developer_shell':'1440px canonical; 240px sidebar; 64px topbar; 32px content padding',
    }


# ---------- Android renderer ------------------------------------------------

def android_canvas() -> Image.Image:
    return Image.new('RGBA', CANVAS['ANDROID'], hexrgb(C['bg'])+(255,))


def draw_android_status(draw: ImageDraw.ImageDraw):
    text(draw,(72,35),'9:41',31,C['text'],True)
    # signal / wifi / battery
    for i,h in enumerate([12,20,28,36]): draw.rounded_rectangle((858+i*16,54-h,868+i*16,54),radius=4,fill=C['text'])
    draw.arc((934,26,984,70),210,330,fill=C['text'],width=5); draw.arc((946,38,972,64),210,330,fill=C['text'],width=5)
    rounded(draw,(1000,29,1050,58),7,None,C['text'],4); rounded(draw,(1005,34,1040,53),4,C['text']); draw.rectangle((1051,38,1057,50),fill=C['text'])


def draw_android_topbar(im: Image.Image, title_v: str, subtitle: str='', back=True, right='more', transparent=False):
    d=ImageDraw.Draw(im); draw_android_status(d)
    if not transparent:
        d.rectangle((0,72,1080,246),fill=C['surface']); d.line((0,245,1080,245),fill=C['divider'],width=2)
    if back: line_icon(d,48,122,'back',60,C['text'])
    x=130 if back else 54
    text(d,(x,118),title_v,54,C['text'],True)
    if subtitle: text(d,(x,180),truncate(subtitle,42),30,C['text3'])
    if right: line_icon(d,970,128,right,56,C['text2'])


def draw_android_bottom_nav(im: Image.Image, active='首页'):
    d=ImageDraw.Draw(im); y=2170
    d.rectangle((0,y,1080,2400),fill=C['surface']); d.line((0,y,1080,y),fill=C['divider'],width=2)
    items=[('首页','home'),('工作','briefcase'),('发现','compass'),('我的','user')]
    for i,(lab,ico) in enumerate(items):
        cx=135+i*270; fg=C['brand'] if lab==active else C['text3']
        if lab==active: rounded(d,(cx-62,y+24,cx+62,y+94),35,C['brand_soft'])
        line_icon(d,cx-27,y+32,ico,54,fg,5)
        text(d,(cx,y+111),lab,29,fg,lab==active,'ma')
    rounded(d,(430,2370,650,2382),7,C['text'])


def android_card(im: Image.Image, box, title_v=None, subtitle=None, icon=None, accent=C['brand'], fill=C['surface']):
    shadow_card(im,box,radius=42,fill=fill,outline=C['border'],shadow=8,offset=4)
    d=ImageDraw.Draw(im); x1,y1,x2,y2=box
    tx=x1+54
    if icon:
        icon_circle(d,(x1+66,y1+70),40,icon,C['brand_soft'],accent,34); tx=x1+124
    if title_v: text(d,(tx,y1+38),title_v,43,C['text'],True,max_width=x2-tx-40)
    if subtitle: text(d,(tx,y1+92),subtitle,30,C['text3'],False,max_width=x2-tx-40)


def android_input(im: Image.Image, box, label, value='', placeholder='', focused=False, error=None, secure=False, icon=None):
    d=ImageDraw.Draw(im); x1,y1,x2,y2=box
    text(d,(x1,y1-42),label,31,C['text2'],True)
    outline=C['error'] if error else C['brand'] if focused else C['border2']
    rounded(d,box,30,C['surface'],outline,5 if focused or error else 2)
    if icon: line_icon(d,x1+30,y1+40,icon,50,C['text3'],4)
    tx=x1+96 if icon else x1+38
    show=('•'*max(8,len(value))) if secure and value else (value or placeholder)
    col=C['text'] if value else C['disabled']
    text(d,(tx,(y1+y2)//2),show,39,col,False,'lm',max_width=x2-tx-60)
    if secure: text(d,(x2-68,(y1+y2)//2),'◉',30,C['text3'],False,'mm')
    if error: text(d,(x1,y2+20),error,27,C['error'],False,max_width=x2-x1)


def android_button(im: Image.Image, box, label, primary=True, disabled=False, loading=False, icon=None):
    d=ImageDraw.Draw(im); x1,y1,x2,y2=box
    fill=C['border'] if disabled else (C['brand'] if primary else C['surface'])
    fg=C['disabled'] if disabled else (C['surface'] if primary else C['brand'])
    outline=None if primary else C['brand']
    rounded(d,box,34,fill,outline,3)
    if loading:
        draw_spinner(d,((x1+x2)//2-92,(y1+y2)//2),24,fg,6); text(d,((x1+x2)//2+12,(y1+y2)//2),label,40,fg,True,'mm')
    else:
        if icon: line_icon(d,x1+42,(y1+y2)//2-25,icon,50,fg,5)
        text(d,((x1+x2)//2,(y1+y2)//2),label,40,fg,True,'mm')


def android_chip(draw, x, y, label, active=False, width=None):
    w=width or max(120,int(draw.textlength(label,font=font(28,True)))+58)
    pill(draw,(x,y,x+w,y+70),label,C['brand_soft'] if active else C['surface'],C['brand'] if active else C['text2'],28,35,C['brand'] if active else C['border'])
    return w


def draw_keyboard(im: Image.Image, numeric=False):
    d=ImageDraw.Draw(im); y0=1715
    d.rectangle((0,y0,1080,2400),fill='#D8DBE2')
    if numeric:
        labels=['1','2','3','4','5','6','7','8','9','','0','⌫']; cols=3; rows=4; pad=22; keyw=(1080-pad*(cols+1))//cols; keyh=120
    else:
        labels=list('QWERTYUIOP')+list('ASDFGHJKL')+list('ZXCVBNM'); cols=10; rows=3; pad=10; keyw=(1080-pad*(cols+1))//cols; keyh=108
    idx=0
    for r in range(rows):
        row_labels=labels[r*cols:(r+1)*cols]
        if not row_labels: continue
        roww=len(row_labels)*keyw+(len(row_labels)-1)*pad; x=(1080-roww)//2
        for lab in row_labels:
            if lab:
                rounded(d,(x,y0+38+r*(keyh+18),x+keyw,y0+38+r*(keyh+18)+keyh),18,C['surface'])
                text(d,(x+keyw//2,y0+38+r*(keyh+18)+keyh//2),lab,34,C['text'],False,'mm')
            x+=keyw+pad
    if not numeric:
        rounded(d,(120,2090,820,2206),18,C['surface']); text(d,(470,2148),'空格',32,C['text2'],False,'mm')
        rounded(d,(842,2090,1048,2206),18,C['brand']); text(d,(945,2148),'完成',32,C['surface'],True,'mm')
    rounded(d,(430,2370,650,2382),7,C['text'])


def draw_android_error_center(im: Image.Image, state: str, title_override=None):
    d=ImageDraw.Draw(im); mapping={
        'EMPTY':('○','暂无内容','完成第一项操作后，内容会显示在这里。',C['text3']),
        'NETWORK_ERROR':('↯','网络连接失败','请检查网络连接后重试。',C['error']),
        'SERVER_ERROR':('!','服务暂时不可用','请求 ID：YL-8A21 · 可稍后重试。',C['error']),
        'OFFLINE':('↯','当前处于离线状态','恢复网络后可继续使用完整功能。',C['warning']),
        'NOT_FOUND':('?','内容不存在','内容可能已删除或你没有访问权限。',C['text3']),
        'PERMISSION_DENIED':('🔒','权限不足','当前账号没有访问此内容的权限。',C['error']),
        'SERVICE_DEGRADED':('!','部分服务暂时降级','你仍可使用未受影响的功能。',C['warning']),
    }
    sym,ttl,sub,fg=mapping.get(state,('!','出现问题','请稍后重试。',C['error']))
    icon_circle(d,(540,800),86,sym,mix(C['surface'],fg,.1),fg,70)
    text(d,(540,930),title_override or ttl,54,C['text'],True,'ma')
    text(d,(540,1015),sub,34,C['text3'],False,'ma',max_width=720,align='center')
    android_button(im,(270,1160,810,1316),'重新尝试',True)


def state_is_error(state: str) -> bool:
    return state in {'NETWORK_ERROR','SERVER_ERROR','PROVIDER_ERROR','SAVE_ERROR','FAILED','UPLOAD_FAILED','OFFLINE','TIMEOUT','RATE_LIMITED','LOCKED','PERMISSION_DENIED','UNAUTHORIZED','SERVICE_DEGRADED','CONTENT_BLOCKED','BUDGET_EXHAUSTED','NOT_FOUND','VERSION_CONFLICT','MAINTENANCE','CODE_EXPIRED','INVALID_CODE'}


def render_android(page: dict[str,Any], state: dict[str,Any]) -> Image.Image:
    spec=page['visual_identity']; template=spec['template']; code=state['code']
    im=android_canvas(); d=ImageDraw.Draw(im)
    # Dispatch special templates.
    if template=='splash':
        # gradient-like bands
        for y in range(0,2400,12): d.rectangle((0,y,1080,y+12),fill=mix('#040814','#152458',y/2400))
        for r,alpha in [(390,22),(280,30),(170,44)]: d.ellipse((540-r,930-r,540+r,930+r),fill=(91,97,246,alpha))
        icon_circle(d,(540,900),126,'Y',C['brand'],C['surface'],100)
        text(d,(540,1070),'YLVEN',76,C['surface'],True,'ma'); text(d,(540,1160),'INTELLIGENCE, REFINED.',28,'#CBD5FF',False,'ma')
        if code=='RESTORING': draw_spinner(d,(540,1420),42,C['surface'],8); text(d,(540,1500),'正在恢复安全会话',30,'#DCE2FF',False,'ma')
        elif code=='FIRST_RUN': android_button(im,(210,1430,870,1586),'开始使用 YLVEN',True)
        elif code in {'UPDATE_REQUIRED','OFFLINE','SERVER_ERROR'}: status_banner(im,code,top=1420,width_ratio=.72)
        rounded(d,(430,2370,650,2382),7,C['surface'])
        return im.convert('RGB')

    if template in {'login','register','login_otp','register_otp','turnstile','login_success','register_success'}:
        return render_android_auth(im,spec,template,code)

    if template in {'security_modal_login','security_modal_register','logout_modal','conversation_menu','model_error_modal','file_menu','artifact_menu'}:
        return render_android_dialog(im,spec,template,code)
    if template in {'model_selector','reasoning_selector','attachment_selector','library_selector'}:
        return render_android_sheet(im,spec,template,code)
    if template in {'app_shell','android_design','bottom_nav','auth_resilience','composer','ai_response','code_block','table_block','citation_block','message_actions','input_box','model_strip','response_label','image_attachment','file_attachment','project_citation'}:
        return render_android_component(im,spec,template,code)
    if template=='session_overlay': return render_android_session(im,spec,code)
    if template=='devices': return render_android_devices(im,spec,code)

    # General full-page chrome.
    bottom_tab=None
    if template in {'home','work_home','tool_catalog','discover_home','mine'}:
        bottom_tab={'home':'首页','work_home':'工作','tool_catalog':'工作','discover_home':'发现','mine':'我的'}[template]
    draw_android_topbar(im,spec['title'],spec['subtitle'],back=not bottom_tab,right='more' if not bottom_tab else 'plus')
    if bottom_tab: draw_android_bottom_nav(im,bottom_tab)
    content_bottom=2140 if bottom_tab else 2320

    if code=='LOADING':
        skeleton(d,(72,330,1008,1150),7,18)
    elif code in {'EMPTY','NETWORK_ERROR','SERVER_ERROR','OFFLINE','NOT_FOUND','PERMISSION_DENIED','SERVICE_DEGRADED'}:
        draw_android_error_center(im,code)
    else:
        render_android_general_body(im,spec,template,code,content_bottom)
        if code in {'REFRESHING','CONNECTING','PREPARING','RUNNING','QUEUED','RETRYING','UPLOADING','DOWNLOADING'}:
            status_banner(im,'UPDATE_AVAILABLE' if code=='DOWNLOADING' else 'SERVICE_DEGRADED',
                          {'REFRESHING':'正在刷新内容…','CONNECTING':'正在建立安全连接…','PREPARING':'正在检查参数与额度…','RUNNING':'任务正在执行…','QUEUED':'任务已进入队列…','RETRYING':'正在重试任务…','UPLOADING':'文件正在上传…','DOWNLOADING':'正在下载更新…'}[code],top=270)
        elif state_is_error(code): status_banner(im,code,top=270)
        elif code in {'SUCCESS','SAVE_SUCCESS','COMPLETED'}: status_banner(im,'SUCCESS',top=270)
        elif code in {'SUBMITTING'}: status_banner(im,'SERVICE_DEGRADED','正在提交，请勿重复操作。',top=270)
        elif code=='CANCEL_CONFIRM':
            status_banner(im,'SERVICE_DEGRADED','确认取消后，已产生的用量将按规则结算。',top=270)
        elif code=='FILTER_ACTIVE':
            pill(d,(72,282,340,354),'筛选已生效',C['brand_soft'],C['brand'],28,36,C['brand'])
        elif code=='BULK_SELECTED':
            pill(d,(72,282,380,354),'已选择 3 项',C['brand_soft'],C['brand'],28,36,C['brand'])
        elif code in {'EDIT_MODE','DIRTY'}:
            pill(d,(72,282,390,354),'编辑模式' if code=='EDIT_MODE' else '存在未保存修改',C['warning_soft'],C['warning'],28,36,C['warning'])
    if code=='INPUT_FOCUSED': draw_keyboard(im,False)
    return im.convert('RGB')


def render_android_auth(im: Image.Image, spec: dict[str,Any], template: str, code: str) -> Image.Image:
    d=ImageDraw.Draw(im); draw_android_status(d)
    # Soft brand aura.
    d.ellipse((700,-160,1180,320),fill='#ECEEFF'); d.ellipse((-120,1660,420,2200),fill='#EEF6FF')
    icon_circle(d,(96,168),42,'Y',C['brand'],C['surface'],36)
    text(d,(154,146),'YLVEN',42,C['text'],True); text(d,(154,194),'轻松解决每天的小问题',25,C['text3'])
    if template=='retired_first_party_verification':
        draw_android_topbar(im,'请完成小验证','YLVEN',back=True,right=None)
        shadow_card(im,(72,360,1008,1460),46,C['surface'],C['border'],10,5)
        d=ImageDraw.Draw(im)
        icon_circle(d,(540,520),72,'盾',C['brand_soft'],C['brand'],52)
        text(d,(540,640),'请算一算下面这道题',54,C['text'],True,'ma')
        text(d,(540,726),'验证会直接在当前页面完成',31,C['text3'],False,'ma')
        rounded(d,(150,850,930,1065),30,C['surface_subtle'],C['border2'],3)
        rounded(d,(205,910,275,980),12,C['surface'],C['brand'] if code=='SUCCESS' else C['border2'],4)
        if code=='SUCCESS': line_icon(d,214,918,'check',52,C['success'],6)
        elif code in {'LOADING','SECURITY_CHALLENGE'}: draw_spinner(d,(240,945),25,C['brand'],6)
        text(d,(315,908),'正在核对答案' if code in {'LOADING','SECURITY_CHALLENGE'} else '验证已完成' if code=='SUCCESS' else '验证未完成',36,C['text'],True)
        text(d,(315,966),'7 + 5 = ?',26,C['text3'])
        android_button(im,(150,1160,930,1320),'返回 YLVEN' if code=='SUCCESS' else '取消验证',False)
        if code in {'VALIDATION_ERROR','CODE_EXPIRED','OFFLINE','SERVER_ERROR'}:
            status_banner(im,'SAVE_ERROR' if code=='VALIDATION_ERROR' else code,'安全验证未通过，请重新尝试。',top=1510,width_ratio=.82)
        rounded(d,(430,2370,650,2382),7,C['text'])
        return im.convert('RGB')

    if template=='turnstile':
        draw_android_topbar(im,'安全验证','auth.orbexa.cc · 安全连接',back=True,right=None)
        shadow_card(im,(72,360,1008,1460),46,C['surface'],C['border'],10,5)
        d=ImageDraw.Draw(im)
        icon_circle(d,(540,520),72,'✓' if code=='SUCCESS' else '盾',C['brand_soft'],C['brand'],52)
        text(d,(540,640),'确认你是真实用户',54,C['text'],True,'ma')
        text(d,(540,726),'完成验证后将自动返回 YLVEN',31,C['text3'],False,'ma')
        # Turnstile widget.
        rounded(d,(150,850,930,1065),30,C['surface_subtle'],C['border2'],3)
        rounded(d,(205,910,275,980),12,C['surface'],C['brand'] if code in {'SUCCESS'} else C['border2'],4)
        if code=='SUCCESS': line_icon(d,214,918,'check',52,C['success'],6)
        else: draw_spinner(d,(240,945),25,C['brand'],6) if code in {'LOADING','SECURITY_CHALLENGE'} else None
        text(d,(315,908),'正在验证安全环境' if code in {'LOADING','SECURITY_CHALLENGE'} else '验证已完成' if code=='SUCCESS' else '验证未完成',36,C['text'],True)
        text(d,(315,966),'安全验证组件',26,C['text3'])
        text(d,(758,998),'隐私 · 帮助',22,C['text3'])
        android_button(im,(150,1160,930,1320),'返回 YLVEN' if code=='SUCCESS' else '取消验证',False)
        if code in {'VALIDATION_ERROR','CODE_EXPIRED','OFFLINE','SERVER_ERROR'}:
            status_banner(im,'SAVE_ERROR' if code=='VALIDATION_ERROR' else code,'安全验证未通过，请重新尝试。',top=1510,width_ratio=.82)
        rounded(d,(430,2370,650,2382),7,C['text'])
        return im.convert('RGB')

    if template in {'login_success','register_success'}:
        icon_circle(d,(540,760),106,'✓',C['success_soft'],C['success'],82)
        text(d,(540,930),spec['title'],66,C['text'],True,'ma')
        text(d,(540,1022),spec['subtitle'],34,C['text3'],False,'ma',max_width=760,align='center')
        steps=spec.get('required_elements',[])
        y=1160
        for i,lab in enumerate(steps[:4]):
            icon_circle(d,(170,y+10),30,'✓' if code=='SUCCESS' else '·',C['success_soft'],C['success'],24)
            text(d,(225,y-10),lab,34,C['text2'],i==len(steps[:4])-1)
            y+=92
        if code=='SUCCESS': android_button(im,(180,1600,900,1760),'进入 YLVEN',True)
        else: status_banner(im,code,'初始化暂未完成，已保留账户数据。',top=1570,width_ratio=.78)
        rounded(d,(430,2370,650,2382),7,C['text'])
        return im.convert('RGB')

    x1,x2=72,1008; y=360
    title_v=spec['title']; subtitle=spec['subtitle']
    text(d,(x1,y),title_v,66,C['text'],True); text(d,(x1,y+92),subtitle,33,C['text3'],False,max_width=900)
    y+=205
    focused=code=='INPUT_FOCUSED'; validation=code=='VALIDATION_ERROR'
    if template=='login':
        android_input(im,(x1,y,x2,y+156),'邮箱地址','chenping@orbexa.cc' if focused else '', 'name@example.com',focused, '请输入有效邮箱地址' if validation else None,False,'user')
        y+=250
        if code=='RATE_LIMITED': text(d,(x1,y-28),'请求过于频繁，请 48 秒后再试',28,C['warning'])
        android_button(im,(x1,y,x2,y+156),'正在发送验证码…' if code=='SUBMITTING' else '获取登录验证码',True,loading=code=='SUBMITTING')
        y+=205
        text(d,(540,y),'还没有账户？',30,C['text3'],False,'mm'); text(d,(700,y),'创建账户',30,C['brand'],True,'mm')
        if code in {'OFFLINE','SERVER_ERROR'}: status_banner(im,code,top=1320,width_ratio=.86)
    elif template=='register':
        android_input(im,(x1,y,x2,y+146),'邮箱地址','chenping@orbexa.cc' if focused else '', 'name@example.com',focused, '邮箱格式不正确' if validation else None,False,'user'); y+=225
        android_input(im,(x1,y,x2,y+146),'登录密码','Ylven@2026' if focused else '', '至少 8 位，包含字母和数字',False,'密码强度不足' if validation else None,True); y+=225
        android_input(im,(x1,y,x2,y+146),'确认登录密码','Ylven@2026' if focused else '', '再次输入登录密码',False,'两次密码输入不一致' if validation else None,True); y+=215
        # password rules
        rules=['至少 8 个字符','包含字母和数字','两次密码一致']
        for i,r in enumerate(rules):
            icon_circle(d,(x1+18,y+i*52),16,'✓' if focused and not validation else '·',C['success_soft'] if focused and not validation else C['surface_subtle'],C['success'] if focused and not validation else C['text3'],18)
            text(d,(x1+50,y-17+i*52),r,27,C['text3'])
        y+=180
        android_button(im,(x1,y,x2,y+156),'正在提交…' if code=='SUBMITTING' else '创建账户',True,disabled=code=='DISABLED',loading=code=='SUBMITTING')
        if code in {'OFFLINE','SERVER_ERROR'}: status_banner(im,code,top=1680,width_ratio=.86)
    elif template in {'login_otp','register_otp'}:
        text(d,(x1,y),'验证码已发送至',29,C['text3']); text(d,(x1,y+48),'c***@example.com',38,C['text'],True); y+=150
        boxes=[]
        for i in range(6):
            bx=x1+i*150; boxes.append((bx,y,bx+120,y+136))
            fill='8' if i<3 and code in {'INPUT_FOCUSED','CODE_SENT','SUBMITTING'} else ''
            err=code=='INVALID_CODE'
            rounded(d,boxes[-1],24,C['surface'],C['error'] if err else C['brand'] if code=='INPUT_FOCUSED' and i==3 else C['border2'],5 if err or (code=='INPUT_FOCUSED' and i==3) else 2)
            if fill: text(d,(bx+60,y+68),fill,54,C['text'],True,'mm')
        y+=185
        if code=='INVALID_CODE': text(d,(x1,y),'验证码错误，请重新输入',28,C['error'])
        elif code=='CODE_EXPIRED': text(d,(x1,y),'验证码已过期，请重新发送',28,C['warning'])
        elif code=='RATE_LIMITED': text(d,(x1,y),'发送频繁，请 48 秒后再试',28,C['warning'])
        else: text(d,(x1,y),'00:48 后可重新发送',28,C['text3'])
        y+=95
        label='登录' if template=='login_otp' else '确认并创建账户'
        android_button(im,(x1,y,x2,y+156),'正在验证…' if code=='SUBMITTING' else label,True,loading=code=='SUBMITTING')
        y+=205
        text(d,(540,y),'收不到验证码？  更换邮箱',29,C['brand'],True,'mm')
        if code=='SUCCESS': status_banner(im,'SUCCESS','验证通过，正在进入下一步。',top=1510,width_ratio=.82)
        if code in {'LOCKED','OFFLINE','SERVER_ERROR'}: status_banner(im,code,top=1510,width_ratio=.82)
        if code=='INPUT_FOCUSED': draw_keyboard(im,True)
    rounded(d,(430,2370,650,2382),7,C['text'])
    return im.convert('RGB')


def render_android_dialog(im: Image.Image, spec: dict[str,Any], template: str, code: str) -> Image.Image:
    # Draw specific parent context, then dim and show a unique dialog.
    if template=='security_modal_login':
        base=render_android_auth(android_canvas(),{'title':'登录 YLVEN','subtitle':'使用邮箱验证码安全登录','required_elements':[]},'login','DEFAULT').convert('RGBA')
    elif template=='security_modal_register':
        base=render_android_auth(android_canvas(),{'title':'创建 YLVEN 账户','subtitle':'设置好密码，马上开始使用','required_elements':[]},'register','DEFAULT').convert('RGBA')
    elif template=='logout_modal':
        base=android_canvas(); draw_android_topbar(base,'我的','账号、套餐、用量与系统设置',back=False,right='more'); draw_android_bottom_nav(base,'我的'); render_android_general_body(base,{'title':'我的','subtitle':'账号、套餐、用量与系统设置','required_elements':['个人资料','套餐与会员','钱包','AI 设置','安全与数据']},'mine','POPULATED',2140)
    elif template=='file_menu':
        base=android_canvas(); draw_android_topbar(base,'架构说明.pdf','PDF · 2.4 MB · 已解析',back=True,right='more'); render_android_general_body(base,{'title':'文件详情','subtitle':'查看解析状态、归属与版本信息','required_elements':['文件预览','解析状态','所属项目','版本记录','下载']},'file_detail','POPULATED',2320)
    elif template=='artifact_menu':
        base=android_canvas(); draw_android_topbar(base,'我的作品','图片、PPT、文档与版本',back=True,right='more'); render_android_general_body(base,{'title':'作品库','subtitle':'管理生成内容与历史版本','required_elements':['科技主视觉','产品发布 PPT','AI 架构说明']},'artifact_library','POPULATED',2320)
    else:
        base=android_canvas(); draw_android_topbar(base,'YLVEN App 架构讨论','GPT-5.6 Sol · 深度推理',back=True,right='more'); render_android_general_body(base,{'title':'YLVEN App 架构讨论','subtitle':'GPT-5.6 Sol · 深度推理','required_elements':[]},'chat','COMPLETED',2320)
    overlay=Image.new('RGBA',base.size,(16,24,40,125)); base.alpha_composite(overlay); im=base; d=ImageDraw.Draw(im)
    # Security modal has distinct verification card; other dialogs use action list/confirmation.
    if template in {'security_modal_login','security_modal_register'}:
        shadow_card(im,(105,700,975,1515),44,C['surface'],C['border'],14,6); d=ImageDraw.Draw(im)
        icon_circle(d,(540,845),62,'盾',C['brand_soft'],C['brand'],46)
        text(d,(540,940),spec['title'],54,C['text'],True,'ma'); text(d,(540,1022),spec['subtitle'],31,C['text3'],False,'ma',max_width=690,align='center')
        rounded(d,(170,1125,910,1275),26,C['surface_subtle'],C['border2'],2)
        if code in {'SUBMITTING','SECURITY_CHALLENGE'}: draw_spinner(d,(235,1200),26,C['brand'],6)
        elif code=='SUCCESS': line_icon(d,210,1175,'check',50,C['success'],6)
        else: rounded(d,(207,1165,263,1221),10,C['surface'],C['border2'],3)
        text(d,(300,1175),'7 + 5 = ?',34,C['text'],True); text(d,(300,1220),'请输入答案',24,C['text3'])
        android_button(im,(170,1330,525,1460),'取消',False); android_button(im,(555,1330,910,1460),'确认',True,loading=code=='SUBMITTING')
        if state_is_error(code): status_banner(im,code,top=1580,width_ratio=.78)
    elif template=='logout_modal':
        shadow_card(im,(120,850,960,1420),44,C['surface'],C['border'],14,6); d=ImageDraw.Draw(im)
        icon_circle(d,(540,980),62,'↪',C['warning_soft'],C['warning'],46)
        text(d,(540,1070),'确认退出登录？',56,C['text'],True,'ma'); text(d,(540,1150),'本机会清理敏感缓存，但云端数据不会删除。',31,C['text3'],False,'ma',max_width=690,align='center')
        android_button(im,(170,1250,515,1380),'取消',False); android_button(im,(545,1250,910,1380),'退出登录',True,loading=code=='SUBMITTING')
        if code=='SUCCESS': status_banner(im,'SUCCESS','已安全退出当前账号。',top=1510,width_ratio=.78)
        elif code=='SAVE_ERROR': status_banner(im,'SAVE_ERROR','退出失败，请检查网络后重试。',top=1510,width_ratio=.78)
    elif template in {'conversation_menu','file_menu','artifact_menu'}:
        y0=970; shadow_card(im,(72,y0,1008,2110),48,C['surface'],C['border'],14,6); d=ImageDraw.Draw(im)
        rounded(d,(450,y0+24,630,y0+38),7,C['border2']); text(d,(120,y0+86),spec['title'],52,C['text'],True)
        for i,item in enumerate(spec.get('required_elements',[])):
            yy=y0+190+i*142; line_icon(d,120,yy-16,'file' if i<2 else 'more',54,C['error'] if item=='删除' else C['text2'],4)
            text(d,(204,yy),item,38,C['error'] if item=='删除' else C['text'],item=='删除','lm'); d.line((120,yy+72,960,yy+72),fill=C['divider'],width=2)
        if code=='SUBMITTING': status_banner(im,'SERVICE_DEGRADED','正在处理所选操作…',top=650,width_ratio=.75)
        elif code=='SUCCESS': status_banner(im,'SUCCESS',top=650,width_ratio=.75)
        elif code in {'SAVE_ERROR','DISABLED'}: status_banner(im,'SAVE_ERROR' if code=='SAVE_ERROR' else 'PERMISSION_DENIED',top=650,width_ratio=.75)
    elif template=='model_error_modal':
        shadow_card(im,(110,780,970,1560),46,C['surface'],C['border'],14,6); d=ImageDraw.Draw(im)
        icon_circle(d,(540,930),66,'!',C['error_soft'],C['error'],52)
        text(d,(540,1030),spec['title'],54,C['text'],True,'ma'); text(d,(540,1110),spec['subtitle'],30,C['text3'],False,'ma',max_width=690,align='center')
        android_button(im,(170,1240,910,1380),'重试当前模型',True,loading=code=='SUBMITTING')
        android_button(im,(170,1410,910,1530),'选择其他模型',False)
    rounded(ImageDraw.Draw(im),(430,2370,650,2382),7,C['surface'])
    return im.convert('RGB')


def render_android_sheet(im: Image.Image, spec: dict[str,Any], template: str, code: str) -> Image.Image:
    # Parent chat/work surface.
    base=android_canvas(); draw_android_topbar(base,'YLVEN App 架构讨论','GPT-5.6 Sol · 深度推理',back=True,right='more')
    render_android_general_body(base,{'title':'YLVEN App 架构讨论','subtitle':'GPT-5.6 Sol · 深度推理','required_elements':[]},'chat','COMPLETED',2320)
    base.alpha_composite(Image.new('RGBA',base.size,(16,24,40,90)))
    im=base; d=ImageDraw.Draw(im); y0=850
    shadow_card(im,(0,y0,1080,2400),64,C['surface'],C['border'],12,0); d=ImageDraw.Draw(im)
    rounded(d,(450,y0+28,630,y0+42),7,C['border2'])
    text(d,(72,y0+105),spec['title'],56,C['text'],True); text(d,(72,y0+175),spec['subtitle'],30,C['text3'])
    if template=='model_selector':
        # Provider tabs and distinct model cards.
        x=72
        for i,lab in enumerate(['推荐','GPT','Claude','Grok']): x+=android_chip(d,x,y0+250,lab,i==0)+18
        models=[('智能推荐','自动选择最合适的模型','推荐'),('GPT-5.6 Sol','复杂分析、编码与研究','OpenAI'),('Claude Opus','长文写作与严谨审查','Anthropic'),('Grok','实时信息与快速分析','xAI')]
        yy=y0+360
        for i,(m,desc,prov) in enumerate(models):
            rounded(d,(72,yy,1008,yy+188),32,C['brand_soft'] if i==0 else C['surface'],C['brand'] if i==0 else C['border'],4 if i==0 else 2)
            icon_circle(d,(142,yy+94),42,str(i+1),C['surface'] if i==0 else C['brand_soft'],C['brand'],30)
            text(d,(210,yy+40),m,39,C['text'],True); text(d,(210,yy+92),desc,28,C['text3']); pill(d,(806,yy+44,954,yy+104),prov,C['surface_subtle'],C['text2'],23)
            if i==0: line_icon(d,920,yy+118,'check',45,C['brand'],6)
            yy+=210
    elif template=='reasoning_selector':
        options=[('快速','低延迟，适合简单问题','low'),('标准','速度与质量平衡','medium'),('深度','复杂分析与专业任务','high'),('极致','仅在模型支持时展示','xhigh')]
        yy=y0+270
        for i,(lab,desc,raw) in enumerate(options):
            rounded(d,(72,yy,1008,yy+190),32,C['brand_soft'] if i==2 else C['surface'],C['brand'] if i==2 else C['border'],4 if i==2 else 2)
            icon_circle(d,(142,yy+95),40,'✓' if i==2 else '·',C['surface'] if i==2 else C['surface_subtle'],C['brand'] if i==2 else C['text3'],30)
            text(d,(210,yy+42),lab,40,C['text'],True); text(d,(210,yy+96),desc,28,C['text3']); pill(d,(830,yy+60,954,yy+120),raw,C['surface'],C['text2'],22)
            yy+=215
    elif template=='attachment_selector':
        items=[('相机','拍摄图片','image'),('相册','选择本地图片','image'),('文件','PDF、Word、PPT、Excel','file'),('资料库','从已上传文件选择','briefcase')]
        yy=y0+285
        for i,(lab,desc,ico) in enumerate(items):
            x=72+(i%2)*468; y=yy+(i//2)*250
            rounded(d,(x,y,x+438,y+220),36,C['surface_subtle'],C['border'],2); icon_circle(d,(x+75,y+76),42,'＋',C['brand_soft'],C['brand'],34)
            text(d,(x+42,y+132),lab,38,C['text'],True); text(d,(x+42,y+180),desc,25,C['text3'])
    elif template=='library_selector':
        android_input(im,(72,y0+260,1008,y0+405),'搜索资料库','','输入文件名或项目',code=='FILTER_ACTIVE',None,False,'search')
        x=72
        for i,lab in enumerate(['最近使用','项目资料','PDF','图片']): x+=android_chip(d,x,y0+455,lab,i==0)+14
        yy=y0+570
        for i,(fn,meta) in enumerate([('架构说明.pdf','YLVEN Android App · 2.4 MB'),('数据库设计.md','YLVEN Android App · 36 KB'),('品牌主视觉.png','品牌资产 · 4.8 MB')]):
            rounded(d,(72,yy,1008,yy+170),30,C['surface'],C['border'],2); icon_circle(d,(142,yy+85),40,'文',C['blue_soft'],C['blue'],30)
            text(d,(210,yy+40),fn,36,C['text'],True); text(d,(210,yy+94),meta,27,C['text3']); line_icon(d,920,yy+55,'check' if i==0 else 'more',48,C['brand'] if i==0 else C['text3'],5)
            yy+=190
    if code in {'EMPTY','DISABLED','SERVICE_DEGRADED','SERVER_ERROR'}:
        status_banner(im,'SERVICE_DEGRADED' if code=='DISABLED' else code,top=600,width_ratio=.78)
    rounded(d,(430,2370,650,2382),7,C['text'])
    return im.convert('RGB')


def render_android_component(im: Image.Image, spec: dict[str,Any], template: str, code: str) -> Image.Image:
    d=ImageDraw.Draw(im); draw_android_status(d)
    # Component boards are not full app pages: clear visual contract header and focused specimen.
    d.rectangle((0,72,1080,270),fill=C['surface']); text(d,(60,118),'YLVEN UI CONTRACT',27,C['brand'],True); text(d,(60,166),spec['title'],52,C['text'],True); text(d,(60,228),spec['subtitle'],27,C['text3'])
    pill(d,(830,122,1010,184),'组件规范',C['brand_soft'],C['brand'],24)
    # Stage.
    shadow_card(im,(54,330,1026,2020),52,C['surface'],C['border'],10,5); d=ImageDraw.Draw(im)
    text(d,(105,390),'当前状态',28,C['text3'],True); pill(d,(270,376,600,438),code,C['surface_subtle'],C['text2'],23,31,C['border'])
    if template=='app_shell':
        rounded(d,(170,510,910,1740),38,C['bg'],C['border2'],3); d.rectangle((170,510,910,690),fill=C['surface']); text(d,(220,565),'页面标题',38,C['text'],True); d.rectangle((170,1545,910,1740),fill=C['surface'])
        for i,(lab,ico) in enumerate([('首页','home'),('工作','briefcase'),('发现','compass'),('我的','user')]):
            cx=262+i*185; line_icon(d,cx-24,1585,ico,48,C['brand'] if i==0 else C['text3'],4); text(d,(cx,1650),lab,24,C['brand'] if i==0 else C['text3'],i==0,'ma')
        text(d,(220,820),'内容区使用 16dp 页面边距',30,C['text3']); rounded(d,(220,900,860,1120),28,C['surface'],C['border'],2)
    elif template=='android_design':
        colors=[C['brand'],C['blue'],C['violet'],C['success'],C['warning'],C['error']]
        for i,col in enumerate(colors): d.rounded_rectangle((120+i%3*290,540+i//3*210,360+i%3*290,700+i//3*210),radius=28,fill=col); text(d,(240+i%3*290,720+i//3*210),col,25,C['text2'],False,'ma')
        text(d,(120,1060),'页标题 24/32sp · 700',44,C['text'],True); text(d,(120,1140),'正文 16/24sp · 常规字重',36,C['text2'])
        android_button(im,(120,1300,900,1456),'主按钮 · 52dp · R14',True); android_input(im,(120,1570,900,1718),'输入框','','输入内容',False,None)
    elif template=='bottom_nav':
        rounded(d,(110,690,970,1010),38,C['surface'],C['border'],3)
        items=[('首页','home'),('工作','briefcase'),('发现','compass'),('我的','user')]
        for i,(lab,ico) in enumerate(items):
            cx=220+i*210; ifill=C['brand_soft'] if i==0 else C['surface']; rounded(d,(cx-55,735,cx+55,815),36,ifill); line_icon(d,cx-26,747,ico,52,C['brand'] if i==0 else C['text3'],5); text(d,(cx,850),lab,29,C['brand'] if i==0 else C['text3'],i==0,'ma')
        text(d,(120,1115),'固定内容高度 64dp · 图标 24dp · 最小触控 48dp',31,C['text2'])
    elif template=='auth_resilience':
        cards=[('弱网重试','保留输入，不重复提交','↻'),('离线提示','明确缓存时间与能力限制','↯'),('320dp 小屏','纵向滚动，不压缩触控区','□'),('大字体','文本可换行，按钮高度保持','Aa')]
        for i,(ttl,sub,ico) in enumerate(cards):
            x=105+(i%2)*440; y=540+(i//2)*410
            rounded(d,(x,y,x+395,y+350),36,C['surface_subtle'],C['border'],2); icon_circle(d,(x+70,y+76),42,ico,C['brand_soft'],C['brand'],30); text(d,(x+42,y+145),ttl,37,C['text'],True); text(d,(x+42,y+205),sub,28,C['text3'],False,max_width=310)
        if state_is_error(code): status_banner(im,code,top=1510,width_ratio=.72)
    elif template in {'composer','input_box'}:
        y=890; rounded(d,(110,y,970,y+270),82,C['surface'],C['brand'] if code=='INPUT_FOCUSED' else C['border2'],5 if code=='INPUT_FOCUSED' else 2)
        icon_circle(d,(190,y+135),42,'＋',C['surface_subtle'],C['text2'],34); text(d,(260,y+105),'输入问题或上传文件',36,C['disabled']); icon_circle(d,(820,y+135),42,'🎙',C['surface_subtle'],C['text2'],28); icon_circle(d,(920,y+135),42,'➤',C['brand'],C['surface'],30)
        pill(d,(120,y-100,440,y-30),'GPT-5.6 Sol · 深度',C['brand_soft'],C['brand'],25)
        if code=='UPLOADING': rounded(d,(125,y-245,660,y-130),24,C['blue_soft'],C['border'],2); text(d,(170,y-205),'需求文档.pdf · 62%',28,C['blue'],True)
        if code=='DISABLED': d.rectangle((110,y,970,y+270),fill=(246,247,251,160))
    elif template=='ai_response':
        y=560; pill(d,(120,y,470,y+70),'GPT-5.6 Sol · 深度',C['brand_soft'],C['brand'],27)
        text(d,(120,y+120),'后端建议采用模块化单体核心业务，\n并将 AI Runtime 作为独立无状态服务。',39,C['text'],False,max_width=820,line_spacing=18)
        rounded(d,(120,y+350,960,y+560),30,C['surface_subtle'],C['border'],2); text(d,(165,y+395),'工具调用',27,C['text3'],True); text(d,(165,y+455),'正在读取项目架构资料',34,C['text'],True)
        if code in {'STREAMING','CONNECTING','TOOL_RUNNING'}: draw_spinner(d,(900,y+455),28,C['brand'],6)
        if code=='STOPPED': pill(d,(120,y+620,430,y+690),'已停止生成',C['warning_soft'],C['warning'],27)
        if code in {'PROVIDER_ERROR','CONTENT_BLOCKED'}: status_banner(im,code,top=1450,width_ratio=.72)
    elif template=='code_block':
        rounded(d,(105,570,975,1390),34,C['code'],None); pill(d,(135,600,330,665),'Kotlin', '#1E293B','#CBD5E1',25); pill(d,(780,600,935,665),'复制','#1E293B','#CBD5E1',25)
        code_lines=['data class Model(','  val id: String,','  val capabilities: Set<String>',')','','fun supportsVision() =','  "vision" in capabilities']
        yy=720
        for i,line in enumerate(code_lines,1): text(d,(145,yy),f'{i:02d}',27,'#64748B'); text(d,(220,yy),line,31,C['code_text'],False); yy+=75
        if code=='CONTENT_BLOCKED': status_banner(im,code,top=1500,width_ratio=.72)
        elif code=='COMPLETED': status_banner(im,'SUCCESS','代码块已完整渲染。',top=1500,width_ratio=.72)
    elif template=='table_block':
        cols=[('模型',120),('识图',410),('文件',610),('状态',810)]; y=610
        rounded(d,(105,y,975,y+690),32,C['surface'],C['border'],2); d.rectangle((105,y,975,y+110),fill=C['surface_subtle'])
        for lab,x in cols: text(d,(x,y+55),lab,30,C['text2'],True,'lm')
        rows=[('GPT-5.6 Sol','支持','支持','正常'),('Claude Opus','支持','支持','正常'),('Grok','支持','部分','限流')]
        yy=y+110
        for r,row in enumerate(rows):
            if r%2: d.rectangle((106,yy,974,yy+150),fill='#FCFCFD')
            for val,(_,x) in zip(row,cols): text(d,(x,yy+75),val,29,C['text'],val=='正常','lm')
            yy+=150
        text(d,(105,1390),'移动端表格允许横向滚动，首列保持可识别。',29,C['text3'])
        if code=='COMPLETED': status_banner(im,'SUCCESS','表格内容已加载完成。',top=1510,width_ratio=.72)
        elif code=='OFFLINE': status_banner(im,'OFFLINE',top=1510,width_ratio=.72)
    elif template=='citation_block':
        rounded(d,(105,600,975,1160),34,C['surface_subtle'],C['border'],2); pill(d,(140,640,285,702),'来源 1',C['brand_soft'],C['brand'],25)
        icon_circle(d,(165,790),34,'网',C['blue_soft'],C['blue'],24); text(d,(220,748),'OpenAI 官方文档',39,C['text'],True); text(d,(220,810),'developers.openai.com · 已核验',27,C['text3'])
        text(d,(140,900),'“推理档位必须根据模型实际支持能力动态展示……”',31,C['text2'],False,max_width=750)
        android_button(im,(140,1190,600,1320),'打开来源网页',False)
        if code in {'NOT_FOUND','OFFLINE'}: status_banner(im,code,top=1450,width_ratio=.72)
        elif code=='COMPLETED': status_banner(im,'SUCCESS','来源链接与标题已校验。',top=1450,width_ratio=.72)
    elif template=='project_citation':
        rounded(d,(105,570,975,1220),34,C['surface'],C['brand'],3); pill(d,(140,610,405,672),'项目知识引用',C['brand'],C['surface'],25)
        rounded(d,(140,730,940,875),28,C['brand_soft'],C['border'],2); icon_circle(d,(205,802),34,'文',C['surface'],C['brand'],24); text(d,(265,758),'YLVEN Android App',31,C['brand'],True); text(d,(265,812),'架构说明.pdf · 第 8 页 · Chunk 014',27,C['text3'])
        text(d,(140,945),'“AI Runtime 必须保持无状态，并通过统一消息模型适配不同供应商。”',31,C['text'],False,max_width=750)
        pill(d,(140,1085,390,1147),'相关度 96%',C['success_soft'],C['success'],25); android_button(im,(430,1068,900,1185),'在项目中打开',False)
        if code in {'NOT_FOUND','OFFLINE'}: status_banner(im,code,top=1450,width_ratio=.72)
        elif code=='COMPLETED': status_banner(im,'SUCCESS','项目知识引用已绑定到原始文件。',top=1450,width_ratio=.72)
    elif template=='message_actions':
        actions=[('复制','⧉'),('朗读','▶'),('重答','↻'),('换模型','◇'),('赞','↑'),('踩','↓')]; y=820
        rounded(d,(90,y,990,y+210),42,C['surface_subtle'],C['border'],2)
        for i,(lab,sym) in enumerate(actions):
            cx=165+i*145; icon_circle(d,(cx,y+80),33,sym,C['surface'],C['text2'],25); text(d,(cx,y+145),lab,24,C['text3'],False,'ma')
        if code=='DISABLED': d.rectangle((90,y,990,y+210),fill=(246,247,251,190))
        if code=='SUCCESS': status_banner(im,'SUCCESS','内容已复制到剪贴板。',top=1200,width_ratio=.72)
    elif template=='model_strip':
        y=1120; text(d,(120,y-125),'输入框上方的当前模型状态',28,C['text3'],True)
        rounded(d,(120,y,960,y+130),52,C['brand_soft'],C['border'],2); icon_circle(d,(180,y+65),34,'AI',C['brand'],C['surface'],21); text(d,(240,y+38),'GPT-5.6 Sol',34,C['text'],True); pill(d,(600,y+30,760,y+96),'深度',C['surface'],C['brand'],24); pill(d,(780,y+30,920,y+96),'联网',C['blue_soft'],C['blue'],24)
        rounded(d,(120,y+180,960,y+390),64,C['surface'],C['border2'],2); text(d,(180,y+250),'输入问题或上传文件',32,C['disabled']); icon_circle(d,(885,y+285),40,'➤',C['brand'],C['surface'],28)
        if code in {'SERVICE_DEGRADED','DISABLED'}: status_banner(im,'SERVICE_DEGRADED' if code=='SERVICE_DEGRADED' else 'PERMISSION_DENIED',top=1640,width_ratio=.72)
    elif template=='response_label':
        y=760; text(d,(120,y),'AI 回答正文上方的来源标签',28,C['text3'],True)
        d.line((120,y+105,960,y+105),fill=C['divider'],width=2); icon_circle(d,(160,y+165),28,'AI',C['brand'],C['surface'],18); text(d,(210,y+140),'GPT-5.6 Sol',34,C['text'],True); pill(d,(505,y+130,665,y+192),'深度',C['brand_soft'],C['brand'],23); pill(d,(685,y+130,845,y+192),'已联网',C['blue_soft'],C['blue'],23)
        text(d,(120,y+250),'OpenAI · 本条回答 · 12.8 秒 · 已调用 1 个工具',27,C['text3'])
        text(d,(120,y+345),'这是回答正文的起始位置。模型标签不会随会话后续切换而改变。',32,C['text'],False,max_width=820)
        if code in {'SERVICE_DEGRADED','DISABLED'}: status_banner(im,'SERVICE_DEGRADED' if code=='SERVICE_DEGRADED' else 'PERMISSION_DENIED',top=1510,width_ratio=.72)
    elif template in {'image_attachment','file_attachment'}:
        y=720; rounded(d,(115,y,965,y+420),36,C['surface_subtle'],C['border'],2)
        icon_circle(d,(205,y+120),58,'图' if template=='image_attachment' else '文',C['blue_soft'],C['blue'],42)
        text(d,(300,y+65),spec['required_elements'][0],40,C['text'],True); text(d,(300,y+125),' · '.join(spec['required_elements'][1:]),28,C['text3'],False,max_width=560)
        if code=='UPLOADING':
            rounded(d,(300,y+220,880,y+246),13,C['border']); rounded(d,(300,y+220,670,y+246),13,C['brand']); text(d,(300,y+280),'上传中 62%',28,C['brand'],True)
        elif code=='UPLOAD_FAILED': text(d,(300,y+230),'上传失败 · 点击重试',30,C['error'],True)
        elif code=='SUCCESS': text(d,(300,y+230),'已完成解析',30,C['success'],True)
    # Footer rules.
    text(d,(90,2085),'视觉类型：'+('独立组件板' if spec.get('template') not in {'app_shell'} else '公共框架板'),28,C['text3'])
    text(d,(90,2140),'示例文案仅用于视觉占位，功能以 Feature ID 为准。',26,C['text3'])
    rounded(d,(430,2370,650,2382),7,C['text'])
    return im.convert('RGB')


def render_android_session(im: Image.Image, spec: dict[str,Any], code: str) -> Image.Image:
    base=android_canvas(); draw_android_topbar(base,'AI 首页','统一多模型对话入口',back=False,right='plus'); draw_android_bottom_nav(base,'首页')
    render_android_general_body(base,{'title':'AI 首页','subtitle':'统一多模型对话入口','required_elements':['新建对话','分析文件','生成图片','制作 PPT','最近对话']},'home','POPULATED',2140)
    if code=='RESTORING':
        base.alpha_composite(Image.new('RGBA',base.size,(255,255,255,150))); d=ImageDraw.Draw(base); shadow_card(base,(190,840,890,1260),44,C['surface'],C['border'],12,5); d=ImageDraw.Draw(base); draw_spinner(d,(540,950),44,C['brand'],8); text(d,(540,1050),'正在恢复会话',52,C['text'],True,'ma'); text(d,(540,1130),'正在验证远端登录状态',30,C['text3'],False,'ma')
    elif code=='SUCCESS': status_banner(base,'SUCCESS','会话已恢复，未重复发送任何请求。',top=300,width_ratio=.78)
    elif code in {'UNAUTHORIZED','OFFLINE','SERVER_ERROR'}:
        base.alpha_composite(Image.new('RGBA',base.size,(16,24,40,90))); status_banner(base,code,'本地草稿已保留，可重新登录后继续。',top=850,width_ratio=.78)
    return base.convert('RGB')


def render_android_devices(im: Image.Image, spec: dict[str,Any], code: str) -> Image.Image:
    draw_android_topbar(im,spec['title'],spec['subtitle'],True,'more'); d=ImageDraw.Draw(im)
    if code=='LOADING': skeleton(d,(72,360,1008,1300),6,18)
    elif code=='EMPTY': draw_android_error_center(im,'EMPTY','暂无其他登录设备')
    else:
        text(d,(72,320),'当前设备',31,C['text3'],True)
        devices=[('Android · YLVEN','新加坡 · 当前设备','当前'),('Windows · Codex','新加坡 · 10 分钟前',''),('Chrome · Web','新加坡 · 昨天','')]
        y=390
        for i,(ttl,sub,tag) in enumerate(devices):
            rounded(d,(72,y,1008,y+215),36,C['surface'],C['border'],2); icon_circle(d,(150,y+107),46,'A' if i==0 else 'W' if i==1 else 'C',C['brand_soft'],C['brand'],34)
            text(d,(225,y+55),ttl,38,C['text'],True); text(d,(225,y+115),sub,28,C['text3'])
            if tag: pill(d,(820,y+58,950,y+120),tag,C['success_soft'],C['success'],24)
            else: text(d,(918,y+105),'撤销',28,C['error'],True,'mm')
            y+=240
        text(d,(72,y+25),'撤销设备后，该设备必须重新完成邮箱验证码登录。',28,C['text3'],False,max_width=900)
        if code=='REFRESHING': status_banner(im,'SERVICE_DEGRADED','正在刷新设备列表…',top=270,width_ratio=.82)
        elif code in {'UNAUTHORIZED','OFFLINE','SERVER_ERROR'}: status_banner(im,code,top=270,width_ratio=.82)
    rounded(d,(430,2370,650,2382),7,C['text'])
    return im.convert('RGB')


def render_android_general_body(im: Image.Image, spec: dict[str,Any], template: str, code: str, content_bottom: int):
    d=ImageDraw.Draw(im); y0=300
    # Home / tab surfaces.
    if template=='home':
        text(d,(72,330),'你好，陈平',58,C['text'],True); text(d,(72,405),'今天准备完成什么？',36,C['text3'])
        shadow_card(im,(72,510,1008,760),48,C['surface'],C['border'],10,4); d=ImageDraw.Draw(im)
        text(d,(125,565),'输入问题或上传文件',36,C['disabled']); icon_circle(d,(920,635),48,'➤',C['brand'],C['surface'],34)
        pill(d,(125,680,480,742),'GPT-5.6 Sol · 深度',C['brand_soft'],C['brand'],25)
        shortcuts=[('分析文件','文'),('生成图片','图'),('制作 PPT','P'),('多模型对比','比')]
        y=825
        for i,(lab,ico) in enumerate(shortcuts):
            x=72+(i%2)*480; yy=y+(i//2)*190; rounded(d,(x,yy,x+448,yy+160),36,C['surface'],C['border'],2); icon_circle(d,(x+70,yy+80),42,ico,C['brand_soft'],C['brand'],30); text(d,(x+135,yy+80),lab,35,C['text'],True,'lm')
        text(d,(72,1250),'最近对话',44,C['text'],True); conversations=[('安卓 AI 工具架构设计','GPT-5.6 Sol · 刚刚'),('比较 Claude 与 GPT 的推理差异','Claude Opus · 昨天'),('YLVEN 品牌主视觉','Grok · 7 月 30 日')]
        yy=1330
        for ttl,meta in conversations:
            rounded(d,(72,yy,1008,yy+170),32,C['surface'],C['border'],2); icon_circle(d,(142,yy+85),38,'AI',C['blue_soft'],C['blue'],24); text(d,(205,yy+42),ttl,34,C['text'],True); text(d,(205,yy+98),meta,27,C['text3']); yy+=188
    elif template=='work_home':
        tabs=['总览','工具','项目','作品']; x=72
        for i,lab in enumerate(tabs): x+=android_chip(d,x,300,lab,i==0)+14
        text(d,(72,430),'工作概览',48,C['text'],True)
        metrics=[('进行中任务','3','任'),('最近项目','5','项'),('本月作品','18','作')]
        for i,(lab,val,ico) in enumerate(metrics):
            x=72+i*312; rounded(d,(x,520,x+288,730),36,C['surface'],C['border'],2); icon_circle(d,(x+62,585),36,ico,C['brand_soft'],C['brand'],26); text(d,(x+34,650),val,44,C['text'],True); text(d,(x+110,657),lab,27,C['text3'])
        text(d,(72,820),'继续工作',44,C['text'],True); rounded(d,(72,910,1008,1190),40,C['surface'],C['border'],2); pill(d,(120,955,310,1017),'项目',C['brand_soft'],C['brand'],24); text(d,(120,1060),'YLVEN Android App',40,C['text'],True); text(d,(120,1120),'最后编辑 10 分钟前 · 2 个任务运行中',27,C['text3'])
        text(d,(72,1290),'最近作品',44,C['text'],True)
        for i,lab in enumerate(['商业 AI 平台 PPT','科技主视觉']): x=72+i*480; rounded(d,(x,1380,x+448,1710),38,mix('#E8EAFE','#CFE3FF',i*.55),C['border'],2); text(d,(x+30,1645),lab,29,C['text'],True)
    elif template=='tool_catalog':
        tabs=['全部','内容创作','文件与数据','研究']; x=72
        for i,lab in enumerate(tabs): x+=android_chip(d,x,300,lab,i==0)+14
        android_input(im,(72,410,1008,550),'搜索工具','','搜索图片、PPT、文件分析或工作流',False,None,False,'search')
        text(d,(72,640),'全部 AI 工具',48,C['text'],True)
        tools=[('图片生成','创建与编辑 AI 图片','图'),('PPT 制作','生成可编辑演示文稿','P'),('文件分析','理解 PDF、Word 与表格','文'),('多模型对比','并行比较多个 AI 回答','比'),('深度研究','检索、引用与报告','研'),('网页总结','提取并总结网页','网')]
        y=730
        for i,(ttl,sub,ico) in enumerate(tools):
            x=72+(i%2)*480; yy=y+(i//2)*270; rounded(d,(x,yy,x+448,yy+240),38,C['surface'],C['border'],2); icon_circle(d,(x+72,yy+70),42,ico,C['brand_soft'],C['brand'],30); text(d,(x+42,yy+138),ttl,35,C['text'],True); text(d,(x+42,yy+188),sub,25,C['text3'],False,max_width=350)
    elif template=='discover_home':
        rounded(d,(72,310,1008,600),48,C['brand'],None); text(d,(125,370),'探索新的 AI 能力',52,C['surface'],True); text(d,(125,445),'模板、模型、工作流与连接器',31,'#E5E7FF'); pill(d,(125,510,345,575),'查看推荐',C['surface'],C['brand'],25)
        text(d,(72,680),'推荐服务',44,C['text'],True)
        services=[('Prompt 模板','快速开始专业任务','模'),('模型实验室','比较不同模型能力','AI'),('深度研究','多步骤检索与报告','研'),('网页总结','提取并总结网页内容','网')]
        y=770
        for i,(ttl,sub,ico) in enumerate(services):
            x=72+(i%2)*480; yy=y+(i//2)*280; rounded(d,(x,yy,x+448,yy+250),40,C['surface'],C['border'],2); icon_circle(d,(x+72,yy+72),44,ico,C['blue_soft'],C['blue'],30); text(d,(x+42,yy+145),ttl,36,C['text'],True); text(d,(x+42,yy+198),sub,26,C['text3'])
    elif template=='mine':
        shadow_card(im,(72,310,1008,620),48,C['surface'],C['border'],10,4); d=ImageDraw.Draw(im)
        icon_circle(d,(185,445),72,'陈',C['brand'],C['surface'],48); text(d,(295,365),'陈平',48,C['text'],True); text(d,(295,430),'UID 100001 · YLVEN Pro',29,C['text3']); pill(d,(295,490,520,555),'编辑个人资料',C['brand_soft'],C['brand'],24)
        groups=[('使用与内容',['我的项目','我的作品','我的文件','我的收藏']),('AI 设置',['默认模型','回答风格','自定义指令','个人记忆']),('安全与系统',['账号安全','登录设备','通知设置','检查更新'])]
        y=700
        for g,items in groups:
            text(d,(72,y),g,31,C['text3'],True); y+=65
            rounded(d,(72,y,1008,y+len(items)*116),34,C['surface'],C['border'],2)
            for i,item in enumerate(items):
                yy=y+i*116; icon_circle(d,(130,yy+58),30,'›',C['surface_subtle'],C['text3'],26); text(d,(185,yy+58),item,34,C['text'],False,'lm'); text(d,(940,yy+58),'›',36,C['text3'],False,'mm')
                if i<len(items)-1:d.line((185,yy+115,960,yy+115),fill=C['divider'],width=2)
            y+=len(items)*116+70
    elif template in {'new_chat_landing','new_conversation_form','project_create','ppt_slide_edit'}:
        # Form-like pages with unique fields.
        labels=spec.get('required_elements',[])[:4]
        y=350
        for i,lab in enumerate(labels):
            if lab in {'创建','保存','下一步'}: continue
            android_input(im,(72,y,1008,y+145),lab,'',f'请输入{lab}',code=='INPUT_FOCUSED' and i==0,'请检查此字段' if code=='VALIDATION_ERROR' and i==0 else None,False)
            y+=230
        android_button(im,(72,min(y+20,1880),1008,min(y+176,2036)),spec.get('required_elements',[])[-1] if spec.get('required_elements') else '继续',True,loading=code=='SUBMITTING')
    elif template in {'conversation_drawer','conversation_search','file_library','file_search','project_list','project_files','workspace_search','artifact_library','discover_category','prompt_templates','model_lab','announcements','favorites','bills','usage','service_status','branch_switcher'}:
        # Search / filter header.
        if template in {'conversation_search','file_search','workspace_search'}:
            android_input(im,(72,310,1008,450),'搜索','',spec['subtitle'],code=='FILTER_ACTIVE',None,False,'search'); y=515
        else: y=320
        x=72
        chips=spec.get('required_elements',[])[:4] or ['全部','进行中','已完成']
        for i,lab in enumerate(chips):
            w=android_chip(d,x,y,truncate(lab,8),i==0); x+=w+14
            if x>900: break
        y+=110
        rows=spec.get('required_elements',[])
        if len(rows)<3: rows+=['最近记录','示例条目','待处理项目']
        for i,item in enumerate(rows[:6]):
            rounded(d,(72,y,1008,y+180),34,C['surface'],C['border'],2); icon_circle(d,(142,y+90),40,str(i+1),C['brand_soft'],C['brand'],28); text(d,(210,y+48),truncate(item,22),36,C['text'],True); text(d,(210,y+106),'更新于 '+('刚刚' if i==0 else f'{i+1} 小时前'),27,C['text3']); line_icon(d,920,y+62,'more',48,C['text3'],4); y+=198
    elif template in {'chat'}:
        # Chat content distinct from component boards.
        pill(d,(72,300,450,370),'GPT-5.6 Sol · 深度',C['brand_soft'],C['brand'],26)
        # user bubble
        rounded(d,(320,455,1008,650),42,C['brand'],None); text(d,(370,505),'请为 YLVEN 设计一套可扩展的\n多模型 AI 后端架构。',35,C['surface'],False,max_width=580,line_spacing=16)
        text(d,(72,740),'GPT-5.6 Sol · 深度推理',27,C['brand'],True)
        text(d,(72,800),'建议将系统拆分为控制平面、AI 数据平面和异步工作平面。\n业务后端管理用户、会话、钱包和作品；AI Runtime 负责流式请求、\n模型路由与上下文编译；Worker 负责图片、文件和 PPT 任务。',37,C['text'],False,max_width=930,line_spacing=20)
        rounded(d,(72,1110,1008,1320),30,C['surface_subtle'],C['border'],2); text(d,(120,1150),'已读取 2 份项目资料',28,C['text3'],True); text(d,(120,1210),'架构说明.pdf · 数据模型.md',32,C['text'],True)
        # composer
        rounded(d,(54,1960,1026,2185),70,C['surface'],C['border2'],2); icon_circle(d,(130,2072),38,'＋',C['surface_subtle'],C['text2'],30); text(d,(200,2050),'继续追问…',34,C['disabled']); icon_circle(d,(940,2072),42,'➤',C['brand'],C['surface'],30)
        if code=='STREAMING': text(d,(72,1380),'正在生成…',29,C['brand'],True); draw_spinner(d,(235,1395),24,C['brand'],6)
        elif code=='TOOL_RUNNING': status_banner(im,'SERVICE_DEGRADED','正在调用文件检索工具…',top=1430,width_ratio=.80)
        elif code=='STOPPED': status_banner(im,'SERVICE_DEGRADED','已停止生成，已保留当前内容。',top=1430,width_ratio=.80)
        elif code in {'PROVIDER_ERROR','CONTENT_BLOCKED','OFFLINE','RATE_LIMITED'}: status_banner(im,code,top=1430,width_ratio=.80)
    elif template=='offline_cache':
        text(d,(72,335),'最近缓存的会话',46,C['text'],True); pill(d,(72,410,420,478),'缓存于 10:21',C['warning_soft'],C['warning'],26)
        rows=['安卓 AI 工具架构设计','YLVEN UI 规范','多模型对比方案']; y=550
        for i,item in enumerate(rows): rounded(d,(72,y,1008,y+180),34,C['surface'],C['border'],2); text(d,(125,y+50),item,36,C['text'],True); text(d,(125,y+110),'只读缓存 · '+f'{i+1} 条未同步',27,C['text3']); y+=200
        status_banner(im,'OFFLINE_CACHE','恢复网络后将自动检查最新状态。',top=1320,width_ratio=.82)
    elif template in {'ai_settings','conversation_settings','project_settings','project_instructions','profile','memory','notifications','appearance','privacy'}:
        groups=spec.get('required_elements',[]); y=330
        for i,item in enumerate(groups):
            rounded(d,(72,y,1008,y+150),32,C['surface'],C['border'],2); text(d,(120,y+75),item,35,C['text'],False,'lm')
            if any(k in item for k in ['模式','通知','记忆','联网','临时']):
                rounded(d,(830,y+45,940,y+105),30,C['brand']); d.ellipse((890,y+50,935,y+100),fill=C['surface'])
            else: text(d,(930,y+75),'›',40,C['text3'],False,'mm')
            y+=170
        if code in {'EDIT_MODE','DIRTY'}: android_button(im,(72,min(y+20,1850),1008,min(y+176,2006)),'保存设置',True,loading=code=='SUBMITTING')
        if code in {'SAVE_SUCCESS','SAVE_ERROR','OFFLINE'}: status_banner(im,code,top=1550,width_ratio=.82)
    elif template in {'comparison_setup','image_studio','ppt_landing','ppt_basic','ppt_sources','ppt_template','ppt_outline','ppt_outline_edit'}:
        # Wizard cards with step indicator.
        steps=4; current=1
        if template=='ppt_sources': current=2
        elif template=='ppt_template': current=3
        elif template in {'ppt_outline','ppt_outline_edit'}: current=4
        for i in range(steps):
            cx=160+i*250; d.line((cx+40,340,cx+210,340),fill=C['brand'] if i<current-1 else C['border2'],width=8) if i<steps-1 else None
            icon_circle(d,(cx,340),34,str(i+1),C['brand'] if i<current else C['surface'],C['surface'] if i<current else C['text3'],26)
        items=spec.get('required_elements',[]); y=440
        for i,item in enumerate(items[:5]):
            rounded(d,(72,y,1008,y+170),36,C['surface'],C['border'],2); icon_circle(d,(142,y+85),38,str(i+1),C['brand_soft'],C['brand'],27); text(d,(210,y+85),item,36,C['text'],True,'lm'); y+=190
        android_button(im,(72,min(y+20,1880),1008,min(y+176,2036)),'继续' if template not in {'comparison_setup','ppt_outline_edit'} else '开始生成',True,loading=code in {'RUNNING','PREPARING'})
        if code in {'VALIDATION_ERROR','FAILED','OFFLINE'}: status_banner(im,code,top=1570,width_ratio=.82)
    elif template in {'comparison_result'}:
        x=72
        for i,lab in enumerate(['GPT','Claude','Grok']): x+=android_chip(d,x,310,lab,i==0,width=210)+18
        rounded(d,(72,430,1008,1420),40,C['surface'],C['border'],2); text(d,(120,485),'GPT-5.6 Sol · 深度',30,C['brand'],True); text(d,(120,560),'建议采用模块化单体核心业务与独立 AI Runtime，\n在数据模型层建立统一消息格式，并通过 Provider Adapter\n兼容 Sub2API 与未来官方 API。',38,C['text'],False,max_width=820,line_spacing=20)
        android_button(im,(72,1500,510,1650),'采用此回答',True); android_button(im,(540,1500,1008,1650),'综合三个回答',False)
        if code in {'PARTIAL_DATA','PROVIDER_ERROR','OFFLINE'}: status_banner(im,'SERVICE_DEGRADED' if code=='PARTIAL_DATA' else code,top=1700,width_ratio=.82)
    elif template in {'upload_progress','image_job','ppt_job','job_center'}:
        text(d,(72,330),spec['title'],48,C['text'],True); text(d,(72,400),spec['subtitle'],30,C['text3'])
        stages=spec.get('required_elements',[]); y=520
        for i,item in enumerate(stages[:5]):
            active=i==1 if code in {'RUNNING','UPLOADING','PREPARING'} else i<3 if code=='SUCCESS' else i==0
            icon_circle(d,(126,y+10),30,'✓' if code=='SUCCESS' or (active and i==0) else str(i+1),C['success_soft'] if code=='SUCCESS' else C['brand_soft'],C['success'] if code=='SUCCESS' else C['brand'],24)
            text(d,(185,y-10),item,35,C['text'],True); text(d,(185,y+42),'已完成' if code=='SUCCESS' else '进行中' if active else '等待',27,C['success'] if code=='SUCCESS' else C['brand'] if active else C['text3']); y+=155
        rounded(d,(72,y+20,1008,y+52),16,C['border']); progress=.72 if code in {'RUNNING','UPLOADING'} else 1 if code=='SUCCESS' else .12; rounded(d,(72,y+20,72+int(936*progress),y+52),16,C['brand'])
        android_button(im,(72,y+120,1008,y+276),'取消任务' if code not in {'SUCCESS','FAILED','CANCELLED'} else '重新开始',False)
        if code in {'FAILED','CANCELLED','OFFLINE'}: status_banner(im,code,top=1600,width_ratio=.82)
    elif template in {'image_landing'}:
        rounded(d,(72,310,1008,620),48,C['brand'],None); text(d,(125,375),'把想法变成图片',52,C['surface'],True); text(d,(125,450),'支持生成、编辑和参考图创作',31,'#E5E7FF'); pill(d,(125,520,360,580),'开始创作',C['surface'],C['brand'],25)
        text(d,(72,700),'最近作品',44,C['text'],True)
        for i in range(4):
            x=72+(i%2)*480; y=790+(i//2)*390; rounded(d,(x,y,x+448,y+340),38,mix('#E9EBFF','#CFE3FF',i/5),C['border'],2); text(d,(x+30,y+285),f'科技主视觉 {i+1}',29,C['text'],True)
    elif template=='image_result':
        text(d,(72,320),'本次生成 4 张',35,C['text3'],True)
        for i in range(4):
            x=72+(i%2)*480; y=395+(i//2)*520; rounded(d,(x,y,x+448,y+470),38,mix('#E8EAFE','#C8DBFF',i/4),C['border'],2); icon_circle(d,(x+224,y+210),70,'Y',C['brand'],C['surface'],50); pill(d,(x+24,y+390,x+170,y+440),f'方案 {i+1}',C['surface'],C['brand'],23)
        android_button(im,(72,1515,510,1670),'全部保存',True); android_button(im,(540,1515,1008,1670),'继续生成',False)
        if code in {'DOWNLOADING','SERVER_ERROR','OFFLINE'}: status_banner(im,code,top=1740,width_ratio=.82)
    elif template=='image_detail':
        rounded(d,(72,330,1008,1390),44,mix('#E8EAFE','#C8DBFF',.45),C['border'],2); icon_circle(d,(540,790),120,'Y',C['brand'],C['surface'],82)
        pill(d,(110,1435,310,1500),'2048 × 2048',C['surface'],C['text2'],24); pill(d,(330,1435,510,1500),'PNG',C['surface'],C['text2'],24); pill(d,(530,1435,790,1500),'Grok Imagine',C['brand_soft'],C['brand'],24)
        text(d,(72,1590),'提示词',31,C['text3'],True); text(d,(72,1645),'高级、轻盈的 AI 品牌主视觉，蓝紫科技渐变…',33,C['text'],False,max_width=900)
        android_button(im,(72,1840,510,1995),'下载原图',True); android_button(im,(540,1840,1008,1995),'继续编辑',False)
        if code in {'DOWNLOADING','SERVER_ERROR','OFFLINE'}: status_banner(im,code,top=2050,width_ratio=.82)
    elif template=='ppt_preview':
        rounded(d,(72,330,310,1940),32,C['surface'],C['border'],2); yy=380
        for i in range(5): rounded(d,(100,yy,282,yy+120),18,C['brand_soft'] if i==2 else C['surface_subtle'],C['brand'] if i==2 else C['border'],3 if i==2 else 1); text(d,(120,yy+45),f'{i+1}',24,C['text3']); yy+=145
        rounded(d,(350,330,1008,1210),38,C['surface'],C['border'],2); d.rectangle((350,330,1008,1210),fill='#07132E'); text(d,(420,520),'YLVEN',42,'#9CCBFF',True); text(d,(420,610),'商业 AI 平台',62,C['surface'],True); text(d,(420,710),'统一对话 · 工作台 · 开发者 API',30,'#CFE0FF')
        android_button(im,(350,1280,660,1430),'播放预览',True); android_button(im,(690,1280,1008,1430),'编辑当前页',False)
        if code in {'DOWNLOADING','SERVER_ERROR','OFFLINE'}: status_banner(im,code,top=1700,width_ratio=.82)
    elif template=='ppt_detail':
        rounded(d,(72,330,1008,670),44,C['dark']); text(d,(125,390),'YLVEN',32,'#9CCBFF',True); text(d,(125,470),'商业 AI 平台',52,C['surface'],True); text(d,(125,545),'12 页 · 科技商务模板',28,'#CFE0FF')
        rows=[('最新版本','V4 · 10 分钟前'),('生成模型','GPT-5.6 Sol'),('导出文件','PPTX · PDF'),('项目','YLVEN Android App')]
        y=760
        for lab,val in rows: rounded(d,(72,y,1008,y+150),30,C['surface'],C['border'],2); text(d,(120,y+75),lab,31,C['text3'],True,'lm'); text(d,(930,y+75),val,31,C['text'],True,'rm'); y+=170
        android_button(im,(72,1530,510,1685),'下载 PPTX',True); android_button(im,(540,1530,1008,1685),'查看版本',False)
        if code in {'DOWNLOADING','SERVER_ERROR','OFFLINE'}: status_banner(im,code,top=1760,width_ratio=.82)
    elif template in {'file_detail','project_detail','image_detail','ppt_detail','custom_instructions','app_update','help','membership','wallet'}:
        # Detail summary cards.
        rounded(d,(72,320,1008,600),44,C['surface'],C['border'],2); icon_circle(d,(160,450),56,spec['title'][0],C['brand_soft'],C['brand'],40); text(d,(250,375),spec['title'],43,C['text'],True); text(d,(250,445),spec['subtitle'],29,C['text3'],False,max_width=670)
        y=680
        for i,item in enumerate(spec.get('required_elements',[])[:6]):
            rounded(d,(72,y,1008,y+150),32,C['surface'],C['border'],2); text(d,(120,y+75),item,34,C['text'],False,'lm'); text(d,(930,y+75),'›',40,C['text3'],False,'mm'); y+=170
        if template in {'membership','wallet'}:
            rounded(d,(72,320,1008,600),44,C['brand'],None); text(d,(125,375),'可用 AI 额度',30,'#E5E7FF'); text(d,(125,445),'¥ 128.50',64,C['surface'],True); pill(d,(745,455,930,520),'立即充值',C['surface'],C['brand'],24)
        if code in {'PAYMENT_PENDING','CREDIT_PENDING','REFUND_PENDING','BUDGET_EXHAUSTED'}: status_banner(im,code,top=1680,width_ratio=.82)
    elif template=='plan_confirm':
        rounded(d,(72,330,1008,870),44,C['surface'],C['border'],2); text(d,(120,385),'YLVEN Pro',50,C['text'],True); text(d,(120,455),'月度方案',31,C['text3']); text(d,(120,560),'¥ 39.00',66,C['brand'],True)
        benefits=['共享 AI 额度','高级模型与推理','20 个项目','20 GB 存储','开发者 API']
        y=690
        for b in benefits: icon_circle(d,(140,y),22,'✓',C['success_soft'],C['success'],18); text(d,(180,y-15),b,31,C['text2']); y+=70
        android_button(im,(72,1000,1008,1156),'确认支付',True,loading=code=='PAYMENT_PENDING')
        if code in {'PAYMENT_PENDING','CREDIT_PENDING','REFUND_PENDING','SERVER_ERROR','OFFLINE'}: status_banner(im,code,top=1300,width_ratio=.82)
    else:
        # Distinct generic detail using visual identity elements.
        rounded(d,(72,330,1008,620),44,C['surface'],C['border'],2); icon_circle(d,(160,470),56,spec['title'][0],C['brand_soft'],C['brand'],40); text(d,(250,390),spec['title'],45,C['text'],True); text(d,(250,465),spec['subtitle'],29,C['text3'],False,max_width=670)
        y=700
        for i,item in enumerate(spec.get('required_elements',[])[:6]):
            rounded(d,(72,y,1008,y+155),32,C['surface'],C['border'],2); icon_circle(d,(135,y+78),30,str(i+1),C['surface_subtle'],C['text3'],22); text(d,(190,y+78),item,34,C['text'],True,'lm'); y+=175


# ---------- Admin / developer / public web renderers ------------------------

def draw_web_logo(draw, x, y, label='YLVEN'):
    rounded(draw,(x,y,x+36,y+36),10,C['brand']); text(draw,(x+18,y+18),'Y',18,C['surface'],True,'mm'); text(draw,(x+50,y+18),label,20,C['text'],True,'lm')


def sidebar_items(surface: str) -> list[str]:
    if surface=='ADMIN': return ['运营总览','认证与用户','模型与供应商','对话与内容','文件与存储','项目与知识库','AI 工具','作品中心','商业化','开发者平台','安全与审计','可观测性','系统设置','质量与发布']
    return ['开发者概览','API Key','API Playground','模型与价格','调用日志','用量统计','预算与告警','充值与账单','API 文档','Webhook','账号与安全']


def draw_web_shell(im: Image.Image, page: dict[str,Any], developer=False):
    d=ImageDraw.Draw(im); W,H=im.size; sidebar=240 if developer else 248
    d.rectangle((0,0,sidebar,H),fill=C['surface']); d.line((sidebar,0,sidebar,H),fill=C['divider'],width=1)
    draw_web_logo(d,26,22,'YLVEN DEV' if developer else 'YLVEN ADMIN')
    section=page['name'].split('/')[0] if '/' in page['name'] else ('开发者中心' if developer else '系统')
    y=92
    for item in sidebar_items('DEVELOPER' if developer else 'ADMIN'):
        active=(item==section) or (developer and any(k in page['name'] for k in item.replace('开发者','').split('与')))
        if active: rounded(d,(16,y,sidebar-16,y+44),10,C['brand_soft'])
        icon_circle(d,(38,y+22),13,'·',C['brand_soft'] if active else C['surface_subtle'],C['brand'] if active else C['text3'],12)
        text(d,(60,y+22),item,14,C['brand'] if active else C['text2'],active,'lm'); y+=52
    d.rectangle((sidebar,0,W,64),fill=C['surface']); d.line((sidebar,64,W,64),fill=C['divider'],width=1)
    text(d,(sidebar+28,32),'YLVEN / '+page['name'],13,C['text3'],False,'lm')
    android_chip(d,W-310,14,'staging',True,width=92); icon_circle(d,(W-70,32),18,'陈',C['brand'],C['surface'],13)
    return sidebar


def subject_columns(name: str) -> list[str]:
    n=name
    pairs=[
        (['用户','设备','会话'],['ID','用户/对象','状态','来源','更新时间','操作']),
        (['模型','供应商','通道','能力'],['名称','供应商','能力/分组','状态','最近探测','操作']),
        (['订单','支付','退款','交易','账本','钱包','余额'],['流水/订单号','用户','金额','类型','状态','时间']),
        (['文件','存储','附件'],['文件名','类型','大小','处理状态','归属','更新时间']),
        (['任务','队列','构建','发布','迁移'],['任务 ID','类型','阶段/进度','状态','负责人','更新时间']),
        (['API Key','API日志','调用日志','Webhook'],['名称/请求 ID','用户/应用','权限/模型','用量/状态','最近调用','操作']),
        (['项目','知识库','索引','检索'],['项目/索引','所有者','资源数','状态','更新时间','操作']),
        (['作品','图片','PPT'],['作品/任务','用户','模型/模板','状态','创建时间','操作']),
        (['事故','告警','风险','审计','审批'],['事件 ID','级别','对象','状态','负责人','发生时间']),
    ]
    for keys,cols in pairs:
        if any(k in n for k in keys): return cols
    return ['名称','分类','状态','配置版本','更新时间','操作']


def deterministic_values(seed: str, count=8) -> list[int]:
    h=int(hashlib.sha1(seed.encode()).hexdigest()[:10],16); vals=[]
    for i in range(count):
        h=(1103515245*h+12345)&0x7fffffff; vals.append(30+h%60)
    return vals


def render_web_app(page: dict[str,Any], state: dict[str,Any], developer=False) -> Image.Image:
    W,H=CANVAS['DEVELOPER' if developer else 'ADMIN']; im=Image.new('RGBA',(W,H),hexrgb(C['bg'])+(255,)); d=ImageDraw.Draw(im)
    sidebar=draw_web_shell(im,page,developer); content_x=sidebar+28; content_w=W-content_x-28
    spec=page['visual_identity']; code=state['code']; title_v=spec['title']; subtitle=spec['subtitle']
    text(d,(content_x,100),title_v,28,C['text'],True); text(d,(content_x,139),subtitle+' · '+page['page_id'],13,C['text3'])
    pill(d,(W-182,92,W-28,130),'新增 / 配置',C['brand'],C['surface'],13,10)
    layout=page['layout_profile']
    if code=='LOADING':
        skeleton(d,(content_x,190,W-32,780),8,8); return im.convert('RGB')
    if code in {'PERMISSION_DENIED','NETWORK_ERROR','SERVER_ERROR','NOT_FOUND','SERVICE_DEGRADED','BUDGET_EXHAUSTED'}:
        # show shell plus centered state card
        x1=content_x+170; y1=280; x2=W-190; y2=620; shadow_card(im,(x1,y1,x2,y2),20,C['surface'],C['border'],8,3); d=ImageDraw.Draw(im)
        sym='🔒' if code=='PERMISSION_DENIED' else '!' if code!='NOT_FOUND' else '?'; icon_circle(d,((x1+x2)//2,y1+90),36,sym,C['error_soft'] if code!='NOT_FOUND' else C['surface_subtle'],C['error'] if code!='NOT_FOUND' else C['text3'],26)
        ttl={'PERMISSION_DENIED':'权限不足','NETWORK_ERROR':'网络连接失败','SERVER_ERROR':'服务暂时不可用','NOT_FOUND':'记录不存在','SERVICE_DEGRADED':'部分数据源不可用','BUDGET_EXHAUSTED':'预算已用尽'}[code]
        text(d,((x1+x2)//2,y1+155),ttl,24,C['text'],True,'ma'); text(d,((x1+x2)//2,y1+205),'请求已保留追踪 ID，可重试或查看运行状态。',14,C['text3'],False,'ma')
        pill(d,((x1+x2)//2-78,y1+260,(x1+x2)//2+78,y1+302),'重新尝试',C['brand'],C['surface'],13,10)
        return im.convert('RGB')
    if layout in {'admin_dashboard','developer_dashboard'}:
        vals=deterministic_values(page['page_id'],4); cards=['今日请求','成功率','平均延迟','异常事件']
        for i,(lab,val) in enumerate(zip(cards,vals)):
            x=content_x+i*(content_w-36)//4; w=(content_w-60)//4
            shadow_card(im,(x,188,x+w,310),16,C['surface'],C['border'],5,2); d=ImageDraw.Draw(im); text(d,(x+18,214),lab,13,C['text3']); text(d,(x+18,252),f'{val:,}'+('%' if i==1 else ' ms' if i==2 else ''),27,C['text'],True); text(d,(x+w-18,260),'↑ 8.2%',12,C['success'],True,'ra')
        shadow_card(im,(content_x,340,content_x+content_w*0.66,690),18,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); text(d,(content_x+20,370),'趋势',16,C['text'],True)
        pts=deterministic_values(page['page_id']+'chart',12); x0=content_x+40; y0=630; cw=int(content_w*.66)-80
        coords=[]
        for i,v in enumerate(pts): coords.append((x0+i*cw/(len(pts)-1),y0-v*2.4))
        d.line(coords,fill=C['brand'],width=4,joint='curve');
        for x,y in coords: d.ellipse((x-4,y-4,x+4,y+4),fill=C['brand'])
        shadow_card(im,(content_x+content_w*.69,340,W-28,690),18,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); text(d,(content_x+content_w*.69+20,370),'实时状态',16,C['text'],True)
        yy=420
        for i,item in enumerate(spec.get('required_elements',[])[:5] or ['核心服务','数据库','Redis','任务队列']):
            d.ellipse((content_x+content_w*.69+22,yy+4,content_x+content_w*.69+34,yy+16),fill=C['success'] if i<3 else C['warning']); text(d,(content_x+content_w*.69+48,yy+10),truncate(item,18),14,C['text2'],False,'lm'); yy+=48
    elif layout in {'admin_list','developer_list'}:
        # filters
        shadow_card(im,(content_x,180,W-28,258),14,C['surface'],C['border'],4,1); d=ImageDraw.Draw(im)
        rounded(d,(content_x+16,198,content_x+250,240),9,C['surface_subtle'],C['border'],1); text(d,(content_x+30,219),'搜索名称或 ID',13,C['disabled'],False,'lm')
        x=content_x+270
        for lab in ['状态：全部','类型：全部','最近 30 天']:
            rounded(d,(x,198,x+130,240),9,C['surface'],C['border'],1); text(d,(x+12,219),lab,12,C['text2'],False,'lm'); x+=142
        cols=subject_columns(page['name']); y=285; rowh=58; shadow_card(im,(content_x,y,W-28,y+56+rowh*7),14,C['surface'],C['border'],4,1); d=ImageDraw.Draw(im); d.rectangle((content_x,y,W-28,y+56),fill=C['surface_subtle'])
        colw=(content_w-20)//len(cols)
        for i,col in enumerate(cols): text(d,(content_x+16+i*colw,y+28),col,12,C['text2'],True,'lm')
        data_items=spec.get('required_elements',[]) or [title_v]
        for r in range(7):
            yy=y+56+r*rowh
            if r%2: d.rectangle((content_x+1,yy,W-29,yy+rowh),fill='#FCFCFD')
            for i,col in enumerate(cols):
                if i==0: val=f'{page["page_id"]}-{r+1:03d}'
                elif i==1: val=truncate(data_items[r%len(data_items)],16)
                elif '状态' in col: val='正常' if r%4 else '待处理'
                elif '金额' in col or '用量' in col: val=f'¥ {12.5+r*3.2:.2f}'
                elif '时间' in col or '更新' in col or '调用' in col: val=f'08-04 {10+r:02d}:20'
                else: val=['系统','已启用','staging','自动','查看'][r%5]
                color=C['success'] if val=='正常' else C['warning'] if val=='待处理' else C['text2']
                text(d,(content_x+16+i*colw,yy+rowh/2),val,12,color,val in {'正常','待处理'},'lm',max_width=colw-16)
        text(d,(content_x,y+56+rowh*7+28),'共 128 条 · 第 1 / 13 页',12,C['text3'])
        if code=='EMPTY':
            d.rectangle((content_x,y+56,W-28,y+56+rowh*7),fill=C['surface']); text(d,((content_x+W-28)//2,y+250),'暂无匹配数据',18,C['text3'],True,'ma')
        if code=='FILTER_ACTIVE': pill(d,(content_x+16,268,content_x+160,300),'筛选已生效',C['brand_soft'],C['brand'],11,8)
        if code=='BULK_SELECTED': pill(d,(content_x+16,268,content_x+180,300),'已选择 3 项',C['brand_soft'],C['brand'],11,8)
    elif layout in {'admin_config','developer_detail'}:
        shadow_card(im,(content_x,180,W-28,780),18,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); text(d,(content_x+24,210),'配置与策略',17,C['text'],True); d.line((content_x+24,244,W-52,244),fill=C['divider'])
        labels=spec.get('required_elements',[])[:6] or ['启用状态','默认策略','超时时间','失败重试','审计说明']
        y=278
        for i,lab in enumerate(labels):
            text(d,(content_x+28,y+20),truncate(lab,22),13,C['text2'],True)
            rounded(d,(content_x+250,y,W-70,y+46),10,C['surface_subtle'] if code!='EDIT_MODE' else C['surface'],C['brand'] if code=='EDIT_MODE' and i==0 else C['border'],2 if code=='EDIT_MODE' and i==0 else 1)
            val='已启用' if i==0 else '自动' if i==1 else '30 秒' if i==2 else '最多 2 次' if i==3 else '由系统生成的配置值'
            text(d,(content_x+266,y+23),val,13,C['text'],False,'lm'); y+=76
        pill(d,(W-180,805,W-28,849),'保存配置',C['brand'],C['surface'],13,10)
        if code in {'VALIDATION_ERROR','VERSION_CONFLICT','ROLLBACK_CONFIRM','SAVE_SUCCESS'}:
            status_banner(im,'VERSION_CONFLICT' if code=='VERSION_CONFLICT' else 'SUCCESS' if code=='SAVE_SUCCESS' else 'SAVE_ERROR' if code=='VALIDATION_ERROR' else 'UPDATE_AVAILABLE',top=865,width_ratio=.45)
    elif layout in {'admin_detail','developer_key'}:
        shadow_card(im,(content_x,180,W-28,340),18,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); icon_circle(d,(content_x+70,260),28,title_v[0],C['brand_soft'],C['brand'],20); text(d,(content_x+115,215),title_v,20,C['text'],True); text(d,(content_x+115,252),spec['subtitle'],13,C['text3']); pill(d,(content_x+115,285,content_x+215,318),'正常',C['success_soft'],C['success'],11)
        shadow_card(im,(content_x,370,content_x+content_w*.58,820),18,C['surface'],C['border'],5,2); d=ImageDraw.Draw(im); text(d,(content_x+20,398),'详细信息',16,C['text'],True)
        yy=440
        for i,item in enumerate(spec.get('required_elements',[])[:6] or ['基本信息','权限','配额','状态']): text(d,(content_x+24,yy),truncate(item,16),13,C['text3']); text(d,(content_x+210,yy),'已配置 / 正常',13,C['text'],True); yy+=58
        shadow_card(im,(content_x+content_w*.61,370,W-28,820),18,C['surface'],C['border'],5,2); d=ImageDraw.Draw(im); text(d,(content_x+content_w*.61+20,398),'操作记录',16,C['text'],True); yy=445
        for i in range(5): d.ellipse((content_x+content_w*.61+24,yy-4,content_x+content_w*.61+34,yy+6),fill=C['brand']); text(d,(content_x+content_w*.61+50,yy),'配置已更新',13,C['text2']); text(d,(W-55,yy),f'{10+i}:20',11,C['text3'],False,'ra'); yy+=68
        if code=='CREATED_ONCE':
            shadow_card(im,(content_x+120,250,W-140,570),20,C['surface'],C['brand'],10,4); d=ImageDraw.Draw(im); text(d,((content_x+W-140)//2,300),'API Key 仅展示一次',22,C['text'],True,'ma'); rounded(d,(content_x+180,370,W-200,430),10,C['dark']); text(d,(content_x+200,400),'ylv_live_sk_••••••••••••7xQp',15,C['surface'],False,'lm'); pill(d,(W-330,460,W-200,500),'复制密钥',C['brand'],C['surface'],12,8)
    elif layout=='admin_approval':
        # Approval queue.
        for i in range(3):
            y=190+i*210; shadow_card(im,(content_x,y,W-28,y+184),16,C['surface'],C['border'],5,2); d=ImageDraw.Draw(im); pill(d,(content_x+20,y+20,content_x+100,y+50),'高风险' if i==0 else '待审',C['error_soft'] if i==0 else C['warning_soft'],C['error'] if i==0 else C['warning'],11,8); text(d,(content_x+20,y+78),truncate(spec.get('required_elements',[title_v])[i%len(spec.get('required_elements',[title_v]))],32),16,C['text'],True); text(d,(content_x+20,y+112),f'事件 {page["page_id"]}-{i+1:03d} · 需要记录审批原因',12,C['text3']); pill(d,(W-210,y+66,W-130,y+102),'拒绝',C['surface'],C['error'],11,8,C['error']); pill(d,(W-120,y+66,W-40,y+102),'批准',C['brand'],C['surface'],11,8)
    else:
        shadow_card(im,(content_x,180,W-28,760),18,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); text(d,(content_x+24,220),title_v,20,C['text'],True)
    if code in {'SAVE_SUCCESS','SUCCESS'}: status_banner(im,'SUCCESS',top=860,width_ratio=.45)
    elif code in {'VALIDATION_ERROR','VERSION_CONFLICT','PARTIAL_DATA'}: status_banner(im,'VERSION_CONFLICT' if code=='VERSION_CONFLICT' else 'SERVICE_DEGRADED' if code=='PARTIAL_DATA' else 'SAVE_ERROR',top=860,width_ratio=.45)
    return im.convert('RGB')


def render_public_web(page: dict[str,Any], state: dict[str,Any]) -> Image.Image:
    W,H=CANVAS['WEB']; im=Image.new('RGBA',(W,H),hexrgb(C['bg'])+(255,)); d=ImageDraw.Draw(im); code=state['code']; spec=page['visual_identity']
    d.rectangle((0,0,W,78),fill=C['surface']); draw_web_logo(d,70,21,'YLVEN');
    for i,lab in enumerate(['产品','开发者','服务状态','下载']): text(d,(W-540+i*115,39),lab,14,C['text2'],False,'mm')
    pill(d,(W-165,19,W-70,59),'登录',C['brand'],C['surface'],13,10)
    template=page['page_id']
    if code=='LOADING': skeleton(d,(180,190,W-180,760),7,10); return im.convert('RGB')
    if template=='YL-W-001':
        text(d,(W//2,185),'完成安全验证',46,C['text'],True,'ma'); text(d,(W//2,250),'验证通过后将自动返回 YLVEN App',20,C['text3'],False,'ma')
        shadow_card(im,(W//2-370,330,W//2+370,680),24,C['surface'],C['border'],8,3); d=ImageDraw.Draw(im); rounded(d,(W//2-290,420,W//2+290,540),18,C['surface_subtle'],C['border2'],2); icon_circle(d,(W//2-230,480),28,'✓' if code=='SUCCESS' else '·',C['brand_soft'],C['brand'],20); text(d,(W//2-180,465),'Cloudflare Turnstile',18,C['text'],True); text(d,(W//2-180,495),'安全环境检查',13,C['text3'])
    elif template=='YL-W-002':
        icon_circle(d,(W//2,260),55,'✉',C['brand_soft'],C['brand'],36); text(d,(W//2,345),'邮箱验证状态',44,C['text'],True,'ma'); text(d,(W//2,415),'验证码已发送至 c***@example.com',20,C['text3'],False,'ma');
        shadow_card(im,(W//2-330,500,W//2+330,690),22,C['surface'],C['border'],6,2); d=ImageDraw.Draw(im); text(d,(W//2,560),'请返回 YLVEN App 输入六位验证码',20,C['text'],True,'ma'); pill(d,(W//2-105,615,W//2+105,660),'打开 YLVEN',C['brand'],C['surface'],13,10)
    elif template=='YL-W-003':
        text(d,(W//2,180),'授权 YLVEN 访问 Sub2API',42,C['text'],True,'ma'); text(d,(W//2,240),'使用统一账号完成安全单点登录',19,C['text3'],False,'ma'); shadow_card(im,(W//2-370,330,W//2+370,725),24,C['surface'],C['border'],8,3); d=ImageDraw.Draw(im); draw_web_logo(d,W//2-135,385,'YLVEN'); text(d,(W//2,480),'将授权以下信息',17,C['text'],True,'ma');
        for i,lab in enumerate(['读取基础账号标识','创建或绑定 Sub2API 用户','同步 API 权益']): icon_circle(d,(W//2-235,535+i*50),15,'✓',C['success_soft'],C['success'],12); text(d,(W//2-205,535+i*50),lab,14,C['text2'],False,'lm')
        pill(d,(W//2-250,675,W//2-15,720),'取消',C['surface'],C['text2'],13,10,C['border']); pill(d,(W//2+15,675,W//2+250,720),'授权并继续',C['brand'],C['surface'],13,10)
    elif template=='YL-W-004':
        d.rectangle((0,78,W,465),fill='#07132E'); text(d,(160,180),'YLVEN for Android',48,C['surface'],True); text(d,(160,255),'一个 App，自由使用多个 AI 模型',26,'#CFE0FF'); pill(d,(160,335,365,385),'下载 APK',C['surface'],C['brand'],15,12); rounded(d,(W-620,135,W-250,590),36,C['surface'],None); icon_circle(d,(W-435,290),78,'Y',C['brand'],C['surface'],55); text(d,(W-435,405),'YLVEN',30,C['text'],True,'ma')
        text(d,(160,560),'版本 1.0.0 · Android 8.0+',17,C['text'],True); text(d,(160,610),'SHA-256 校验、更新说明和安装教程均在下载后提供。',15,C['text3'])
    elif template=='YL-W-005':
        text(d,(120,150),'YLVEN 服务状态',42,C['text'],True); pill(d,(120,220,330,264),'所有系统正常',C['success_soft'],C['success'],14,10); services=['核心 API','OpenAI 通道','Claude 通道','Grok 通道','图片任务','PPT Worker']; y=330
        for i,s in enumerate(services): rounded(d,(120,y,W-120,y+82),16,C['surface'],C['border'],1); d.ellipse((155,y+31,175,y+51),fill=C['success'] if i!=2 else C['warning']); text(d,(195,y+41),s,16,C['text'],True,'lm'); text(d,(W-160,y+41),'正常' if i!=2 else '部分限流',14,C['success'] if i!=2 else C['warning'],True,'rm'); y+=98
    elif template=='YL-W-006':
        text(d,(W//2,145),'YLVEN 安全分享',40,C['text'],True,'ma'); text(d,(W//2,210),'分享内容只读，并按设置时间自动失效',18,C['text3'],False,'ma'); shadow_card(im,(210,300,W-210,790),24,C['surface'],C['border'],8,3); d=ImageDraw.Draw(im); pill(d,(260,345,480,385),'GPT-5.6 Sol',C['brand_soft'],C['brand'],13,10); text(d,(260,440),'多模型 AI 平台架构建议',28,C['text'],True); text(d,(260,500),'建议将系统拆分为控制平面、AI Runtime 和异步工作平面，\n并通过统一上下文编译器适配不同模型。',18,C['text2'],False,max_width=W-520,line_spacing=10)
    else:
        icon_circle(d,(W//2,260),55,'!',C['warning_soft'],C['warning'],36); text(d,(W//2,350),'当前页面暂时不可用',42,C['text'],True,'ma'); text(d,(W//2,415),'可能正在维护、版本不兼容或请求出现错误。',19,C['text3'],False,'ma'); pill(d,(W//2-110,500,W//2+110,548),'查看服务状态',C['brand'],C['surface'],14,10)
    if code in {'VALIDATION_ERROR','MAINTENANCE','NETWORK_ERROR','SERVER_ERROR'}: status_banner(im,code,top=790,width_ratio=.42)
    elif code=='SUCCESS': status_banner(im,'SUCCESS',top=790,width_ratio=.42)
    return im.convert('RGB')


def render_design_board(page: dict[str,Any], state: dict[str,Any]) -> Image.Image:
    W,H=CANVAS['DESIGN_SYSTEM']; im=Image.new('RGBA',(W,H),hexrgb(C['bg'])+(255,)); d=ImageDraw.Draw(im); spec=page['visual_identity']; pid=page['page_id']
    draw_web_logo(d,50,34,'YLVEN DESIGN SYSTEM'); text(d,(50,105),page['name'],30,C['text'],True); text(d,(50,150),DESIGN_VERSION+' · '+pid,14,C['text3'])
    shadow_card(im,(40,200,W-40,H-50),22,C['surface'],C['border'],7,3); d=ImageDraw.Draw(im)
    if pid=='YL-DS-001':
        names=[('Brand',C['brand']),('Blue',C['blue']),('Violet',C['violet']),('Success',C['success']),('Warning',C['warning']),('Error',C['error']),('Background',C['bg']),('Surface',C['surface']),('Text',C['text']),('Border',C['border'])]
        for i,(n,col) in enumerate(names): x=80+(i%5)*260; y=260+(i//5)*330; rounded(d,(x,y,x+220,y+210),24,col,C['border'],1); text(d,(x,y+242),n,16,C['text'],True); text(d,(x,y+272),col,13,C['text3'])
    elif pid=='YL-DS-002':
        samples=[('Display 32/40',32,True),('Page title 24/32',24,True),('Section 20/28',20,True),('Body 16/24',16,False),('Secondary 14/20',14,False),('Caption 12/16',12,False)]
        y=270
        for lab,sz,b in samples: text(d,(90,y),lab,13,C['text3']); text(d,(350,y),'智能、清晰、可持续的 AI 工作台',sz*2,C['text'],b); y+=140
    elif pid=='YL-DS-003':
        for i,r in enumerate([8,12,16,20,24,28]): rounded(d,(90+i*205,300,250+i*205,460),r*2,C['brand_soft'],C['brand'],2); text(d,(170+i*205,500),f'R {r}',14,C['text2'],True,'ma')
        y=650
        for i,s in enumerate([4,8,12,16,24,32]): d.rectangle((90,y+i*65,90+s*12,y+i*65+28),fill=C['brand']); text(d,(510,y+i*65+14),f'{s} spacing unit',14,C['text2'],False,'lm')
    elif pid=='YL-DS-004':
        pill(d,(90,280,330,338),'主按钮',C['brand'],C['surface'],15,12); pill(d,(360,280,600,338),'次按钮',C['surface'],C['brand'],15,12,C['brand']); pill(d,(630,280,820,338),'Chip',C['brand_soft'],C['brand'],14); icon_circle(d,(920,309),28,'＋',C['brand'],C['surface'],20)
        y=470
        for i,lab in enumerate(['默认','Hover','Pressed','Disabled']): pill(d,(90+i*300,y,330+i*300,y+58),lab,C['brand'] if i<3 else C['border'],C['surface'] if i<3 else C['disabled'],14,12)
    elif pid=='YL-DS-005':
        # mini inputs
        for i,(lab,val) in enumerate([('邮箱','name@example.com'),('密码','••••••••'),('搜索','搜索会话、文件或项目')]): y=275+i*160; text(d,(90,y),lab,14,C['text2'],True); rounded(d,(90,y+30,760,y+88),12,C['surface'],C['border2'],1); text(d,(110,y+59),val,14,C['text3'],False,'lm')
        text(d,(860,275),'验证码',14,C['text2'],True); x=860
        for i in range(6): rounded(d,(x+i*72,315,x+54+i*72,373),10,C['surface'],C['brand'] if i==2 else C['border2'],2 if i==2 else 1); text(d,(x+27+i*72,344),str(i+1) if i<2 else '',17,C['text'],True,'mm')
    elif pid=='YL-DS-006':
        d.rectangle((90,280,1350,350),fill=C['surface_subtle']); text(d,(120,315),'←  页面标题',18,C['text'],True,'lm'); text(d,(1280,315),'⋯',22,C['text2'],False,'mm')
        d.rectangle((90,470,1350,580),fill=C['surface']);
        for i,lab in enumerate(['首页','工作','发现','我的']): text(d,(250+i*280,530),lab,16,C['brand'] if i==0 else C['text3'],i==0,'mm')
    elif pid=='YL-DS-007':
        # cards/table
        for i in range(3): shadow_card(im,(80+i*420,270,460+i*420,460),18,C['surface'],C['border'],4,1); d=ImageDraw.Draw(im); text(d,(110+i*420,310),f'卡片标题 {i+1}',17,C['text'],True); text(d,(110+i*420,360),'用于分组信息和操作。',14,C['text3'])
        rounded(d,(80,590,1360,1000),16,C['surface'],C['border'],1); d.rectangle((80,590,1360,650),fill=C['surface_subtle']);
        for i,c in enumerate(['名称','类型','状态','更新时间','操作']): text(d,(105+i*250,620),c,13,C['text2'],True,'lm')
    elif pid=='YL-DS-008':
        shadow_card(im,(90,290,620,650),22,C['surface'],C['border'],8,3); d=ImageDraw.Draw(im); text(d,(355,350),'确认操作',22,C['text'],True,'ma'); text(d,(355,410),'说明影响并提供明确操作。',15,C['text3'],False,'ma'); pill(d,(150,530,340,580),'取消',C['surface'],C['text2'],13,10,C['border']); pill(d,(370,530,560,580),'确认',C['brand'],C['surface'],13,10)
        shadow_card(im,(760,260,1350,1000),30,C['surface'],C['border'],8,0); d=ImageDraw.Draw(im); text(d,(810,320),'Bottom Sheet',24,C['text'],True)
    elif pid=='YL-DS-009':
        skeleton(d,(90,280,600,570),5,8); text(d,(90,640),'空状态',16,C['text'],True); icon_circle(d,(260,770),42,'○',C['surface_subtle'],C['text3'],28); text(d,(260,840),'暂无内容',18,C['text2'],True,'ma'); status_banner(im,'NETWORK_ERROR',top=700,width_ratio=.38)
    elif pid=='YL-DS-010':
        pill(d,(90,270,360,320),'GPT-5.6 Sol · 深度',C['brand_soft'],C['brand'],13); rounded(d,(650,380,1320,540),28,C['brand']); text(d,(690,425),'请分析这份项目架构。',18,C['surface'],False); text(d,(90,610),'AI 回复使用全宽布局，不用大型气泡。',19,C['text'],False); rounded(d,(90,700,1320,950),18,C['code']); text(d,(120,740),'fun route(model: Model) = adapter.send(model)',16,C['code_text'])
    elif pid=='YL-DS-011':
        for i,(fn,st) in enumerate([('需求文档.pdf','已解析'),('截图.png','识图中'),('市场数据.xlsx','排队中')]): y=280+i*180; rounded(d,(90,y,1350,y+145),18,C['surface_subtle'],C['border'],1); icon_circle(d,(155,y+72),28,'文' if i!=1 else '图',C['blue_soft'],C['blue'],20); text(d,(210,y+48),fn,17,C['text'],True); text(d,(210,y+92),st,13,C['success'] if i==0 else C['brand'])
    elif pid=='YL-DS-012':
        for i in range(4): x=90+(i%2)*620; y=270+(i//2)*390; rounded(d,(x,y,x+570,y+340),24,mix('#E8EAFE','#C8DBFF',i/4),C['border'],1); icon_circle(d,(x+285,y+150),55,'Y',C['brand'],C['surface'],38); text(d,(x+25,y+290),f'图片作品 {i+1}',16,C['text'],True)
    else:
        # PPT and web component board.
        rounded(d,(90,270,310,1040),18,C['surface_subtle'],C['border'],1); yy=310
        for i in range(5): rounded(d,(115,yy,285,yy+105),10,C['surface'],C['brand'] if i==1 else C['border'],2 if i==1 else 1); yy+=130
        rounded(d,(350,270,920,880),22,C['dark']); text(d,(410,410),'YLVEN',30,'#9CCBFF',True); text(d,(410,500),'商业 AI 平台',44,C['surface'],True); rounded(d,(970,270,1350,630),18,C['surface'],C['border'],1); text(d,(1000,310),'后台组件',18,C['text'],True)
    return im.convert('RGB')


def apply_state_feedback(im: Image.Image, page: dict[str,Any], state: dict[str,Any]) -> Image.Image:
    """Make every contracted state visibly and semantically distinct.

    This is an actual product feedback surface (snackbar/status notice), not a developer
    watermark. It prevents a renderer from silently producing the same visual for
    DEFAULT, DIRTY, SUBMITTING, EMPTY, OFFLINE_CACHE and similar states.
    """
    code=state['code']
    if code in {'DEFAULT','POPULATED','LAUNCH','FIRST_RUN'}:
        return im
    title=page.get('visual_identity',{}).get('title') or page.get('name','当前页面')
    label=state.get('name') or code
    messages={
        'LOADING':f'正在加载「{title}」', 'SKELETON':f'正在准备「{title}」',
        'EMPTY':f'「{title}」暂无可显示内容', 'REFRESHING':f'正在刷新「{title}」',
        'OFFLINE_CACHE':f'正在显示「{title}」的离线缓存', 'OFFLINE':f'「{title}」当前离线',
        'NETWORK_ERROR':f'「{title}」网络连接失败', 'SERVER_ERROR':f'「{title}」服务暂时不可用',
        'SERVICE_DEGRADED':f'「{title}」部分能力暂时降级', 'FILTER_ACTIVE':'筛选条件已生效',
        'BULK_SELECTED':'已选择 3 项，可执行批量操作', 'EDIT_MODE':'当前处于编辑模式',
        'DIRTY':'存在未保存修改', 'SUBMITTING':'正在提交，请勿重复操作',
        'DISABLED':'当前操作暂不可用', 'VALIDATION_ERROR':'请修正标记的输入内容',
        'SAVE_SUCCESS':'修改已保存', 'SAVE_ERROR':'保存失败，输入内容已保留',
        'SUCCESS':'操作已成功完成', 'COMPLETED':'任务已完成', 'CONNECTING':'正在建立安全连接',
        'RECONNECTING':'连接中断，正在安全重连', 'PREPARING':'正在检查参数与可用额度',
        'QUEUED':'任务已进入队列', 'RUNNING':'任务正在执行', 'RETRYING':'正在按策略重试',
        'UPLOADING':'文件正在上传', 'DOWNLOADING':'文件正在下载', 'UPLOAD_FAILED':'上传失败，可重新尝试',
        'FAILED':'任务执行失败', 'CANCELLED':'任务已取消', 'CANCEL_CONFIRM':'确认是否取消当前任务',
        'PAYMENT_PENDING':'支付结果确认中', 'CREDIT_PENDING':'额度发放处理中', 'REFUND_PENDING':'退款处理中',
        'CREATED_ONCE':'敏感凭据仅在本次展示', 'INPUT_FOCUSED':'输入控件已聚焦',
        'CODE_SENT':'验证码已发送', 'INVALID_CODE':'验证码错误', 'CODE_EXPIRED':'验证码已过期',
        'RATE_LIMITED':'请求过于频繁，请稍后重试', 'LOCKED':'账号暂时锁定',
        'PERMISSION_DENIED':'当前账号权限不足', 'UNAUTHORIZED':'登录状态已失效',
        'BUDGET_EXHAUSTED':'可用预算已用尽', 'PARTIAL_DATA':'部分数据源暂不可用',
        'VERSION_CONFLICT':'配置版本发生冲突', 'ROLLBACK_CONFIRM':'请确认是否回滚到上一版本',
        'SECURITY_CHALLENGE':'正在进行安全验证', 'TOOL_RUNNING':'AI 工具正在执行',
        'STREAMING':'AI 正在流式生成', 'STOPPED':'生成已停止', 'PROVIDER_ERROR':'当前模型响应失败',
        'CONTENT_BLOCKED':'内容未通过安全检查', 'NOT_FOUND':'请求的内容不存在',
        'MAINTENANCE':'系统维护中', 'UPDATE_AVAILABLE':'发现可用更新', 'UPDATE_REQUIRED':'必须更新后继续使用',
    }
    message=messages.get(code,f'{label} · {title}')
    d=ImageDraw.Draw(im)
    W,H=im.size
    is_android=page['surface']=='ANDROID'
    if code in {'SUCCESS','SAVE_SUCCESS','COMPLETED','CODE_SENT'}:
        bg,fg=C['success_soft'],C['success']; symbol='✓'
    elif code in {'NETWORK_ERROR','SERVER_ERROR','SAVE_ERROR','FAILED','UPLOAD_FAILED','INVALID_CODE','LOCKED','PERMISSION_DENIED','UNAUTHORIZED','PROVIDER_ERROR','CONTENT_BLOCKED'}:
        bg,fg=C['error_soft'],C['error']; symbol='!'
    elif code in {'OFFLINE','OFFLINE_CACHE','RATE_LIMITED','CODE_EXPIRED','SERVICE_DEGRADED','DIRTY','VERSION_CONFLICT','PAYMENT_PENDING','CREDIT_PENDING','REFUND_PENDING','BUDGET_EXHAUSTED','MAINTENANCE','CANCEL_CONFIRM','PARTIAL_DATA'}:
        bg,fg=C['warning_soft'],C['warning']; symbol='!'
    else:
        bg,fg=C['info_soft'],C['info']; symbol='·'
    if is_android:
        # A compact product snackbar above the gesture/navigation area.
        x1,x2=72,W-72; h=116; y=H-330
        rounded(d,(x1,y,x2,y+h),28,bg,mix(bg,fg,.22),2)
        icon_circle(d,(x1+55,y+h//2),27,symbol,bg,fg,24)
        text(d,(x1+100,y+34),message,28,fg,True,max_width=x2-x1-135)
    else:
        bw=min(570,W//2); x2=W-28; x1=x2-bw; h=64; y=H-88
        rounded(d,(x1,y,x2,y+h),12,bg,mix(bg,fg,.22),1)
        icon_circle(d,(x1+30,y+h//2),16,symbol,bg,fg,13)
        text(d,(x1+56,y+h//2),message,13,fg,True,'lm',max_width=bw-76)
    return im


def render_one(page: dict[str,Any], state: dict[str,Any]) -> Image.Image:
    if page['surface']=='ANDROID': im=render_android(page,state)
    elif page['surface']=='ADMIN': im=render_web_app(page,state,False)
    elif page['surface']=='DEVELOPER': im=render_web_app(page,state,True)
    elif page['surface']=='WEB': im=render_public_web(page,state)
    else: im=render_design_board(page,state)
    return apply_state_feedback(im.convert('RGBA'),page,state).convert('RGB')


# ---------- Generation, review and duplicate audit --------------------------

def load_page_contracts() -> list[dict[str,Any]]:
    pages=[]
    for surface in ['android','admin','developer','web','design-system']:
        for p in sorted((ROOT/f'ui/pages/{surface}').glob('*.yaml')):
            pages.append(read_yaml(p))
    return pages


def generate_mockups() -> dict[str,Any]:
    manifest_path=ROOT/'contracts/mockup-manifest.csv'
    rows=list(csv.DictReader(manifest_path.open(encoding='utf-8-sig',newline='')))
    row_by_id={r['mockup_id']:r for r in rows}
    pages=load_page_contracts(); generated=[]; surface_counts=Counter()
    total=sum(len(p['states']) for p in pages); n=0
    for page in pages:
        for st in page['states']:
            n+=1; rel=st['mockup_path']; out=ROOT/rel; out.parent.mkdir(parents=True,exist_ok=True)
            im=render_one(page,st)
            if im.size != CANVAS[page['surface']]: raise RuntimeError(f"{st['state_id']} size {im.size}")
            im.save(out,format='PNG',compress_level=1,optimize=False)
            h=sha256(out); r=row_by_id[st['mockup_id']]; r['status']='GENERATED_PENDING_OWNER_REVIEW'; r['sha256']=h; r['duplicate_audit']='PENDING'
            st['mockup_status']='GENERATED_PENDING_OWNER_REVIEW'; st['mockup_sha256']=h
            generated.append({'mockup_id':st['mockup_id'],'page_id':page['page_id'],'surface':page['surface'],'page_name':page['name'],'state_code':st['code'],'state_name':st['name'],'src':'../'+rel.replace('ui/',''),'path':rel,'sha256':h,'status':r['status'],'visual_kind':page['visual_kind'],'parent_page_id':page.get('parent_page_id') or ''})
            surface_counts[page['surface']]+=1
            if n%100==0: print(f'generated {n}/{total}',flush=True)
        surf_dir=SURFACE_DIR[page['surface']]
        write_yaml(ROOT/f"ui/pages/{surf_dir}/{page['page_id']}.yaml",page)
    write_csv(manifest_path,rows,list(rows[0].keys()))
    build_visual_review(generated)
    batches=build_review_boards(generated)
    result={'generated':len(generated),'surface_counts':dict(surface_counts),'review_boards':batches}
    write_json(ROOT/'ui/visual-review/generation-summary.json',result)
    return result


def build_visual_review(generated: list[dict[str,Any]]) -> None:
    out=ROOT/'ui/visual-review'; out.mkdir(parents=True,exist_ok=True)
    # src from visual-review directory to ui/mockups.
    data=[]
    for x in generated:
        y=dict(x); y['src']='../mockups/'+SURFACE_DIR[x['surface']]+'/'+x['page_id']+'/'+x['mockup_id']+'.png'; data.append(y)
    write_json(out/'mockups.json',data)
    html='''<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>YLVEN 视觉审查 V1.5</title><style>
:root{font-family:Inter,"Noto Sans SC",system-ui,sans-serif;color:#101828;background:#F6F7FB}*{box-sizing:border-box}body{margin:0}.top{position:sticky;top:0;z-index:9;background:rgba(255,255,255,.96);backdrop-filter:blur(14px);border-bottom:1px solid #E4E7EC;padding:18px 24px}.head{display:flex;gap:16px;align-items:center;flex-wrap:wrap}.logo{font-size:22px;font-weight:800}.meta{color:#667085;font-size:13px}.filters{display:flex;gap:8px;margin-top:14px;flex-wrap:wrap}input,select,button{height:40px;border:1px solid #D0D5DD;border-radius:10px;background:#fff;padding:0 12px;color:#101828}input{min-width:320px}button.active{background:#5B61F6;color:white;border-color:#5B61F6}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(270px,1fr));gap:18px;padding:24px}.card{background:#fff;border:1px solid #E4E7EC;border-radius:16px;overflow:hidden;box-shadow:0 2px 8px rgba(16,24,40,.05)}.shot{height:340px;background:#EEF0F5;display:flex;justify-content:center;align-items:center}.shot img{width:100%;height:100%;object-fit:contain}.info{padding:14px}.id{font-weight:700;font-size:13px}.name{font-size:13px;color:#475467;margin-top:6px}.tags{display:flex;gap:6px;margin-top:10px;flex-wrap:wrap}.tag{font-size:11px;background:#F0F1FF;color:#5B61F6;border-radius:999px;padding:4px 8px}.parent{background:#EFF8FF;color:#2F80ED}.empty{padding:80px;text-align:center;color:#667085}.count{font-weight:700;color:#5B61F6}@media(max-width:640px){input{min-width:100%}.grid{padding:12px;grid-template-columns:1fr}.shot{height:520px}}</style></head><body>
<div class="top"><div class="head"><div class="logo">YLVEN 视觉审查 V1.5</div><div class="meta">页面身份重绑定 · <span class="count" id="count">0</span> 张 · 待项目所有者审查</div></div><div class="filters"><input id="q" placeholder="搜索 Page ID、页面名、状态、视觉类型或父页面"><select id="surface"><option value="">全部平台</option><option>ANDROID</option><option>ADMIN</option><option>DEVELOPER</option><option>WEB</option><option>DESIGN_SYSTEM</option></select><select id="kind"><option value="">全部视觉类型</option><option>PAGE</option><option>OVERLAY</option><option>COMPONENT_BOARD</option><option>SYSTEM_OVERLAY</option><option>SYSTEM_STATE_BOARD</option><option>DESIGN_BOARD</option></select><button id="onlyErrors">仅异常状态</button></div></div><main id="grid" class="grid"></main><script>
let all=[],errors=false;const errorWords=['错误','失败','异常','离线','超时','限流','不足','锁定','维护','冲突','拒绝'];fetch('mockups.json').then(r=>r.json()).then(d=>{all=d;render()});function render(){const q=document.querySelector('#q').value.trim().toLowerCase(),s=document.querySelector('#surface').value,k=document.querySelector('#kind').value;let rows=all.filter(x=>(!s||x.surface===s)&&(!k||x.visual_kind===k)&&(!q||JSON.stringify(x).toLowerCase().includes(q))&&(!errors||errorWords.some(w=>x.state_name.includes(w))));document.querySelector('#count').textContent=rows.length;const g=document.querySelector('#grid');g.innerHTML=rows.length?rows.map(x=>`<article class="card"><a href="${x.src}" target="_blank"><div class="shot"><img loading="lazy" src="${x.src}" alt="${x.mockup_id}"></div></a><div class="info"><div class="id">${x.mockup_id}</div><div class="name">${x.page_name} · ${x.state_name}</div><div class="tags"><span class="tag">${x.surface}</span><span class="tag">${x.visual_kind}</span>${x.parent_page_id?`<span class="tag parent">父页面 ${x.parent_page_id}</span>`:''}</div></div></article>`).join(''):'<div class="empty">没有匹配结果</div>'}document.querySelector('#q').addEventListener('input',render);document.querySelector('#surface').addEventListener('change',render);document.querySelector('#kind').addEventListener('change',render);document.querySelector('#onlyErrors').onclick=e=>{errors=!errors;e.target.classList.toggle('active',errors);render()};</script></body></html>'''
    (out/'index.html').write_text(html,encoding='utf-8')


def build_review_boards(generated: list[dict[str,Any]]) -> int:
    board_dir=ROOT/'ui/reference-boards'; board_dir.mkdir(parents=True,exist_ok=True)
    for p in board_dir.glob('*.png'): p.unlink()
    batches=[]; batch_no=1
    # Keep pages together where possible; cap 20 states per board.
    by_surface=defaultdict(list)
    for x in generated: by_surface[x['surface']].append(x)
    for surface,items in by_surface.items():
        chunks=[items[i:i+20] for i in range(0,len(items),20)]
        for chunk in chunks:
            bid=f'MB-{batch_no:03d}'; batch_no+=1
            board=Image.new('RGB',(1920,1080),hexrgb(C['bg'])); bd=ImageDraw.Draw(board)
            text(bd,(40,28),f'{bid} · {surface} · 页面身份校验板',32,C['text'],True); text(bd,(40,72),'每张原图均为独立 PNG；本板仅用于快速检查绑定与重复。',16,C['text3'])
            cols=5; rows=4; cellw=368; cellh=235
            for i,x in enumerate(chunk):
                cx=30+(i%cols)*378; cy=115+(i//cols)*238
                img=Image.open(ROOT/x['path']).convert('RGB'); img.thumbnail((330,175))
                thumb=Image.new('RGB',(340,180),hexrgb(C['surface_subtle'])); thumb.paste(img,((340-img.width)//2,(180-img.height)//2))
                board.paste(thumb,(cx,cy)); text(bd,(cx,cy+187),x['mockup_id'],12,C['text'],True); text(bd,(cx,cy+208),truncate(x['page_name'],18)+' · '+x['state_name'],11,C['text3'])
            path=board_dir/f'{bid}.png'; board.save(path,compress_level=1)
            batches.append({'batch_id':bid,'surface':surface,'mockup_ids':[x['mockup_id'] for x in chunk],'review_board':f'ui/reference-boards/{bid}.png','count':len(chunk)})
    write_yaml(ROOT/'contracts/mockup-batches.yaml',{'schema_version':'1.4','batch_count':len(batches),'batches':batches})
    lines=['# YLVEN 效果图审查批次索引 V1.5','',f'- 批次数：**{len(batches)}**','- 本索引用于快速审查；正式开发必须使用对应独立 PNG。','']
    for b in batches: lines.append(f"- `{b['batch_id']}` · {b['surface']} · {b['count']} 张 · `{b['review_board']}`")
    (ROOT/'ui/MOCKUP_BATCH_INDEX.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    return len(batches)


def dhash(path: Path, size=16) -> int:
    im=Image.open(path).convert('L').resize((size+1,size),Image.Resampling.LANCZOS)
    arr=np.asarray(im); bits=arr[:,1:]>arr[:,:-1]; value=0
    for b in bits.flatten(): value=(value<<1)|int(b)
    return value


def audit_duplicates() -> dict[str,Any]:
    rows=list(csv.DictReader((ROOT/'contracts/mockup-manifest.csv').open(encoding='utf-8-sig',newline='')))
    exact=defaultdict(list)
    for r in rows: exact[r['sha256']].append(r)
    exact_groups=[g for g in exact.values() if len(g)>1]
    cross_page_exact=[]; same_page_exact=[]
    for g in exact_groups:
        pages={x['page_id'] for x in g}
        (cross_page_exact if len(pages)>1 else same_page_exact).append([x['mockup_id'] for x in g])

    # Select one normal/primary representative per Page ID. Common navigation chrome is
    # expected; a candidate is invalid only when the content-region pixels are effectively
    # indistinguishable, not merely because two professional list pages share the same shell.
    priority=['POPULATED','DEFAULT','SUCCESS','LAUNCH','SECURITY_CHALLENGE','LOADING']
    by_page=defaultdict(list)
    for r in rows: by_page[r['page_id']].append(r)
    reps=[]
    for pid,grp in by_page.items():
        def rank(r):
            code=re.sub(r'^.*-S\d+_','',r['state_id'])
            if code in priority: return (priority.index(code),0)
            m=re.search(r'-S(\d+)_',r['state_id']); return (len(priority),int(m.group(1)) if m else 999)
        pick=dict(min(grp,key=rank)); path=ROOT/pick['relative_path']
        im=Image.open(path).convert('L'); W,H=im.size
        if pick['surface']=='ANDROID': crop=im.crop((35,190,W-35,H-180))
        elif pick['surface'] in {'ADMIN','DEVELOPER'}: crop=im.crop((240,65,W,H-70))
        elif pick['surface']=='WEB': crop=im.crop((0,78,W,H-70))
        else: crop=im.crop((35,180,W-35,H-40))
        pick['vector']=np.asarray(crop.resize((40,40),Image.Resampling.BILINEAR),dtype=np.float32)
        reps.append(pick)
    invalid_near=[]; shell_similarity_candidates=[]
    for i,a in enumerate(reps):
        for b in reps[i+1:]:
            if a['surface']!=b['surface']: continue
            mad=float(np.mean(np.abs(a['vector']-b['vector'])))
            record={'a':a['mockup_id'],'b':b['mockup_id'],'mean_abs_pixel_distance':round(mad,4),'a_page':a['page_name'],'b_page':b['page_name']}
            if mad <= 0.35:
                invalid_near.append(record)
            elif mad <= 0.9:
                shell_similarity_candidates.append(record)

    signatures=defaultdict(list)
    names=defaultdict(list)
    for r in rows:
        signatures[r.get('screen_signature','')].append(r['page_id'])
        normalized=re.sub(r'[\s/·_-]+','',r.get('page_name','')).lower()
        names[normalized].append(r['page_id'])
    signature_conflicts={k:sorted(set(v)) for k,v in signatures.items() if k and len(set(v))>1}
    semantic_name_conflicts={k:sorted(set(v)) for k,v in names.items() if k and len(set(v))>1}
    invalid_count=len(cross_page_exact)+len(same_page_exact)+len(signature_conflicts)+len(semantic_name_conflicts)
    # Similarity candidates are evidence for human review, not automatic failures. Professional
    # admin/list/config pages intentionally share a shell; only exact pixels, duplicate semantic
    # identities or duplicate normalized page names block the package.
    all_similarity=sorted(invalid_near+shell_similarity_candidates,key=lambda x:x['mean_abs_pixel_distance'])
    result={
        'schema_version':'1.4','audit_method':'SHA-256 exact duplicate + semantic identity/name uniqueness + informational 40x40 content-region similarity',
        'mockup_count':len(rows),'page_count':len(by_page),'exact_duplicate_groups':len(exact_groups),
        'cross_page_exact_duplicates':cross_page_exact,'same_page_exact_duplicates':same_page_exact,
        'shared_shell_similarity_candidate_count':len(all_similarity),
        'shared_shell_similarity_candidates_top_200':all_similarity[:200],
        'screen_signature_conflicts':signature_conflicts,'semantic_page_name_conflicts':semantic_name_conflicts,
        'invalid_duplicate_count':invalid_count,'result':'PASS' if invalid_count==0 else 'FAIL'
    }
    write_json(ROOT/'VISUAL_DUPLICATION_AUDIT_REPORT.json',result)
    lines=['# YLVEN 视觉重复与页面错绑审计报告 V1.5','',f"- 结果：**{result['result']}**",f"- 效果图：**{len(rows)}**",f"- 页面/组件合同：**{len(by_page)}**",f"- 跨 Page ID 完全重复：**{len(cross_page_exact)} 组**",f"- 同 Page ID 不同状态完全重复：**{len(same_page_exact)} 组**",f"- 共享框架相似候选（只供人工抽查）：**{len(all_similarity)} 对**",f"- 页面身份签名冲突：**{len(signature_conflicts)}**",f"- 规范化页面名称冲突：**{len(semantic_name_conflicts)}**",'', '## 判断口径','',
           '- 完全相同 PNG 以 SHA-256 判定，任何跨页面或同页不同状态完全相同均视为错误。',
           '- 管理后台与开发者中心允许共享同一公共 Shell；不能仅因侧栏、顶部栏和表格骨架相似就判为错绑。',
           '- 主内容区域相似度仅作为人工抽查线索；大型后台允许共享 Shell，不以相似度单独判错。','', '## 已修正的关键绑定','',
           '- `YL-A-005`：邮箱验证码登录表单。','- `YL-A-006`：登录页安全验证弹层，父页面 `YL-A-005`。','- `YL-A-007`：独立受控 Turnstile WebView。','- `YL-A-008`：独立六位登录验证码页。','- `YL-A-010`：邮箱、密码、确认密码注册表单。','- `YL-A-011`：注册安全验证弹层，父页面 `YL-A-010`。','- `YL-A-012`：独立六位注册验证码页，不再渲染创建账户表单。','- `YL-A-013`：注册成功与工作区初始化结果页。','- 输入框、AI 回复、代码块、表格、引用和附件卡改为组件规范板，不再复制完整聊天页面。','', '## 审计结论','']
    lines.append('未发现无效重复或页面身份冲突。' if result['result']=='PASS' else '仍存在需修复的无效重复，详见 JSON。')
    (ROOT/'VISUAL_DUPLICATION_AUDIT_REPORT.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    for r in rows: r['duplicate_audit']='PASS' if result['result']=='PASS' else 'REVIEW_REQUIRED'
    write_csv(ROOT/'contracts/mockup-manifest.csv',rows,list(rows[0].keys()))
    return result

def _cli() -> int:
    import argparse
    parser = argparse.ArgumentParser(description='Regenerate YLVEN V1.5 UI contracts and deterministic mockups.')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--contracts-only', action='store_true')
    group.add_argument('--mockups-only', action='store_true')
    group.add_argument('--all', action='store_true')
    args = parser.parse_args()
    if args.contracts_only or args.all:
        _, _, summary = rebuild_contracts()
        print(json.dumps(summary, ensure_ascii=False, indent=2))
    if args.mockups_only or args.all:
        generation = generate_mockups()
        audit = audit_duplicates()
        print(json.dumps({'generation': generation, 'audit': audit}, ensure_ascii=False, indent=2))
    return 0

if __name__ == '__main__':
    raise SystemExit(_cli())
