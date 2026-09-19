# Push PRD (Product Requirements Document)

- 버전: 1.0
- 작성일: 2026-09-16
- 상태: 초안
- 관련 문서: [plan.md](plan.md), [api.md](api.md), [ui-contract.md](ui-contract.md), [acceptance.md](acceptance.md)

## 1. 개요

Push는 근거 수집부터 문서 작성, 지원, 면접, 오퍼까지 구직 전 과정을 하나의 작업공간에서 관리하는 macOS 우선 데스크톱 앱이다. `탐색 → 역량 보완 → 문서 생성 → 지원 → 면접 → 오퍼` 흐름을 단일 제품으로 통합한다.

## 2. 배경과 문제 정의

- 구직 활동이 이력서 도구, 스프레드시트, 메일, 캘린더, 메모 앱에 분산되어 있다.
- 기존 AI 문서 생성 도구는 근거 없는 경력·수치를 만들어 신뢰를 해친다.
- 지원 진행 상황과 후속 행동(팔로업 메일, 면접 준비)을 추적하는 통합 도구가 없다.

## 3. 목표

- Career Vault에 수집한 근거가 공고 분석, 문서, 프로젝트, 면접 답변까지 추적된다.
- 근거 없는 경험·수치·기술은 AI가 생성하지 못한다.
- 지원 자동화는 미승인 제출이 절대 발생하지 않는다.

## 4. 비목표

- 기업 채용 담당자(ATS 관리자) 기능
- 내장 터미널/자체 PTY (v1)
- 모바일 앱
- 법적 허가가 확인되지 않은 채용 사이트 자동 제출

## 5. 대상 사용자

- 개인 구직자 (초기 유일 대상)
- 한국어·영어 UI 지원

## 6. 플랫폼과 기술 스택

| 영역 | 스택 |
| --- | --- |
| 데스크톱 | Tauri v2, React 19, TypeScript, Vite, TipTap, SQLite |
| 프런트엔드 구조 | FSD v2.1, ky, Zustand |
| 서버 | Python, FastAPI, LangGraph, PostgreSQL, Caddy, Docker Compose, Hostinger VPS |
| 인증 | Google·GitHub OAuth + Tauri 딥링크 복귀 |
| AI | 관리형 OpenAI + BYOK (OpenAI·Claude·Gemini·Grok), BYOK 키는 AES-GCM 암호화 |
| 결제 | Stripe 월 구독 + 원가 크레딧 |

- macOS 우선 출시, 같은 Tauri 코드베이스로 Windows 지원
- UI는 Aside 레퍼런스를 재현하되 Push 고유 자산 사용 (상표·로고·삽화 복사 금지)

## 7. 기능 요구사항

### 7.1 Career Vault

- 기존 이력서, GitHub, 경력, 학력, 기술, 성과 수치, 프로젝트 증거를 한 번만 수집
- 모든 AI 생성 내용은 원본 근거와 연결, 근거 없는 경험 생성 차단

### 7.2 공고 및 지원 관리

- URL·텍스트·내장 브라우저 DOM에서 공고 수집
- 요구 기술·역할·우대사항·키워드·위험 요소와 후보자 차이 분석
- 상태 기계: `DISCOVERED → PREPARING → READY → APPLIED → SCREENING → INTERVIEW → OFFER → ACCEPTED | REJECTED | WITHDRAWN`

### 7.3 문서 작업실

- 이력서·포트폴리오 PDF, 자기소개서 DOCX 출력
- 기본 템플릿 3개 + 실시간 미리보기
- TipTap 선택 영역 AI 버블: 문장 수정, 압축, 수치 강조, 톤 변경, 공고 맞춤화
- 문서별 공고 적합도·근거 충실도·가독성·ATS 점수 표시

### 7.4 역량 보완 프로젝트

- 부족 역량 기준 프로젝트 블루프린트 4개 제안
- 선택 시 별도 폴더에 복사 후 외부 Terminal에서 코딩 CLI 실행
- GitHub 커밋·테스트·실행 결과·측정 지표 검증 후에만 Career Vault와 문서에 반영
- `CliProvider = CODEX | CLAUDE_CODE | GROK_BUILD`
- `CliRunState = DRAFT | APPROVAL_REQUIRED | RUNNING | VERIFYING | VERIFIED | FAILED`
- 명령·작업 디렉터리·프롬프트 사전 표시 후 승인, 외부 Terminal 실행

### 7.5 지원과 후속 관리

