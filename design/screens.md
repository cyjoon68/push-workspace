status: confirmed
confirmed_at: 2026-03-18

# Screens

구성은 위→아래(또는 좌→우) 순서. 원문자 번호는 최종 미리보기 프레임의 라벨과 같다 (화면당 최대 5개).
컴포넌트 이름은 od-builder의 컴포넌트 클래스 이름과 정확히 같아야 한다.

**공통 셸**: 로그인을 제외한 모든 화면은 `Sidebar` + 캔버스. 아래 "구성"에는 캔버스 안 요소만 적고 Sidebar는 생략한다.

컴포넌트 어휘 (데스크톱):
- `Sidebar` — 좌측 220px. 기능 섹션 + 채팅 섹션(상태 점·프로젝트 태그) + 하단 설정
- `CanvasHeader` — 캔버스 상단. 제목 h1 + 우측 액션 최대 2개
- `CommandInput` — 대형 명령 입력창. 실행 버튼 내부 우측 + 첨부 아이콘
- `Composer` — 채팅 하단 고정 입력창. 보내기 버튼 내부 우측
- `ChatStream` — 메시지 스트림. AI 메시지 + 사용자 메시지 + 카드 삽입
- `SuggestChips` — 제안 액션 칩 행 (빈 채팅·홈 입력 아래)
- `Card` — surface-1 카드. radius 12
- `CardGrid` — 3열 카드 그리드 (최근 작업·지원 관리)
- `DocCard` — 문서 카드 (단색 — 컬러 띠 제거됨, 4.5단계 피드백)
- `Editor` — 문서 편집 본문 (순수 편집 — AI 없음)
- `ApprovalCard` — 채팅 안 승인 카드. 내용 + 승인/거부 버튼 2개
- `StatusChip` — 단계·상태 칩 (READY/RUNNING/VERIFIED 등)
- `DataList` — 행 목록 (커리어 볼트·면접·반복 작업)
- `Form` — 설정 폼 필드 행
- `EmptyState` — 아이콘 48 + 안내 + 버튼 (채팅 빈 상태는 SuggestChips만)
- `Dialog` — 확인·경고 모달 480
- `Toast` — 하단 중앙 토스트

| 순번 | 화면 | slug | 구성 (위→아래) | 상태 프레임 |
|---|---|---|---|---|
| 1 | 로그인 | login | ① Card(소개) · ② Button(Google로 계속) · ③ Button(GitHub로 계속) | default·loading·error |
| 2 | 홈(명령창) | home | ① CommandInput(실행 버튼 내부) · ② SuggestChips · ③ CanvasHeader(최근 작업) · ④ CardGrid(최근 작업 카드×3) | default·loading·error·empty |
| 3 | 회사별 채팅 | chat | ① CanvasHeader(채팅 제목+더보기) · ② ChatStream(메시지) · ApprovalCard(문서 확정 승인) · ③ Composer(추론 설정 아이콘 — UltraResume 토글 포함) | default·empty(제안칩만)·loading·error |
| 4 | 내 서류 | documents | ① CanvasHeader(내 서류+새 문서) · ② DocCard×n(탭 → 문서 편집 하위 페이지) | default·empty·loading·error |
| 4-1 | 문서 편집 | doc-editor | ① CanvasHeader(문서명+PDF/DOCX 버튼) · ② Editor(순수 편집 — AI 없음) | default·loading·error |
| 5 | 지원 관리 | applications | ① CanvasHeader(지원 관리+공고 추가) · ② CardGrid(지원 카드×n) · StatusChip · ③ Toast | default·empty·loading(그리드 스켈레톤)·error |
| 6 | 공고 상세 | job-detail | ① CanvasHeader(회사·직무) · ② Card(요구사항 분석) · ③ Card(커리어 볼트 차이) · ④ Button(지원 준비 시작) | default·loading·error |
| 7 | 커리어 볼트 | vault | ① CanvasHeader(커리어 볼트+근거 추가) · ② DataList(근거 행×n) · StatusChip(종류 칩) · ③ Toast | default·empty·loading·error |
| 8 | 프로젝트 채팅 | project-chat | ① CanvasHeader(프로젝트명+프로젝트 태그) · ② ChatStream(블루프린트 카드·실행 상태 카드) · ApprovalCard(CLI 실행 승인) · StatusChip · ③ Composer(추론 설정 아이콘) | default·loading·error |
| 9 | 면접 | interview | ① CanvasHeader(면접+면접 준비) · ② DataList(예상 질문×n) · ③ Card(STAR 답변·회고) | default·empty·loading·error |
| 10 | 캘린더 | calendar | ① CanvasHeader(캘린더+동기화) · ② Card(월간 달력) · ③ DataList(다가오는 일정) · StatusChip(D-day) | default·empty·loading·error |
| 11 | 설정 | settings | ① CanvasHeader(설정) · ② Form(계정·AI(OpenAI)·요금제·외관 섹션 — UltraResume 없음) · ③ Button(저장) | default·loading·error |

## 사용자 수정 이력

| 화면 | 변경 | 원문 | 회차 |
|---|---|---|---|
| 오퍼 | 화면 제거 | "오퍼 자체제거" | 1 |
| 반복 작업 | 화면 제거 | "반복 작업 자체 제거" | 1 |
| 로그인 | 덜 밋밋하게 — 그래디언트 배경·로고 뱃지·가치 칩 추가 | "좀 밋밋해" | 1 |
| 설정 | BYOK 제거 → OpenAI만. 요금제 섹션 추가 (Free/Ultra — 울트라 이력서 포함) | "BYOK — 제거. OpenAI만. 요금제도 있어야 됨. 요금제 있으면 울트라resume 기능도 사용 가능하게." | 1 |
| 로그인 | 로고 뱃지 제거(로고 미정), 카드 안 칩 라벨 제거 | "2번에 쓸때없는 라벨 있음. 1번은 빼자 로고는 아직 없음. 3,4번 괜찮차" | 2 |
| 설정 | UltraResume은 설정이 아니라 채팅 입력창 추론 설정 UI로 이동. 영문 표기 띄어쓰기 없이 | "울트라이력서는 영어로 띄어쓰기없이임. 설정에는 없음. 채팅 인풋에서 추론 설정 UI에 있어야 됨." | 2 |
| 내 서류 | 목록/편집 분리 — DocCard 탭 시 문서 편집 하위 페이지로 이동 (인라인 펼침 아님) | "누르면 하위 페이지로 이동임. 내 서류 페이지에서 펼쳐지는게 아니라 다른 페이지로 이동해서 보기-수정 가능한거임." | 2 |
| 설정 | 요금제에서 업그레이드 노출 제거 — 플랜 표시만 | "업그레이드는 노출안함." | 3 |
| 내 서류 | DocCard 컬러 띠 제거 — 단색 카드 | "쓸때없는 색 제거" | 3 |
