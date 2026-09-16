#!/usr/bin/env python3
"""Generate design/probes/final-preview.html — 4.5단계 최종 미리보기 (데스크톱)."""
import json, html

ICONS = json.load(open('/tmp/icons.json'))

def ic(name, size=20, sw=None):
    sw = sw or {16: '1.5', 20: '1.75', 24: '2', 48: '2'}.get(size, '1.75')
    inner = ICONS.get(name, '')
    return (f'<svg width="{size}" height="{size}" viewBox="0 0 24 24" fill="none" '
            f'stroke="currentColor" stroke-width="{sw}" stroke-linecap="round" '
            f'stroke-linejoin="round">{inner}</svg>')

def sidebar(active='', active_chat=None):
    funcs = [('file-text', '내 서류'), ('briefcase', '지원 관리'), ('archive', '커리어 볼트'),
             ('mic', '면접'), ('calendar', '캘린더')]
    chats = [('삼성전자 백엔드', '#3B82F6', ''), ('Kafka 파이프라인', '#9CA3AF', '프로젝트'),
             ('자소서 다듬기', '#9CA3AF', '')]
    f = ''.join(f'<div class="sb-i{" on" if active == n else ""}">{ic(i)}<span>{n}</span></div>' for i, n in funcs)
    c = ''.join(f'<div class="sb-i{" on" if active_chat == n else ""}"><span class="dot" style="background:{d}"></span><span style="flex:1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{n}</span>{"<span class=tag>프로젝트</span>" if t else ""}</div>' for n, d, t in chats)
    return f'''<div class="sb">
<div class="sb-top"><b>Push</b><span class="m">{ic('panel-left-close', 20)}</span></div>
<div class="sb-s">기능</div>{f}
<div class="sb-s">채팅 <span class="m" style="float:right">{ic('square-pen', 16)}</span></div>{c}
<div class="sb-i" style="margin-top:auto">{ic('settings')}<span>설정</span></div></div>'''

def hdr(title, actions=''):
    return f'<div class="ch"><h1>{title}</h1><div class="ch-a">{actions}</div></div>'

def btn(label, kind='pri', size='md', icon=None):
    i = ic(icon, 16) + ' ' if icon else ''
    return f'<button class="b b-{kind} b-{size}">{i}{label}</button>'

def chip(t, color=None):
    st = f' style="background:{color}22;color:{color};border-color:{color}44"' if color else ''
    return f'<span class="cp"{st}>{t}</span>'

def num(n):
    return f'<span class="num">{n}</span>'

# ---------- screen canvases ----------
def s_login():
    return f'''<div style="display:flex;align-items:center;justify-content:center;flex:1;background:linear-gradient(160deg,#fff 0%,var(--acs) 100%)">
<div style="width:400px;text-align:center">
<div style="font-size:28px;font-weight:700;margin-bottom:6px">Push</div>
<div class="m" style="margin-bottom:20px">공고 분석부터 서류 확정까지</div>
<div class="card" style="text-align:left;margin-bottom:16px">{num('①')}<p style="margin-top:8px">AI는 채팅에서 일하고, 근거는 커리어 볼트에 쌓여요. 검증된 프로젝트 결과가 곧 경력이 됩니다.</p></div>
{num('②')}{btn('Google로 계속', 'pri', 'lg')}<div style="height:8px"></div>
{num('③')}{btn('GitHub로 계속', 'sec', 'lg')}
</div></div>'''

