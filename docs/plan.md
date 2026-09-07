# Push 데스크톱 앱 구현 계획

## 1. 제품과 UI 방향

- Super Resume를 리모델링해 `탐색 → 역량 보완 → 문서 생성 → 지원 → 면접 → 오퍼`를 하나의 작업공간에서 관리한다.
- macOS 우선 데스크톱 앱으로 출시하고 Windows를 같은 코드베이스에서 지원한다.
- Aside를 레이아웃·밀도·표면·상호작용까지 거의 동일하게 재현한다. 로고, 명칭, 콘텐츠는 Push 고유 자산을 사용한다.
- Aside의 UI 구조를 Push 도메인으로 치환한다.
  - `Bookmarks` → 고정한 공고·문서
  - `Chats` → 회사별 지원 워크스페이스
  - `Tabs` → 열어둔 공고·문서·면접
  - 중앙 Ask 입력창 → Push AI 명령창
  - `Project / Access / Model / Effort` → 지원 대상 / 실행 권한 / AI 모델 / 작업 강도
  - Aside 우측 요약 패널 → 공고 요구사항, 지원 단계, 문서 점수, 다음 행동
- 개인정보가 있는 Account 캡처는 UI 분석에만 사용하고 제품 문서·테스트 픽스처에는 포함하지 않는다.

주요 캡처 레퍼런스:

- 대화 작업공간: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-01-chat-workspace.jpeg
- 홈·새 작업: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-02-new-tab.jpeg
- 설정·외관: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-05-settings-appearance.png
- 메모리 3열 구조: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-13-memory.png
- 루틴: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-15-routines.png
- CLI 설정: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-17-developers.png
- 지원 작업 목록형 화면: /Users/cyjoon/.codex/visualizations/2026/09/01/01a05c1d-cda1-75d3-a2c2-13a7d9709587/aside-22-all-chats-list.png

## 2. 핵심 기능

- Career Vault
  - 기존 이력서, GitHub, 경력, 학력, 기술, 성과 수치, 프로젝트 증거를 한 번만 수집한다.
  - 모든 AI 생성 내용은 원본 근거와 연결하며 근거 없는 경험은 생성하지 않는다.
- 공고 및 지원 관리
  - URL·텍스트·내장 브라우저 DOM에서 공고를 수집한다.
  - 요구 기술, 역할, 우대사항, 키워드, 위험 요소와 후보자의 차이를 분석한다.
  - `DISCOVERED → PREPARING → READY → APPLIED → SCREENING → INTERVIEW → OFFER → ACCEPTED | REJECTED | WITHDRAWN` 단계로 관리한다.
- 문서 작업실
  - 이력서와 포트폴리오는 PDF, 자기소개서는 DOCX로 출력한다.
  - 3개 기본 템플릿과 실시간 미리보기를 제공한다.
  - TipTap 선택 영역에 Aside와 유사한 AI 버블을 띄워 문장 수정, 압축, 수치 강조, 톤 변경, 공고 맞춤화를 수행한다.
  - 문서별 공고 적합도, 근거 충실도, 가독성, ATS 점수를 표시한다.
- 역량 보완 프로젝트
  - 부족한 역량을 기준으로 구현 가능한 프로젝트 블루프린트 4개를 제안한다.
  - 사용자가 선택하면 별도 폴더에 복사하고 외부 Terminal에서 코딩 CLI를 실행한다.
  - GitHub 커밋, 테스트, 실행 결과, 측정 지표가 검증되어야 Career Vault와 문서에 반영된다.
- 지원과 후속 관리
  - Wanted, Jumpit, JobKorea 어댑터를 우선 지원한다.
  - DOM 기반으로 필드를 채우되 최종 제출은 지원 건마다 사용자 승인을 받는다.
  - Gmail에서 지원·면접 메일을 동기화하고 Google Calendar와 내부 캘린더에 일정을 반영한다.
  - 면접 예상 질문, STAR 답변, 회사 조사, 면접 회고, 오퍼 기록·비교를 지원한다.
- 반복 작업
  - Aside Routines와 같은 화면에서 마감 임박, 답변 지연, 면접 준비, 후속 메일 필요 작업을 제안한다.
  - 자동 실행보다 "제안 → 사용자 확인 → 실행"을 기본값으로 한다.

## 3. 기술 구조와 인터페이스

- 데스크톱: Tauri v2, React, TypeScript, Vite, TipTap, SQLite.
- 서버: Go + Echo, PostgreSQL, Caddy, Docker Compose, Hostinger VPS.
- 인증: Google·GitHub OAuth와 Tauri 딥링크 복귀.
- AI:
  - Push 관리형 OpenAI와 BYOK OpenAI·Claude·Gemini·Grok을 지원한다.
  - BYOK 키는 서버에서 AES-GCM으로 암호화하고 평문 로그를 남기지 않는다.
  - 관리형 AI는 Stripe 월 구독과 원가 크레딧으로 사용량을 제한한다.
- 코딩 CLI:
  - `CliProvider = CODEX | CLAUDE_CODE | GROK_BUILD`
  - `CliRunState = DRAFT | APPROVAL_REQUIRED | RUNNING | VERIFYING | VERIFIED | FAILED`
  - `codex`, `claude`, Grok Build 실행 파일의 설치 여부와 버전을 감지한다.
  - 명령·작업 디렉터리·전달 프롬프트를 미리 보여준 후 승인받고 외부 Terminal에서 실행한다.
  - v1에서는 내장 터미널과 자체 PTY를 만들지 않는다.
