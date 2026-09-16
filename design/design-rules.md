status: confirmed
confirmed_at: 2026-03-18
preview: design/probes/rules-preview.html

# Design Rules — Push (데스크톱, macOS first)

모바일 기본값을 데스크톱으로 옮긴 예외: pressed → hover+active, 탭 영역 → 클릭 영역+커서, 바텀시트 → 모달 다이얼로그, 탭바 → 좌측 사이드바, safe-area → 창 최소 크기.

## A. 토큰

| 키 | 값 | 출처 |
|---|---|---|
| color.bg | #FFFFFF | 축 1: A |
| color.surface-1 | #F5F5F7 | 축 1: A |
| color.surface-2 | #E9E9EE | 축 1: A |
| color.text | #111827 | 축 1: A |
| color.text-muted | #6B7280 | 축 1: A |
| color.border | #E5E7EB | 축 1: A |
| color.accent | #2563EB | 축 4: A |
| color.accent-pressed | #1D4ED8 | 자동 |
| color.accent-soft | rgba(37,99,235,.10) | 자동 |
| color.danger | #DC2626 | 고정 |
| color.success | #166534 | 3단계 레퍼런스 (단계·검증 상태용 추가) |
| color.warn | #B45309 | 3단계 레퍼런스 (차이 분석·경고용 추가) |
| color.overlay | rgba(0,0,0,.5) | 고정 |
| space.scale | 4 / 8 / 12 / 16 / 24 / 32 / 48 | 축 2: B |
| space.screen-padding | 좌우 24 (데스크톱 캔버스) | 축 2: B + 데스크톱 |
| space.section | 24 | 축 2: B |
| space.card-padding | 16 | 축 2: B |
| radius | sm 4 / md 8 / lg 12 / xl 16 / full 9999 | 축 3: B |
| shadow | sm 0 1px 2px rgba(0,0,0,.06) / md 0 4px 12px rgba(0,0,0,.08) | 축 3: B |
| font.family | "Pretendard", -apple-system, "Apple SD Gothic Neo", system-ui, sans-serif | 축 5: A |
| type.roles | display 28/700 · h1 24/600 · h2 20/600 · h3 17/600 · body 15/400 · body-sm 14/400 · caption 12/400 · label 13/500 | 축 5: A |
| z.scale | base 0 · sticky 100 · sidebar 200 · overlay 300 · dialog 500 · toast 600 | 고정 |
| motion | 200ms ease-out. 다이얼로그 250ms | 고정 |
| device.frame | 데스크톱 1440×900 기준, 최소 창 1024×640 (모바일 폭 390 제외 — 데스크톱 전용) | 1단계 플랫폼 예외 |
| safe-area | (미사용 — 데스크톱). 창 최소 1024×640에서 레이아웃 유지 | 고정 |
| tap.min | 44×44 유지(데스크톱도 동일 기준), 행 높이 36 이상. 커서 pointer + hover 반응 필수 | 고정 |
| platform | macOS first (Tauri), Windows 동일 코드베이스 | 1단계 |

## B. 컴포넌트 규칙

### B1. 버튼

| 키 | 값 | 출처 |
|---|---|---|
| button.sizes | sm 36h/px12/text13 · md 44h/px16/text14 · lg 52h/px20/text15 | 1단계 (데스크톱 조정) |
| button.radius | radius.md (8) | 축 3: B |
| button.variants | primary(accent bg, 흰 글자) · secondary(흰 bg, border) · ghost(투명, accent 글자) · danger | 기본값 |
| button.states | default · hover(accent-soft 배경) · pressed(12% 명도 변화) · selected(accent-soft bg + accent 1px border) · disabled(opacity .4) · loading(스피너, 폭 유지) | 1단계 (데스크톱, hover 추가) |
| button.row-rule | 같은 줄 같은 size·radius. 두 개면 secondary 왼쪽, primary 오른쪽 | 기본값 |
| button.text | 한 줄. 넘치면 문구 줄임 | 기본값 |
| button.primary-per-screen | 화면당 primary 1개. 위치는 화면 주요 작업 영역 (하단 고정 바 없음 — 데스크톱) | 2단계 |
| button.duplicate | 같은 동작 이중 배치 금지 | 기본값 |
| button.in-input | 실행·보내기 버튼은 입력창 안 우측 (사용자 원문: "버튼은 Input 안에다가 둬") | 2단계 피드백 |

### B2. 아이콘

| 키 | 값 | 출처 |
|---|---|---|
| icon.set | lucide 단일, `design/icons.md` 허용 목록만 | 기본값 |
| icon.sizes | 16 / 20 / 24 / 48 (빈 상태 전용) | 기본값 |
| icon.stroke | 16→1.5 / 20→1.75 / 24→2 | 기본값 |
| icon.color | currentColor | 기본값 |
| icon-button.hit | 클릭 영역 32×32, 시각 아이콘 20 | 1단계 (데스크톱) |
| icon-button.name | 접근성 라벨 필수 | 기본값 |
| icon.overflow-menu | 앱바 액션 최대 2개 노출, 나머지 더보기 | 기본값 |
| tap.feedback | hover 시각 반응 필수 (데스크톱 커서) | 1단계 (데스크톱) |

### B3. 이미지·썸네일

| 키 | 값 | 출처 |
|---|---|---|
| thumbnail.spec | (미사용 — 이미지 중심 앱 아님) | 1단계 |
| image.placeholder | 로딩 surface-2 스켈레톤 / 실패 image-off 24 | 기본값 |

