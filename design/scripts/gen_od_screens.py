#!/usr/bin/env python3
"""gen_od_screens.py — design/od/screens/<slug>-<state>.html 생성.

규약: <main class="screen"> + ../tokens.css + ../components.css 링크만.
<style>·style= 금지, 아이콘은 icons.svg <use> 스프라이트, button-primary 화면당 1개.
컴포넌트 클래스 = design/screens.md 구성표의 kebab 이름.
"""
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent / "od"
SCREENS = ROOT / "screens"


def icon(name, size=20):
    return ('<svg class="icon" data-icon="{n}" width="{s}" height="{s}">'
            '<use href="../icons.svg#i-{n}"/></svg>').format(n=name, s=size)


def sb_item(label, ic=None, dot=None, tag=None, active=False):
    cls = "sidebar-item" + (" is-active" if active else "")
    lead = icon(ic) if ic else ''
    d = '<span class="sidebar-dot{}"></span>'.format(
        " " + dot if dot else "") if dot else ''
    t = '<span class="sidebar-tag">{}</span>'.format(tag) if tag else ''
    return ('<button class="{c}">{i}{d}<span class="sidebar-item-label">{l}</span>{t}'
            '</button>').format(c=cls, i=lead, d=d, l=label, t=t)


def sidebar(active=None):
    items = {
        "chat": sb_item("삼성전자", dot="is-on", active=(active == "chat")),
        "chat2": sb_item("네이버", dot="", active=False),
        "project": sb_item("Kafka 파이프라인", dot="is-busy", tag="프로젝트",
                           active=(active == "project-chat")),
        "documents": sb_item("내 서류", ic="file-text", active=(active == "documents")),
        "applications": sb_item("지원 관리", ic="briefcase", active=(active == "applications")),
        "vault": sb_item("커리어 볼트", ic="archive", active=(active == "vault")),
        "interview": sb_item("면접", ic="mic", active=(active == "interview")),
        "calendar": sb_item("캘린더", ic="calendar", active=(active == "calendar")),
        "settings": sb_item("설정", ic="settings", active=(active == "settings")),
    }
    return """<aside class="sidebar">
<div class="sidebar-head"><span class="t-label">Push</span><button class="icon-button">{i}</button></div>
<div class="sidebar-section"><div class="sidebar-label">채팅</div>
{nw}{c1}{c2}{pr}</div>
<div class="sidebar-section"><div class="sidebar-label">기능</div>
{doc}{app}{vlt}{itv}{cal}</div>
<div class="sidebar-foot">{st}</div>
</aside>""".format(
        i=icon("panel-left-close"),
        nw=sb_item("새 채팅", ic="square-pen"),
        c1=items["chat"], c2=items["chat2"], pr=items["project"],
        doc=items["documents"], app=items["applications"], vlt=items["vault"],
        itv=items["interview"], cal=items["calendar"], st=items["settings"])


def header(title, actions=""):
    return ('<div class="canvas-header"><h1 class="canvas-header-title">{t}</h1>'
            '<div class="canvas-header-actions">{a}</div></div>').format(t=title, a=actions)


def page(slug, inner, active=None, bare=False):
    side = "" if bare else sidebar(active)
    return """<!doctype html>
<html lang="ko">
<head>
<meta charset="utf-8">
<title>Push — {slug}</title>
<link rel="stylesheet" href="../tokens.css">
<link rel="stylesheet" href="../components.css">
</head>
<body>
<main class="screen">
{side}<div class="canvas">{inner}</div>
</main>
</body>
</html>
""".format(slug=slug, side=side, inner=inner)


def body(inner):
    return '<div class="canvas-body">{}</div>'.format(inner)


def empty_state(text, btn=None):
    b = '<button class="button button-secondary button-md">{}</button>'.format(btn) if btn else ""
    return ('<div class="empty-state"><div class="empty-state-icon">{i}</div>'
            '<p class="t-body">{t}</p>{b}</div>').format(i=icon("inbox", 48), t=text, b=b)