def s_home():
    return f'''<div class="cv">
<div style="max-width:640px;margin:56px auto 0">
{num('①')}<div class="cmd"><input placeholder="무엇을 할까요? 공고 링크를 붙여넣거나 명령을 입력하세요"><span class="m">{ic('paperclip')}</span>{btn('실행', 'pri', 'sm', 'corner-down-left')}</div>
{num('②')}<div style="display:flex;gap:8px;margin:12px 0 40px"><span class="cp">공고 분석하기</span><span class="cp">자소서 초안</span><span class="cp">프로젝트 추천</span><span class="cp">면접 준비</span></div>
{num('③')}<div style="font-size:13px;font-weight:500;color:var(--mu);margin-bottom:12px">최근 작업</div>
{num('④')}<div class="cg">
<div class="card"><b>삼성전자 백엔드</b><p>자소서 초안 · 근거 12/12</p>{chip('READY', '#166534')}</div>
<div class="card"><b>네이버 플랫폼</b><p>공고 분석 완료</p>{chip('ANALYZED', '#2563EB')}</div>
<div class="card"><b>Kafka 파이프라인</b><p>프로젝트 실행 중</p>{chip('RUNNING', '#B45309')}</div>
</div></div></div>'''

def s_chat():
    return f'''<div class="cv" style="display:flex;flex-direction:column">
{num('①')}{hdr('삼성전자 백엔드', ic('ellipsis'))}
<div class="cs">
<div class="msg ai"><b>Push</b><p>공고 분석이 끝났어요. 요구 역량 8개 중 6개가 볼트 근거와 연결됐어요. 자소서 초안을 만들까요?</p></div>
<div class="msg me"><p>자소서 초안 만들어줘</p></div>
<div class="msg ai"><b>Push</b>
<div class="card" style="margin-top:4px">{num('②')}<b>자소서 초안 완성</b><p>삼성전자_자소서_v1.docx — 근거 12개 연결</p><div style="display:flex;gap:8px;margin-top:8px">{btn('문서로 열기', 'pri', 'sm')}{btn('다시 생성', 'sec', 'sm')}</div></div></div>
<div class="msg ai"><b>Push</b>
<div class="card" style="margin-top:4px;border-color:var(--ac)"><b>문서 확정 승인</b><p style="font-family:monospace;font-size:12px">삼성전자_자소서_v1.docx → PDF보내기</p><div style="display:flex;gap:8px;margin-top:8px">{btn('승인', 'pri', 'sm', 'check')}{btn('거부', 'sec', 'sm', 'x')}</div></div></div>
</div>
{num('③')}<div class="cmp"><input placeholder="메시지 입력 — 문서 수정은 여기서 요청"><span class="m" title="추론 설정">{ic('sliders-horizontal')}</span><span class="m">{ic('paperclip')}</span>{btn('보내기', 'pri', 'sm', 'send-horizontal')}</div>
</div>'''

def s_documents():
    return f'''<div class="cv">
{num('①')}{hdr('내 서류', btn('새 문서', 'pri', 'sm', 'plus'))}
{num('②')}<div class="cg">
<div class="card"><b>삼성전자_자소서_v1</b><p>자기소개서 · 2시간 전</p><p style="margin-top:4px;font-size:11px">탭하면 문서 페이지로 이동 →</p></div>
<div class="card"><b>이력서_2026</b><p>이력서 · 어제</p></div>
<div class="card"><b>포트폴리오</b><p>포트폴리오 · 3일 전</p></div>
</div>
<p class="m" style="margin-top:16px">문서를 누르면 하위 페이지(문서 편집)로 이동 — 이 화면에서 펼쳐지지 않음</p>
</div>'''

def s_editor():
    return f'''<div class="cv">
{num('①')}<div class="ch"><h1 style="font-size:20px">삼성전자_자소서_v1 <span class="m" style="font-size:12px;font-weight:400">← 내 서류</span></h1><div class="ch-a">{btn('PDF', 'sec', 'sm', 'file-down')}{btn('DOCX', 'sec', 'sm', 'file-down')}</div></div>
{num('②')}<div class="card" style="padding:28px;min-height:380px;font-size:15px;line-height:1.7">
<p style="font-weight:600;font-size:17px;margin-bottom:12px">지원 동기</p>
<p>대규모 트래픽 처리 경험을 삼성전자 백엔드 플랫폼에서 확장하고 싶습니다. 전 회사에서 일 2억 건의 이벤트를 처리하는 파이프라인을 운영하며…</p>
<p style="font-weight:600;font-size:17px;margin:20px 0 12px">주요 경험</p>
<p>Kafka 기반 CDC 스트림을 설계해 consumer lag를 100ms 이하로 유지했습니다…</p>
<p style="color:var(--mu);font-size:13px;margin-top:24px">순수 문서 편집기 — AI 수정은 채팅에서 요청</p>
</div>
</div>'''