- 주요 데이터 타입:
  - `CareerEvidence`, `JobPosting`, `Application`, `Document`, `DocumentVersion`
  - `GapAnalysis`, `ProjectBlueprint`, `ProjectEvidence`
  - `InterviewSession`, `Offer`, `Routine`, `Approval`, `AiUsage`
- 승인 종류:
  - `EVIDENCE_USE`, `DOCUMENT_FINALIZE`, `APPLICATION_SUBMIT`, `CLI_EXECUTE`
- API 영역:
  - `/auth`, `/career-evidence`, `/jobs`, `/applications`
  - `/documents`, `/projects`, `/interviews`, `/offers`
  - `/integrations/google`, `/ai`, `/billing`, `/approvals`
- Gmail과 Calendar는 앱 실행 시 증분 동기화하며 Gmail 제한 범위 검증 전에는 베타 기능 플래그로 제한한다.
- 채용 사이트 자동화도 서비스 약관과 공식 허가가 확인될 때까지 기본 비활성화하고, 수동 지원 체크리스트는 항상 제공한다.

## 4. 구현 순서

1. Super Resume의 파서·공고 분석·콘텐츠 전략·문서 생성·품질 검토 흐름을 독립 도메인 모듈로 이식한다.
2. Aside 기반 데스크톱 셸, 세로 탭, 홈 명령창, 회사별 채팅, 우측 컨텍스트 패널을 구현한다.
3. Career Vault, 공고 분석, 문서 편집·버전·PDF/DOCX 출력을 연결한다.
4. 애플리케이션 파이프라인, 캘린더, 면접, 오퍼를 추가해 전체 지원 생명주기를 완성한다.
5. 프로젝트 블루프린트와 Codex CLI·Claude Code CLI·Grok Build 실행 및 검증 흐름을 연결한다.
6. Google 연동, 결제, BYOK, 채용 사이트 어댑터를 기능 플래그 뒤에서 통합한다.
7. 서명·공증된 Universal DMG와 자동 업데이트 채널을 만들고 20명 비공개 베타를 진행한다.

## 5. 검증과 완료 조건

- Career Vault 근거가 공고 분석, 문서, 프로젝트, 면접 답변까지 추적되는지 검증한다.
- 서로 다른 공고 두 개에서 문서 버전과 지원 데이터가 섞이지 않아야 한다.
- 근거 없는 수치·기술·경력을 AI가 생성하면 확정 단계에서 차단해야 한다.
- PDF와 DOCX 출력의 한글 글꼴, 페이지 분할, 링크, ATS 텍스트 추출을 검사한다.
- OAuth 복귀, 오프라인 편집, 동기화 충돌, 토큰 만료, Stripe 웹훅 중복을 테스트한다.
- Codex·Claude Code·Grok Build 각각에 대해 미설치, 승인 거부, 실행 실패, 검증 성공을 테스트한다.
- 지원 자동화는 미승인 제출이 절대 발생하지 않도록 E2E 테스트한다.
- 키·이력서·메일 내용이 로그와 분석 이벤트에 포함되지 않는지 보안 테스트한다.
- macOS 키보드 탐색, VoiceOver 레이블, 축소 창, 다크 모드가 Aside 레퍼런스와 같은 수준으로 동작해야 한다.
- 베타 완료 기준은 20명 중 80% 이상이 실제 공고 하나를 등록하고 문서 생성부터 지원 추적까지 완료하는 것이다.

## 가정

- 초기 사용자는 개인 구직자이며 기업 채용 담당자 기능은 제외한다.
- UI는 Aside에 최대한 가깝게 만들되 Aside 상표, 로고, 삽화와 독점 자산은 복사하지 않는다.
- 한국어·영어 UI를 제공하고 생성 문서 언어는 공고와 사용자 입력에서 추론한다.
- macOS를 먼저 완성한 뒤 같은 Tauri 코드베이스로 Windows 적응형 UI를 제공한다.
- 법적 허가가 없는 지원 사이트에서는 자동 제출 대신 DOM 분석과 수동 체크리스트까지만 활성화한다.

## UI 결정 업데이트 (프로토타입 반영)

프로토타입 반복 후 확정된 사용자 결정:

- 좌측 사이드바는 Aside 사이드바의 외형(폭·밀도·섹션·구분선·선택 행)을 그대로 따르고 내용만 Push용으로 치환한다: `기능 / 채팅 / 프로젝트`.
- 기능 항목 명칭은 `이력서 · 포트폴리오`가 아니라 `내 서류`로 확정한다.
- 오른쪽 사이드바(지원 컨텍스트 패널)는 만들지 않는다. 채팅 캔버스가 전체 폭을 사용한다.
- 확정된 최종 프로토타입: docs/no-right-sidebar-prototype-v6.png

## 개발 방식 확정 (2026-09-07)

- API 계약은 이 저장소의 `docs/api.md`에 먼저 작성한다. 그 계약을 기준으로 FE·BE를 병렬 개발한다.
- 기존 코드를 참고하거나 이식하지 않고 처음부터 새로 개발한다. 앞선 Super Resume 이식 문구보다 이 결정이 우선한다.
- React HTTP 클라이언트는 `ky`, 상태 관리는 `Zustand`, 프런트엔드 구조는 스킬에 따른 FSD v2.1로 확정한다.
- 외부 계정·서명·배포·베타 준비 전에는 구현과 로컬 검증을 진행하고 실제 출시 조건은 별도로 남긴다.