def error_state():
    return ('<div class="error-state"><div class="t-h3">불러오지 못했어요</div>'
            '<p class="t-body-sm">네트워크 상태를 확인한 뒤 다시 시도해 주세요.</p>'
            '<button class="button button-secondary button-md">다시 시도</button></div>')


def skels(n=3, h=64):
    return "".join('<div class="skeleton" style="height:{}px"></div>'.format(h) for _ in range(n))


def skel_block(h):
    # skeleton inline style 금지 → 고정 높이 variant 클래스 없이, 규약상 skeleton만 사용
    return '<div class="skeleton" style="height:{}px"></div>'.format(h)


COMPOSER = """<div class="composer">
<button class="icon-button">{pc}</button>
<button class="icon-button">{sl}</button>
<input class="composer-field" placeholder="메시지 입력 — UltraResume 등 추론 설정은 슬라이더 아이콘에서">
<button class="icon-button">{sd}</button>
</div>""".format(pc=icon("paperclip"), sl=icon("sliders-horizontal"), sd=icon("send-horizontal", 16))

SUGGEST = """<div class="suggest-chips">
<button class="suggest-chips-item">지원 현황 정리해줘</button>
<button class="suggest-chips-item">이력서 초안 써줘</button>
<button class="suggest-chips-item">면접 질문 뽑아줘</button>
</div>"""

DOCS = """<div class="card-grid">
<div class="doc-card"><div class="doc-card-title">삼성전자_자소서_v1</div><div class="doc-card-meta">자기소개서 · 2시간 전</div></div>
<div class="doc-card"><div class="doc-card-title">이력서_2026</div><div class="doc-card-meta">이력서 · 어제</div></div>
<div class="doc-card"><div class="doc-card-title">포트폴리오</div><div class="doc-card-meta">포트폴리오 · 3일 전</div></div>
</div>"""

# ── 화면 정의: slug → {state: canvas inner html} ────────────────────────────
S = {}

# 1. login (사이드바 없음)
S["login"] = {
    "default": """<div class="center">
<div class="card"><div class="card-title">Push</div>
<p class="t-body-sm">채팅으로 지원·문서·프로젝트를 관리하는 데스크톱 앱</p></div>
<button class="button button-primary button-lg">Google로 계속</button>
<button class="button button-secondary button-lg">GitHub로 계속</button>
</div>""",
    "loading": """<div class="center">
<div class="card"><div class="card-title">Push</div>
<p class="t-body-sm">채팅으로 지원·문서·프로젝트를 관리하는 데스크톱 앱</p></div>
<button class="button button-primary button-lg is-loading">로그인 중</button>
<button class="button button-secondary button-lg" disabled>GitHub로 계속</button>
</div>""",
    "error": """<div class="center">
<div class="error-state"><div class="t-h3">로그인에 실패했어요</div>
<p class="t-body-sm">계정 정보를 확인한 뒤 다시 시도해 주세요.</p>
<button class="button button-primary button-lg">Google로 계속</button>
<button class="button button-secondary button-lg">GitHub로 계속</button>
</div></div>""",
}

# 2. home
_home_main = """<div class="center">
<h1 class="t-display">무엇을 도와드릴까요?</h1>
<div class="command-input">
<button class="icon-button">{pc}</button>
<input class="command-input-field" placeholder="공고 URL을 붙여넣거나, 명령을 입력하세요">
<button class="button button-primary button-md">실행 {run}</button>
</div>
{sg}</div>
<div class="canvas-header"><h2 class="canvas-header-title">최근 작업</h2></div>
<div class="card-grid">
<div class="card"><div class="card-title">삼성전자 백엔드 지원</div><div class="card-meta">자소서 초안 · 2시간 전</div></div>
<div class="card"><div class="card-title">Kafka 파이프라인</div><div class="card-meta">프로젝트 · VERIFIED</div></div>
<div class="card"><div class="card-title">네이버 프론트엔드</div><div class="card-meta">공고 분석 · 어제</div></div>
</div>""".format(pc=icon("paperclip"), run=icon("corner-down-left", 16), sg=SUGGEST)