def s_applications():
    return f'''<div class="cv">
{num('①')}{hdr('지원 관리', btn('공고 추가', 'pri', 'sm', 'plus'))}
{num('②')}<div class="cg">
<div class="card"><b>삼성전자 백엔드</b><p>플랫폼 개발 · D-7</p>{chip('READY', '#166534')}</div>
<div class="card"><b>네이버 플랫폼</b><p>서비스 개발 · D-14</p>{chip('ANALYZED', '#2563EB')}</div>
<div class="card"><b>카카오 인프라</b><p>서버 개발 · D-21</p>{chip('COLLECTED', '#6B7280')}</div>
<div class="card"><b>토스 코어</b><p>백엔드 · 마감 지남</p>{chip('SUBMITTED', '#6B7280')}</div>
</div>
<div style="margin-top:24px;display:flex;justify-content:center">{num('③')}<div class="toast">공고 분석이 완료됐어요 <button>보기</button></div></div>
</div>'''

def s_jobdetail():
    return f'''<div class="cv">
{num('①')}{hdr('삼성전자 — 백엔드 플랫폼 개발', ic('ellipsis'))}
{num('②')}<div class="card" style="margin-bottom:16px"><b>요구사항 분석</b><p>필수: Java/Kotlin, 대용량 트래픽, MSA. 우대: Kafka, k8s. 마감 D-7.</p><p style="margin-top:8px">{chip('근거 연결 12/12', '#166534')}</p></div>
{num('③')}<div class="card" style="margin-bottom:16px"><b>커리어 볼트 차이</b><p>충족 6개 · 부분 1개(Kafka — 프로젝트로 보완 가능) · 부족 1개(k8s 운영)</p><p style="margin-top:8px">{chip('부족: k8s', '#DC2626')}{chip('보완 후보: Kafka 파이프라인', '#B45309')}</p></div>
{num('④')}{btn('지원 준비 시작', 'pri', 'lg')}
</div>'''

def s_vault():
    rows = ''.join(f'<div class="li"><span style="flex:1"><b>{t}</b><span class="m"> — {d}</span></span>{chip(c, col)}<span class="m">{ic("ellipsis")}</span></div>' for t, d, c, col in [
        ('Kafka 스트리밍 파이프라인', 'GitHub 프로젝트 · 검증됨', 'VERIFIED', '#166534'),
        ('대용량 트래픽 처리', '이전 회사 · 수치 근거 있음', 'VERIFIED', '#166534'),
        ('k8s 운영 경험', '미검증 — 근거 필요', 'PENDING', '#B45309')])
    return f'''<div class="cv">
{num('①')}{hdr('커리어 볼트', btn('근거 추가', 'pri', 'sm', 'plus'))}
{num('②')}<div class="card" style="padding:0">{rows}</div>
<div style="margin-top:24px;display:flex;justify-content:center">{num('③')}<div class="toast">프로젝트 결과가 근거로 추가됐어요</div></div>
</div>'''