### B4. 텍스트·문구

| 키 | 값 | 출처 |
|---|---|---|
| text.role-lock | 같은 역할 = 같은 type.role | 기본값 |
| text.truncate | 카드 제목 2줄, 목록 제목 1줄, 설명 3줄 | 기본값 |
| copy.user-language | 내부 명칭 금지, 사용자 언어 | 기본값 |
| copy.error | 이유 + 재시도 조건 | 기본값 |

### B5. 레이아웃·스크롤·고정 요소

| 키 | 값 | 출처 |
|---|---|---|
| layout.shell | 좌측 사이드바(240) + 캔버스. 우측 컨텍스트 패널 없음 (1단계 확정) | 1단계 |
| layout.sidebar | 섹션: 기능 / 채팅. 프로젝트 채팅은 채팅 목록 안에 '프로젝트' 태그 (2단계: "프로젝트 탭 완전 제거") | 2단계 피드백 |
| layout.sidebar-item | 행 36, dot 8 + 라벨 body-sm, 활성 accent-soft 배경 | 추가 |
| layout.sidebar-collapse | 창 1024 미만이면 사이드바 아이콘-only로 접힘 (닫기 아이콘 — Craft 참고) | 1단계 + 레퍼런스 |
| layout.left-edge | 캔버스 모든 섹션 좌측 = space.screen-padding (24) | 1단계 (데스크톱) |
| layout.surface-tiers | bg → surface-1 → surface-2. 3단 이상 금지 | 기본값 |
| scroll.single | 세로 스크롤 컨테이너 화면당 1개 (채팅 스트림만 스크롤, 입력창 고정) | 기본값 |
| fixed.tab-bar | (미사용 — 사이드바로 대체) | 1단계 |
| fixed.composer | 채팅 입력창은 캔버스 하단 고정, 본문 위에 뜸 | 추가 |

### B6. 시트·다이얼로그·피드백

| 키 | 값 | 출처 |
|---|---|---|
| sheet.sizes | (미사용 — 바텀시트 없음). 모달 다이얼로그 사용 | 1단계 (데스크톱) |
| dialog | 확인·경고·승인. 폭 480, 가운데, 제목 h3 + 본문 + 버튼 2개 | 1단계 (데스크톱) |
| dialog.approval | 승인은 다이얼로그가 아니라 채팅 안 인라인 카드 (CLI 실행·문서 확정·볼트 반영) | 2단계 피드백 |
| app-bar | (미사용 — 창 타이틀바 + 캔버스 헤더) | 1단계 (데스크톱) |
| help.inline | 설명은 필드 아래 caption. 툴팁은 hover 시 아이콘 버튼에만 허용 | 1단계 (데스크톱) |
| snackbar | 화면 하단 중앙 토스트. 4초, 액션 1개 | 기본값 |
| layer.order | z.scale 준수 | 기본값 |

### B7. 상태 커버리지

| 키 | 값 | 출처 |
|---|---|---|
| state.required | 모든 화면·목록·카드에 초기·빈·로딩·성공·실패·비활성 6상태 | 기본값 |
| state.empty | 아이콘 48 + 안내 1줄 + primary 버튼. 단 채팅 빈 상태는 문구 없이 제안 칩만 (2단계: "비어있다는 건 노출 안함") | 2단계 피드백 |
| state.loading | 스켈레톤 = 실제 레이아웃 형태와 일치 (그리드면 그리드 스켈레톤). 풀스크린 스피너 금지 | 2단계 피드백 |
| state.error | 이유 + 재시도 버튼. 네트워크 오류는 토스트 | 기본값 |
| state.chat-status | 채팅 목록 항목에 실행 상태 점 (Aside 참고) | 3단계 레퍼런스 |

### B8. 변경 범위

| 키 | 값 | 출처 |
|---|---|---|
| change.scope | 요청된 결함만 고침. 조작 제거 금지 | 고정 |
| change.relation | 위치 요청은 대상·기준·순서 확인 후 실행 | 고정 |
| change.propagate | 컴포넌트 고치면 모든 인스턴스 화면 재확인 | 고정 |
| change.no-dup | 새 공통 요소 추가 전 중복 확인 | 고정 |

## C. 프로젝트 전용 규칙

| 키 | 값 | 출처 |
|---|---|---|
| chat.central | AI 상호작용은 채팅에서만. 에디터 인라인 AI·버블 금지 (사용자 원문: "AI는 채팅에서만 사용 가능함") | 2단계 피드백 |
| chat.project-group | 프로젝트 = 채팅 그룹. 별도 화면·탭 없음. 승인·실행·검증 전부 해당 채팅 안 카드 | 2단계 피드백 |
| editor.plain | 내 서류는 순수 문서 편집기 + PDF/DOCX 출력 버튼만 | 2단계 피드백 |
| home.command | 홈 = 중앙 대형 명령 입력창(버튼 내부) + 입력 아래 제안 액션 + 최근 작업 카드 그리드 | 2·3단계 |
| card.doc-color | (폐기 — 4.5단계: "쓸때없는 색 제거". DocCard는 단색) | 3단계 레퍼런스 |
| approval.inline | 모든 승인(애플리케이션 제출·CLI 실행·문서 확정·볼트 반영)은 채팅 인라인 카드. 명령·디렉터리·프롬프트 원문 표시 | PRD + 2단계 |
| evidence.gate | 근거 없는 기술·수치·경험 생성 금지 — UI에 근거 연결 수 표시 | 1단계 (PRD) |