S["home"] = {
    "default": body(_home_main),
    "empty": body("""<div class="center">
<h1 class="t-display">무엇을 도와드릴까요?</h1>
<div class="command-input">
<button class="icon-button">{pc}</button>
<input class="command-input-field" placeholder="공고 URL을 붙여넣거나, 명령을 입력하세요">
<button class="button button-primary button-md">실행 {run}</button>
</div>
{sg}</div>
{e}""".format(pc=icon("paperclip"), run=icon("corner-down-left", 16), sg=SUGGEST,
              e=empty_state("아직 최근 작업이 없어요"))),
    "loading": body("""<div class="center">
<h1 class="t-display">무엇을 도와드릴까요?</h1>
<div class="command-input">
<button class="icon-button">{pc}</button>
<input class="command-input-field" placeholder="공고 URL을 붙여넣거나, 명령을 입력하세요">
<button class="button button-primary button-md">실행 {run}</button>
</div>
{sg}</div>
<div class="canvas-header"><h2 class="canvas-header-title">최근 작업</h2></div>
<div class="card-grid">{sk}</div>""".format(pc=icon("paperclip"), run=icon("corner-down-left", 16),
              sg=SUGGEST, sk=skels(3, 96))),
    "error": body(error_state()),
}

# 3. chat
_chat_stream = """<div class="chat-stream">
<div class="chat-stream-msg chat-stream-msg-ai">이력서 초안을 작성했어요. 근거 4개가 연결되어 있습니다.</div>
<div class="chat-stream-msg chat-stream-msg-user">성과 수치를 더 구체적으로 다듬어줘</div>
<div class="chat-stream-msg chat-stream-msg-ai">수치 표현을 보강했어요. 확정하면 문서로 저장됩니다.</div>
<div class="approval-card">
<div class="approval-card-title">문서 확정 — 이력서_삼성전자_v1</div>
<div class="approval-card-body">다음 내용으로 문서를 확정할까요? 근거 4개 연결됨</div>
<div class="approval-card-actions">
<button class="button button-secondary button-sm">{x} 거부</button>
<button class="button button-primary button-sm">{c} 승인</button>
</div></div>
</div>""".format(x=icon("x"), c=icon("check"))

S["chat"] = {
    "default": header("삼성전자", '<button class="icon-button">{}</button>'.format(icon("ellipsis")))
               + _chat_stream + COMPOSER,
    "empty": header("새 채팅", '<button class="icon-button">{}</button>'.format(icon("ellipsis")))
             + '<div class="chat-stream">{}</div>'.format(SUGGEST) + COMPOSER,
    "loading": header("삼성전자", '<button class="icon-button">{}</button>'.format(icon("ellipsis")))
               + '<div class="chat-stream">{}</div>'.format(skels(4, 56)) + COMPOSER,
    "error": header("삼성전자", '<button class="icon-button">{}</button>'.format(icon("ellipsis")))
             + body(error_state()) + COMPOSER,
}

# 4. documents
S["documents"] = {
    "default": header("내 서류", '<button class="button button-primary button-sm">{} 새 문서</button>'.format(icon("plus")))
               + body(DOCS),
    "empty": header("내 서류", '<button class="button button-primary button-sm">{} 새 문서</button>'.format(icon("plus")))
             + body(empty_state("아직 문서가 없어요", "새 문서")),
    "loading": header("내 서류", '<button class="button button-primary button-sm">{} 새 문서</button>'.format(icon("plus")))
               + body('<div class="card-grid">{}</div>'.format(skels(3, 96))),
    "error": header("내 서류", '<button class="button button-primary button-sm">{} 새 문서</button>'.format(icon("plus")))
             + body(error_state()),
}