- Wanted, Jumpit, JobKorea 어댑터 우선
- DOM 기반 필드 채움, 최종 제출은 건별 사용자 승인
- Gmail 지원·면접 메일 동기화, Google Calendar·내부 캘린더 일정 반영
- 면접 예상 질문, STAR 답변, 회사 조사, 면접 회고, 오퍼 기록·비교

### 7.6 반복 작업 (Routines)

- 마감 임박, 답변 지연, 면접 준비, 후속 메일 작업 제안
- `제안 → 사용자 확인 → 실행` 기본값

### 7.7 승인 체계

- 승인 종류: `EVIDENCE_USE`, `DOCUMENT_FINALIZE`, `APPLICATION_SUBMIT`, `CLI_EXECUTE`

## 8. UI 계약

- 좌측 사이드바: Aside 외형 재현, 내용은 `기능 / 채팅 / 프로젝트`
- 문서 기능명: `내 서류`
- 오른쪽 컨텍스트 패널 없음 — 채팅 캔버스 전체 폭
- 확정 프로토타입: `docs/no-right-sidebar-prototype-v6.png`

## 9. 데이터 모델 (주요 타입)

`CareerEvidence`, `JobPosting`, `Application`, `Document`, `DocumentVersion`, `GapAnalysis`, `ProjectBlueprint`, `ProjectEvidence`, `InterviewSession`, `Offer`, `Routine`, `Approval`, `AiUsage`

## 10. API 영역

`/auth`, `/career-evidence`, `/jobs`, `/applications`, `/documents`, `/projects`, `/interviews`, `/offers`, `/integrations/google`, `/ai`, `/billing`, `/approvals`

- API 계약은 `docs/api.md`에 먼저 작성, FE·BE는 계약 기준 병렬 개발
- Gmail·Calendar는 앱 실행 시 증분 동기화, Gmail 제한 범위 검증 전 베타 플래그
- 채용 사이트 자동화는 약관·허가 확인 전 기본 비활성화, 수동 체크리스트는 항상 제공

## 11. 비기능 요구사항

- 보안: 키·이력서·메일 내용이 로그와 분석 이벤트에 포함되지 않음
- 오프라인 편집 지원, 동기화 충돌 처리
- 접근성: macOS 키보드 탐색, VoiceOver 레이블, 축소 창, 다크 모드
- 개인정보가 포함된 캡처는 제품 문서·테스트 픽스처에 포함 금지

## 12. 성공 지표와 완료 조건

- 근거 추적: Career Vault 근거가 공고 분석→문서→프로젝트→면접 답변까지 추적
- 격리: 서로 다른 공고 간 문서 버전·지원 데이터 미혼합
- 차단: 근거 없는 수치·기술·경력 생성 시 확정 단계 차단
- 출력: PDF/DOCX 한글 글꼴·페이지 분할·링크·ATS 텍스트 추출 검사
- 안정성: OAuth 복귀, 토큰 만료, Stripe 웹훅 중복, CLI 미설치/승인 거부/실패/성공 테스트
- 안전: 미승인 제출 부재 E2E 테스트
- 베타: 20명 중 80% 이상이 공고 등록 → 문서 생성 → 지원 추적 완료

## 13. 마일스톤

1. 독립 도메인 모듈: 파서·공고 분석·콘텐츠 전략·문서 생성·품질 검토
2. 데스크톱 셸: 세로 탭, 홈 명령창, 회사별 채팅
3. Career Vault + 공고 분석 + 문서 편집·버전·PDF/DOCX 출력
4. 애플리케이션 파이프라인 + 캘린더 + 면접 + 오퍼
5. 프로젝트 블루프린트 + 코딩 CLI 실행·검증
6. Google 연동, 결제, BYOK, 채용 사이트 어댑터 (기능 플래그 뒤)
7. 서명·공증 Universal DMG + 자동 업데이트 + 20명 비공개 베타

## 14. 가정과 리스크

- 초기 사용자는 개인 구직자
- 생성 문서 언어는 공고와 사용자 입력에서 추론
- 법적 허가 없는 사이트는 DOM 분석 + 수동 체크리스트까지만
- 외부 계정·서명·배포·베타 준비 전까지는 구현과 로컬 검증만 수행

## 15. 개발 원칙

- 처음부터 새로 개발, 기존 코드 이식 없음
- `develop` 브랜치 기준, `feat|fix|refactor|chore|docs|test/<short-kebab>` 브랜치, scoped Conventional Commit, squash merge
- 세 저장소(push-workspace, push-fe, push-be)는 서브모듈로 고정 커밋 관리