def s_project():
    return f'''<div class="cv" style="display:flex;flex-direction:column">
{num('①')}{hdr('Kafka 파이프라인 ' + chip('프로젝트'), ic('ellipsis'))}
<div class="cs">
<div class="msg ai"><b>Push</b>
<div class="card" style="margin-top:4px"><b>블루프린트: 실시간 이벤트 파이프라인</b><p>목표: Kafka로 CDC 스트림 처리. 부족 역량 "Kafka" 커버. 예상 2주.</p></div></div>
<div class="msg me"><p>이걸로 진행해</p></div>
<div class="msg ai"><b>Push</b>
<div class="card" style="margin-top:4px;border-color:var(--ac)"><b>실행 승인 — CLI</b>
<p style="font-family:monospace;font-size:12px;background:var(--s2);padding:8px;border-radius:6px;margin:6px 0">$ docker-compose up kafka<br>dir: ~/projects/kafka-pipeline<br>prompt: scaffold consumer…</p>
<div style="display:flex;gap:8px">{btn('승인', 'pri', 'sm', 'check')}{btn('거부', 'sec', 'sm', 'x')}</div></div></div>
<div class="msg ai"><b>Push</b>
<div class="card" style="margin-top:4px"><b>실행 상태</b><p>{chip('RUNNING', '#B45309')} → {chip('VERIFIED', '#166534')} · 검증: consumer lag &lt; 100ms</p><div style="margin-top:8px">{btn('근거로 추가', 'sec', 'sm', 'link-2')}</div></div></div>
</div>
{num('③')}<div class="cmp"><input placeholder="프로젝트 채팅 — 실행·검증은 이 스트림에서"><span class="m" title="추론 설정">{ic('sliders-horizontal')}</span><span class="m">{ic('paperclip')}</span>{btn('보내기', 'pri', 'sm', 'send-horizontal')}</div>
</div>'''

def s_interview():
    rows = ''.join(f'<div class="li"><span style="flex:1">{t}</span>{chip(c)}</div>' for t, c in [
        ('대규모 트래픽 장애 대응 경험을 말해주세요', 'STAR'),
        ('Kafka와 RabbitMQ 차이는?', '기술'),
        ('왜 삼성전자인가요?', '동기')])
    return f'''<div class="cv">
{num('①')}{hdr('면접', btn('면접 준비', 'pri', 'sm', 'plus'))}
{num('②')}<div class="card" style="padding:0;margin-bottom:16px">{rows}</div>
{num('③')}<div class="card"><b>STAR 답변 — 트래픽 장애</b><p>S: 피크 트래픽 중 DB 커넥션 고갈 · T: 30분 내 복구 · A: 풀 사이즈 조정+슬로우 쿼리 제거 · R: p99 40% 개선</p></div>
</div>'''

def s_calendar():
    days = ''.join(f'<div class="cal-d{ " on" if d in (12, 19) else ""}">{d}</div>' for d in range(1, 29))
    rows = ''.join(f'<div class="li"><span style="flex:1"><b>{t}</b><span class="m"> — {d}</span></span></div>' for t, d in [
        ('삼성전자 서류 마감', '3월 12일'), ('네이버 코딩테스트', '3월 19일'), ('카카오 면접', '3월 24일')])
    return f'''<div class="cv">
{num('①')}{hdr('캘린더', btn('Google 동기화', 'sec', 'sm'))}
{num('②')}<div class="card" style="margin-bottom:16px"><b>2026년 3월</b><div class="cal">{days}</div></div>
{num('③')}<div class="card" style="padding:0">{rows}</div>
</div>'''



def s_settings():
    fld = lambda l, v: f'<div class="fld"><span class="fl">{l}</span><span class="fv">{v}</span></div>'
    return f'''<div class="cv">
{num('①')}{hdr('설정')}
{num('②')}<div class="card" style="margin-bottom:16px">
<div class="fs">계정</div>{fld('이메일', 'user@gmail.com')}{fld('연동', 'Google 캘린더 · GitHub')}
<div class="fs">AI</div>{fld('모델', 'OpenAI')}
<div class="fs">요금제</div>{fld('플랜', 'Free')}
<div class="fs">외관</div>{fld('테마', '라이트')}{fld('언어', '한국어')}
</div>
{num('③')}{btn('저장', 'pri', 'md', 'check')}
</div>'''