# 4-1. doc-editor
S["doc-editor"] = {
    "default": header("삼성전자_자소서_v1",
                      '<button class="button button-secondary button-sm">{d} DOCX</button>'
                      '<button class="button button-primary button-sm">{p} PDF</button>'.format(
                          d=icon("file-down"), p=icon("file-down")))
               + """<div class="editor"><div class="editor-body">
<h1 class="t-h1">삼성전자_자소서_v1</h1>
<p>저는 데이터 파이프라인 구축 경험을 바탕으로 삼성전자 백엔드 직무에 지원합니다.</p>
<p>Kafka 기반 실시간 수집 파이프라인을 설계·구현해 일 2억 건 처리를 달성했습니다.</p>
</div></div>""",
    "loading": header("문서") + '<div class="editor"><div class="editor-body">{}</div></div>'.format(skels(6, 20)),
    "error": header("문서") + body(error_state()),
}

# 5. applications
_app_cards = """<div class="card-grid">
<div class="card"><div class="card-title">삼성전자 백엔드</div><div class="card-meta">자소서 작성 중</div><span class="status-chip status-chip-ready">READY</span></div>
<div class="card"><div class="card-title">네이버 프론트엔드</div><div class="card-meta">지원 완료</div><span class="status-chip status-chip-verified">SUBMITTED</span></div>
<div class="card"><div class="card-title">카카오 플랫폼</div><div class="card-meta">공고 분석 중</div><span class="status-chip status-chip-pending">ANALYZING</span></div>
</div>"""
S["applications"] = {
    "default": header("지원 관리", '<button class="button button-primary button-sm">{} 공고 추가</button>'.format(icon("plus")))
               + body(_app_cards)
               + '<div class="toast">{} 상태가 업데이트됐어요</div>'.format(icon("check", 16)),
    "empty": header("지원 관리", '<button class="button button-primary button-sm">{} 공고 추가</button>'.format(icon("plus")))
             + body(empty_state("아직 지원 내역이 없어요", "공고 추가")),
    "loading": header("지원 관리", '<button class="button button-primary button-sm">{} 공고 추가</button>'.format(icon("plus")))
               + body('<div class="card-grid">{}</div>'.format(skels(3, 96))),
    "error": header("지원 관리", '<button class="button button-primary button-sm">{} 공고 추가</button>'.format(icon("plus")))
             + body(error_state()),
}

# 6. job-detail
S["job-detail"] = {
    "default": header("삼성전자 — 백엔드 엔지니어")
               + body("""<div class="card"><div class="card-title">요구사항 분석</div>
<p class="t-body-sm">핵심: 분산 시스템 경험, Java/Kotlin, 대용량 트래픽 처리</p></div>
<div class="card"><div class="card-title">커리어 볼트 차이</div>
<p class="t-body-sm">근거 4개 연결 가능 · Kafka 파이프라인 프로젝트가 요구사항과 일치</p></div>
<button class="button button-primary button-lg">지원 준비 시작</button>"""),
    "loading": header("공고 상세") + body(skels(3, 96)),
    "error": header("공고 상세") + body(error_state()),
}

# 7. vault
_vault_rows = """<div class="data-list">
<div class="data-list-row"><div class="data-list-main"><div class="t-body">Kafka 파이프라인 — 일 2억 건 처리</div><div class="card-meta">프로젝트 · 3일 전</div></div><span class="status-chip status-chip-verified">VERIFIED</span></div>
<div class="data-list-row"><div class="data-list-main"><div class="t-body">사내 해커톤 우승</div><div class="card-meta">수상 · 2025</div></div><span class="status-chip status-chip-pending">PENDING</span></div>
<div class="data-list-row"><div class="data-list-main"><div class="t-body">AWS SAA 자격</div><div class="card-meta">자격 · 2025</div></div><span class="status-chip status-chip-verified">VERIFIED</span></div>
</div>"""
S["vault"] = {
    "default": header("커리어 볼트", '<button class="button button-primary button-sm">{} 근거 추가</button>'.format(icon("plus")))
               + body(_vault_rows)
               + '<div class="toast">{} 근거가 검증됐어요</div>'.format(icon("check", 16)),
    "empty": header("커리어 볼트", '<button class="button button-primary button-sm">{} 근거 추가</button>'.format(icon("plus")))
             + body(empty_state("아직 근거가 없어요", "근거 추가")),
    "loading": header("커리어 볼트", '<button class="button button-primary button-sm">{} 근거 추가</button>'.format(icon("plus")))
               + body(skels(5, 52)),
    "error": header("커리어 볼트", '<button class="button button-primary button-sm">{} 근거 추가</button>'.format(icon("plus")))
             + body(error_state()),
}

# 8. project-chat
_pchat_stream = """<div class="chat-stream">
<div class="chat-stream-msg chat-stream-msg-ai">블루프린트를 작성했어요. Kafka 수집 → Spark 변환 → 적재 3단계입니다.</div>
<div class="card"><div class="card-title">블루프린트 — kafka-pipeline</div>
<div class="card-meta">producer.py · consumer.py · docker-compose.yml · README</div></div>
<div class="chat-stream-msg chat-stream-msg-user">실행 계획 보여줘</div>
<div class="approval-card">
<div class="approval-card-title">CLI 실행 승인</div>
<div class="approval-card-cmd">$ npm run test -- --coverage
디렉터리: ~/projects/kafka-pipeline</div>
<div class="approval-card-body">프로젝트 검증을 위해 테스트를 실행합니다.</div>
<div class="approval-card-actions">
<button class="button button-secondary button-sm">{x} 거부</button>
<button class="button button-primary button-sm">{c} 승인</button>
</div></div>
<div class="card"><div class="card-title">실행 상태</div>
<span class="status-chip status-chip-running">{l} RUNNING</span>
<div class="card-meta">npm run test · 12초 경과</div></div>
<div class="card"><div class="card-title">검증 결과</div>
<span class="status-chip status-chip-verified">{b} VERIFIED</span>
<div class="card-meta">테스트 42/42 통과 · 커버리지 81%</div></div>
</div>""".format(x=icon("x"), c=icon("check"), l=icon("loader-circle", 16), b=icon("badge-check", 16))

S["project-chat"] = {
    "default": header("Kafka 파이프라인", '<span class="status-chip status-chip-ready">프로젝트</span>')
               + _pchat_stream + COMPOSER,
    "loading": header("Kafka 파이프라인", '<span class="status-chip status-chip-ready">프로젝트</span>')
               + '<div class="chat-stream">{}</div>'.format(skels(4, 56)) + COMPOSER,
    "error": header("Kafka 파이프라인", '<span class="status-chip status-chip-ready">프로젝트</span>')
             + body(error_state()) + COMPOSER,
}

# 9. interview
_itv_rows = """<div class="data-list">
<div class="data-list-row"><div class="data-list-main"><div class="t-body">대용량 트래픽 처리 경험을 설명해 주세요</div><div class="card-meta">예상 질문 · 기술</div></div></div>
<div class="data-list-row"><div class="data-list-main"><div class="t-body">팀과 의견 충돌이 있었던 경험은?</div><div class="card-meta">예상 질문 · 협업</div></div></div>
</div>
<div class="card"><div class="card-title">STAR 답변 — Kafka 파이프라인</div>
<p class="t-body-sm">S: 일일 수집량 급증 · T: 실시간 파이프라인 설계 · A: Kafka+Spark 도입 · R: 지연 70% 단축</p></div>"""
S["interview"] = {
    "default": header("면접", '<button class="button button-primary button-sm">{} 면접 준비</button>'.format(icon("plus")))
               + body(_itv_rows),
    "empty": header("면접", '<button class="button button-primary button-sm">{} 면접 준비</button>'.format(icon("plus")))
             + body(empty_state("아직 면접 준비가 없어요", "면접 준비")),
    "loading": header("면접", '<button class="button button-primary button-sm">{} 면접 준비</button>'.format(icon("plus")))
               + body(skels(5, 52)),
    "error": header("면접", '<button class="button button-primary button-sm">{} 면접 준비</button>'.format(icon("plus")))
             + body(error_state()),
}