SCREENS = [
    ('login', '로그인', s_login(), '① Card(소개) · ② Button(Google) · ③ Button(GitHub)', None),
    ('home', '홈(명령창)', s_home(), '① CommandInput · ② SuggestChips · ③ 섹션 헤더 · ④ CardGrid', 'home'),
    ('chat', '회사별 채팅', s_chat(), '① CanvasHeader · ② ChatStream+ApprovalCard · ③ Composer', 'chat'),
    ('documents', '내 서류', s_documents(), '① CanvasHeader · ② DocCard×n (탭 → 문서 편집 하위 페이지)', 'documents'),
    ('doc-editor', '문서 편집 (하위 페이지)', s_editor(), '① CanvasHeader(제목+PDF/DOCX) · ② Editor(순수 편집)', 'documents'),
    ('applications', '지원 관리', s_applications(), '① CanvasHeader · ② CardGrid+StatusChip · ③ Toast', 'applications'),
    ('job-detail', '공고 상세', s_jobdetail(), '① CanvasHeader · ② Card(요구사항) · ③ Card(차이) · ④ Button', 'job'),
    ('vault', '커리어 볼트', s_vault(), '① CanvasHeader · ② DataList · ③ Toast', 'vault'),
    ('project-chat', '프로젝트 채팅', s_project(), '① CanvasHeader+태그 · ② ChatStream(블루프린트·CLI승인·상태) · ③ Composer', 'project'),
    ('interview', '면접', s_interview(), '① CanvasHeader · ② DataList · ③ Card(STAR)', 'interview'),
    ('calendar', '캘린더', s_calendar(), '① CanvasHeader · ② Card(달력) · ③ DataList', 'calendar'),
    ('settings', '설정', s_settings(), '① CanvasHeader · ② Form · ③ Button(저장)', 'settings'),
]

ACTIVE = {'home': ('', ''), 'chat': ('', '삼성전자 백엔드'), 'project': ('', 'Kafka 파이프라인'),
          'documents': ('내 서류', ''), 'applications': ('지원 관리', ''), 'job': ('지원 관리', ''),
          'vault': ('커리어 볼트', ''), 'interview': ('면접', ''), 'calendar': ('캘린더', ''),
          'settings': ('', '')}

def frame(slug, title, canvas, key):
    if key is None:
        inner = f'<div class="win-cv">{canvas}</div>'
    else:
        fa, fc = ACTIVE[key]
        inner = f'<div style="display:flex;flex:1;min-height:0">{sidebar(fa, fc)}<div style="flex:1;display:flex;flex-direction:column;min-width:0">{canvas}</div></div>'
    return f'''<div class="win">
<div class="win-h"><i style="background:#FF5F57"></i><i style="background:#FEBC2E"></i><i style="background:#28C840"></i><span style="margin-left:8px;font-size:12px;color:var(--mu)">{html.escape(title)}</span></div>
{inner}</div>'''

sections = []
for n, (slug, title, canvas, spec, key) in enumerate(SCREENS, 1):
    sections.append(f'''
<section class="scr" id="s-{slug}">
<div class="scr-h"><b>{n}. {html.escape(title)}</b><span class="m" style="font-weight:400">{html.escape(spec)}</span></div>
<div class="scr-b">
<div>{frame(slug, title, canvas, key)}
<div class="th"><span class="th-i">empty</span><span class="th-i">loading</span><span class="th-i">error</span></div></div>
<div class="panel">
<div class="p-t">화면 구성 <span class="m">(빼거나 바꿀 수 있음)</span></div>
<div class="p-spec">{html.escape(spec)}</div>
<div class="p-btns">
<button class="b b-sec b-sm" onclick="react('{slug}','ok')">괜찮아요 👍</button>
<button class="b b-sec b-sm" onclick="react('{slug}','fix')">고칠 게 있어요 🤔</button>
</div>
<textarea id="t-{slug}" placeholder="고칠 내용 — 예: ③ 빼요 / 카드 순서 / 문구" style="display:none"></textarea>
<div class="p-save" id="st-{slug}"></div>
</div></div>
</section>''')