# 10. calendar
_cal = """<div class="card"><div class="card-title">2026년 3월</div>
<p class="t-body-sm">18(수) 삼성전자 서류 마감 · 20(금) 네이버 코딩테스트 · 24(화) 카카오 1차 면접</p></div>
<div class="data-list">
<div class="data-list-row"><div class="data-list-main"><div class="t-body">삼성전자 서류 마감</div><div class="card-meta">3월 18일 · D-0</div></div><span class="status-chip status-chip-error">D-DAY</span></div>
<div class="data-list-row"><div class="data-list-main"><div class="t-body">네이버 코딩테스트</div><div class="card-meta">3월 20일</div></div><span class="status-chip status-chip-ready">D-2</span></div>
<div class="data-list-row"><div class="data-list-main"><div class="t-body">카카오 1차 면접</div><div class="card-meta">3월 24일</div></div><span class="status-chip status-chip-ready">D-6</span></div>
</div>"""
S["calendar"] = {
    "default": header("캘린더", '<button class="button button-primary button-sm">{} 동기화</button>'.format(icon("calendar")))
               + body(_cal),
    "empty": header("캘린더", '<button class="button button-primary button-sm">{} 동기화</button>'.format(icon("calendar")))
             + body(empty_state("다가오는 일정이 없어요")),
    "loading": header("캘린더", '<button class="button button-primary button-sm">{} 동기화</button>'.format(icon("calendar")))
               + body(skels(4, 64)),
    "error": header("캘린더", '<button class="button button-primary button-sm">{} 동기화</button>'.format(icon("calendar")))
             + body(error_state()),
}

# 11. settings
_settings_form = """<div class="form">
<div class="form-section"><div class="form-section-title">계정</div>
<div class="form-row"><span class="form-row-label">이메일</span><span class="form-row-hint">cyjoon@gmail.com</span></div>
<div class="form-row"><span class="form-row-label">연결된 계정</span><span class="form-row-hint">Google · GitHub</span></div></div>
<div class="form-section"><div class="form-section-title">AI</div>
<div class="form-row"><span class="form-row-label">모델</span><span class="form-row-hint">OpenAI GPT</span></div></div>
<div class="form-section"><div class="form-section-title">요금제</div>
<div class="form-row"><span class="form-row-label">현재 플랜</span><span class="form-row-hint">Free</span></div></div>
<div class="form-section"><div class="form-section-title">외관</div>
<div class="form-row"><span class="form-row-label">테마</span><span class="form-row-hint">라이트</span></div></div>
<button class="button button-primary button-md">저장</button>
</div>"""
S["settings"] = {
    "default": header("설정") + body(_settings_form),
    "loading": header("설정") + body(skels(6, 40)),
    "error": header("설정") + body(error_state()),
}

STATES = {
    "login": ["default", "loading", "error"],
    "home": ["default", "loading", "error", "empty"],
    "chat": ["default", "empty", "loading", "error"],
    "documents": ["default", "empty", "loading", "error"],
    "doc-editor": ["default", "loading", "error"],
    "applications": ["default", "empty", "loading", "error"],
    "job-detail": ["default", "loading", "error"],
    "vault": ["default", "empty", "loading", "error"],
    "project-chat": ["default", "loading", "error"],
    "interview": ["default", "empty", "loading", "error"],
    "calendar": ["default", "empty", "loading", "error"],
    "settings": ["default", "loading", "error"],
}

NO_SIDEBAR = {"login"}


def main():
    SCREENS.mkdir(parents=True, exist_ok=True)
    written = []
    for slug, states in STATES.items():
        for st in states:
            inner = S[slug][st]
            html = page("{}-{}".format(slug, st), inner,
                        active=slug, bare=(slug in NO_SIDEBAR))
            p = SCREENS / "{}-{}.html".format(slug, st)
            p.write_text(html, encoding="utf-8")
            written.append(p.name)
    print("wrote {} files".format(len(written)))


if __name__ == "__main__":
    main()