page = '''<!doctype html>
<meta charset="utf-8">
<html><head><style>
:root{--bg:#FFFFFF;--s1:#F5F5F7;--s2:#E9E9EE;--tx:#111827;--mu:#6B7280;--bd:#E5E7EB;--ac:#2563EB;--acp:#1D4ED8;--acs:rgba(37,99,235,.10);--dg:#DC2626;--ok:#166534;--wn:#B45309;--frame-w:1440px}
*{margin:0;box-sizing:border-box;font-family:Pretendard,-apple-system,"Apple SD Gothic Neo",system-ui,sans-serif;color:var(--tx)}
body{background:var(--s2);padding:24px}
.sum{max-width:1180px;margin:0 auto 32px;background:#fff;border-radius:12px;padding:24px;box-shadow:0 4px 12px rgba(0,0,0,.08)}
.sum h1{font-size:24px;font-weight:600;margin-bottom:8px}
.sum p{font-size:14px;color:var(--mu);line-height:1.6}
.scr{max-width:1180px;margin:0 auto 40px}
.scr-h{display:flex;justify-content:space-between;align-items:baseline;margin-bottom:12px;font-size:15px}
.scr-h b{font-size:17px;font-weight:600}
.scr-b{display:flex;gap:20px;align-items:flex-start}
.win{width:860px;background:var(--bg);border-radius:12px;box-shadow:0 4px 12px rgba(0,0,0,.08);overflow:hidden;border:1px solid var(--bd);display:flex;flex-direction:column;height:560px}
.win-h{background:var(--s1);border-bottom:1px solid var(--bd);padding:8px 12px;display:flex;align-items:center;gap:6px}
.win-h i{width:10px;height:10px;border-radius:50%;display:inline-block}
.win-cv{flex:1;display:flex}
.sb{width:150px;background:var(--s1);border-right:1px solid var(--bd);padding:8px 6px;display:flex;flex-direction:column;gap:1px;flex-shrink:0}
.sb-top{display:flex;justify-content:space-between;align-items:center;padding:4px 10px 8px;font-size:14px}
.sb-s{font-size:10px;color:var(--mu);padding:8px 10px 2px;font-weight:500}
.sb-i{display:flex;align-items:center;gap:8px;padding:0 10px;height:30px;border-radius:6px;font-size:12.5px;cursor:default}
.sb-i.on{background:var(--acs);color:var(--ac)}
.dot{width:7px;height:7px;border-radius:50%;flex-shrink:0}
.tag{display:inline-block;font-size:9px;padding:1px 6px;border-radius:9999px;background:var(--acs);color:var(--ac);font-weight:500;flex-shrink:0}
.cv{flex:1;padding:24px;overflow:hidden;position:relative}
.ch{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px}
.ch h1{font-size:22px;font-weight:600;display:inline-flex;gap:8px;align-items:center}
.ch-a{color:var(--mu);display:flex;gap:8px;align-items:center}
.cmd{display:flex;align-items:center;gap:8px;border:1px solid var(--bd);border-radius:12px;padding:6px 6px 6px 16px;background:#fff;box-shadow:0 1px 2px rgba(0,0,0,.06);margin-top:8px}
.cmd input{flex:1;border:none;outline:none;font-size:15px;height:36px}
.cmp{display:flex;align-items:center;gap:8px;border:1px solid var(--bd);border-radius:12px;padding:6px 6px 6px 16px;background:#fff;margin:0 24px 20px}
.cmp input{flex:1;border:none;outline:none;font-size:14px;height:32px}
.cs{flex:1;overflow:hidden;padding:0 24px;display:flex;flex-direction:column;gap:10px}
.msg{max-width:75%;font-size:14px}
.msg b{font-size:12px;color:var(--mu);font-weight:500}
.msg p{margin-top:2px;line-height:1.5}
.msg.ai{align-self:flex-start}
.msg.me{align-self:flex-end;background:var(--acs);color:var(--ac);border-radius:12px;padding:8px 12px}
.card{background:var(--s1);border:1px solid var(--bd);border-radius:12px;padding:14px;box-shadow:0 1px 2px rgba(0,0,0,.06)}
.card b{font-size:14px;font-weight:600;display:block}
.card p{font-size:12.5px;color:var(--mu);line-height:1.5;margin-top:2px}
.cg{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-top:8px}
.cp{display:inline-block;font-size:12px;padding:4px 10px;border-radius:9999px;border:1px solid var(--bd);background:#fff;color:var(--mu);margin-right:6px;margin-top:8px}
.b{border:none;cursor:pointer;border-radius:8px;font-weight:500;display:inline-flex;align-items:center;justify-content:center;gap:6px;font-family:inherit}
.b-pri{background:var(--ac);color:#fff}.b-pri:hover{background:var(--acp)}
.b-sec{background:#fff;border:1px solid var(--bd)}.b-sec:hover{background:var(--s1)}
.b-gho{background:transparent;color:var(--ac)}.b-gho:hover{background:var(--acs)}
.b-dan{background:var(--dg);color:#fff}
.b-sm{height:30px;padding:0 12px;font-size:12.5px}
.b-md{height:36px;padding:0 16px;font-size:14px}
.b-lg{height:44px;padding:0 20px;font-size:15px;width:100%}
.li{display:flex;align-items:center;gap:10px;padding:10px 16px;border-bottom:1px solid var(--bd);font-size:13.5px}
.li:last-child{border:none}
.m{color:var(--mu);font-size:12px}
.num{display:inline-flex;width:22px;height:22px;border-radius:50%;background:var(--ac);color:#fff;font-size:12px;font-weight:600;align-items:center;justify-content:center;flex-shrink:0}
.dlg{background:#fff;border-radius:12px;box-shadow:0 4px 12px rgba(0,0,0,.08);width:360px;padding:16px;border:1px solid var(--bd)}
.dlg b{font-size:15px}
.toast{background:#1F2937;color:#F9FAFB;font-size:12.5px;border-radius:8px;padding:8px 14px;display:inline-flex;gap:12px;align-items:center}
.toast button{background:none;border:none;color:#93C5FD;font-size:12.5px;cursor:pointer;font-weight:500}
.cal{display:grid;grid-template-columns:repeat(7,1fr);gap:4px;margin-top:10px}
.cal-d{text-align:center;font-size:11.5px;padding:6px 0;border-radius:6px;color:var(--mu)}
.cal-d.on{background:var(--acs);color:var(--ac);font-weight:600}
.fs{font-size:12px;font-weight:600;color:var(--mu);padding:12px 0 6px;border-bottom:1px solid var(--bd)}
.fld{display:flex;justify-content:space-between;padding:8px 0;font-size:13.5px;border-bottom:1px solid var(--s2)}
.fl{color:var(--mu)}
.panel{width:280px;flex-shrink:0;background:#fff;border:1px solid var(--bd);border-radius:12px;padding:16px}
.p-t{font-size:13px;font-weight:600;margin-bottom:8px}
.p-spec{font-size:12px;color:var(--mu);line-height:1.6;margin-bottom:12px}
.p-btns{display:flex;flex-direction:column;gap:6px}
.panel textarea{width:100%;height:64px;border:1px solid var(--bd);border-radius:8px;padding:8px;font-size:12.5px;font-family:inherit;margin-top:8px;resize:vertical}
.p-save{font-size:11.5px;color:var(--ok);margin-top:6px}
.th{display:flex;gap:8px;margin-top:10px}
.th-i{font-size:11px;color:var(--mu);border:1px solid var(--bd);border-radius:6px;padding:4px 10px;background:#fff}
.foot{max-width:1180px;margin:0 auto 60px;background:#fff;border-radius:12px;padding:24px;box-shadow:0 4px 12px rgba(0,0,0,.08);text-align:center}
.foot textarea{width:100%;max-width:560px;height:60px;border:1px solid var(--bd);border-radius:8px;padding:10px;font-size:13px;font-family:inherit;resize:vertical;margin:12px 0}
.go{background:var(--ac);color:#fff;border:none;border-radius:8px;height:44px;padding:0 32px;font-size:15px;font-weight:600;cursor:pointer;opacity:.4}
.go.on{opacity:1}
</style></head><body>

<div class="sum">
<h1>최종 미리보기</h1>
<p>이렇게 정했어요: <b>밝기</b> 라이트 중립 · <b>밀도</b> 보통 · <b>형태</b> 부드러운(radius 8/12) · <b>강조색</b> 파랑 #2563EB · <b>글꼴</b> Pretendard 균일 · <b>내비</b> 좌측 사이드바</p>
<p style="margin-top:8px">이 화면들은 OpenDesign으로 만들기 전 예상 모습이에요. 세부 간격은 artifact에서 규칙대로 맞춰져요. 화면마다 "괜찮아요" 또는 고칠 내용을 남겨주세요.</p>
</div>
''' + '\n'.join(sections) + '''
<div class="foot">
<h1 style="font-size:20px;font-weight:600">전체 의견</h1>
<textarea id="t-overall" placeholder="전체적으로 고칠 점 (선택)"></textarea>
<div><button class="go" id="go" onclick="go()">이대로 만들어 주세요</button></div>
<div class="m" id="go-st" style="margin-top:8px">모든 화면에 "괜찮아요"를 누르면 활성화돼요</div>
</div>

<script>
const LS='push-hubdb';const load=()=>{try{return JSON.parse(localStorage.getItem(LS))||{}}catch(e){return{}}};
let db=null;if(window.claude&&claude.use)claude.use('db').then(d=>db=d);
let okSet=new Set();
async function save(k,v){const a=load();a['feedback/'+k]=v;localStorage.setItem(LS,JSON.stringify(a));if(db)db.doc('feedback/'+k).set(v)}
async function react(slug,r){
  const t=document.getElementById('t-'+slug);
  if(r==='fix'){t.style.display='block';t.focus()}
  else{t.style.display='none';okSet.add(slug)}
  await save('screen-'+slug,{unit:'screen',label:slug,reaction:r,text:t?t.value:'',updatedAt:new Date().toISOString()});
  document.getElementById('st-'+slug).textContent=r==='ok'?'저장됨 ✓':'고칠 내용을 적고 저장';
  upd();
}
async function go(){
  const overall=document.getElementById('t-overall').value;
  await save('preview-overall',{unit:'overall',text:overall,updatedAt:new Date().toISOString()});
  await save('preview-go',{unit:'preview',label:'확정',text:'go',updatedAt:new Date().toISOString()});
  document.getElementById('go-st').textContent='확정 저장됨 ✓';
}
function upd(){const all=''' + str(len(SCREENS)) + ''';document.getElementById('go').className=okSet.size>=all?'go on':'go'}
document.querySelectorAll('textarea[id^=t-]').forEach(t=>{if(t.id==='t-overall')return;t.addEventListener('change',async()=>{await save('screen-'+t.id.slice(2),{unit:'screen',label:t.id.slice(2),reaction:'fix',text:t.value,updatedAt:new Date().toISOString()});document.getElementById('st-'+t.id.slice(2)).textContent='저장됨 ✓'})});
</script>
</body></html>'''

# restore saved states
page = page.replace("</script>", '''
(()=>{const a=load();const slugs=''' + str([s[0] for s in SCREENS]) + ''';
for(const s of slugs){const d=a['feedback/screen-'+s];if(!d)continue;
if(d.reaction==='ok'){okSet.add(s);document.getElementById('st-'+s).textContent='저장됨 ✓'}
else{const t=document.getElementById('t-'+s);t.style.display='block';t.value=d.text||'';document.getElementById('st-'+s).textContent='저장됨 ✓'}}
if(a['feedback/preview-go'])document.getElementById('go-st').textContent='확정 저장됨 ✓';
upd()})();
</script>''')

open('design/probes/final-preview.html', 'w').write(page)
print('written', len(page))
