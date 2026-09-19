# 구현·출시 검증 현황

계획 원문은 `plan.md`다. 사용자 결정에 따라 신규 구현과 로컬 검증을 진행했으며, 실제 외부 계정·배포·서명·베타는 준비 후 별도로 검증한다.

| 단계 | 상태 | 증거 |
| --- | --- | --- |
| API 계약 | 작성 완료 | 원본 workspace `docs/api.md`에 먼저 작성, 구현과 함께 동기화 |
| 저장소 분리 | 완료 | 비공개 `push-dot/push-fe`, `push-dot/push-be`; 각각 develop에 squash 병합하고 검증한 동일 트리를 서브모듈로 고정 |
| 1. 독립 도메인 신규 구현 | 로컬 검증 통과 | Python·FastAPI·LangGraph·실제 PostgreSQL, Go 구현에서 전면 재작성 |
| 2. 데스크톱 셸 | 로컬 검증 통과 | React·ky·Zustand·FSD; Tauri 실행·단일 인스턴스·창 제어; v6 시각 비교 |
| 3. 근거·문서·출력 | 로컬 검증 통과 | 근거→분석→버전 확정, 한글 PDF 래스터·DOCX 렌더·링크; 편집/동기화 경합 회귀 검사 |
| 4. 지원 생명주기 | 로컬 검증 통과 | 승인·면접·일정·오퍼·루틴·회사 출처·불합격 기록 실제 API 및 UI 검증 |
| 5. CLI 실행·검증 | 로컬 검증 통과 | 실제 fixture subprocess, 영속 실행 기록·중복 실행 차단·폴더 교체 방어·commit SHA·검증 artifact 경계 |
| 6. 외부 연동 | 로컬 경계 검증 통과 | OAuth PKCE/회전, AI 공급자 HTTP, Google 페이지 실패/커서, Stripe 서명/원장/환불 fixture. 실제 계정 준비 전 |
| 7. 서명·공증·베타 | 외부 입력 필요 | Universal 앱·DMG 로컬 빌드/마운트 확인. 실제 인증서·업데이트 채널·베타 참가자 필요 |

## 검증 범위

검증한 제품 코드: FE `5bec651` (검증한 `9aad318`과 동일한 squash 결과), BE `d53b69f` (검증한 `5f99281`과 동일한 squash 결과). FE 40개 회귀·출력 검사와 별도 실제 API 4개, BE 30개 PostgreSQL race 검사, Rust 25개 검사, 실제 API 스모크 16개 항목이 통과했다.

최종 FE 커밋의 GitHub Actions `34110973822`에서 macOS Universal DMG와 Windows NSIS 빌드 및 출력 검사가 통과했다. Windows 실제 설치·업데이트 실행과 Apple 서명·공증은 별도 검증 대상이다.

상세 항목은 `acceptance.md`, 실행 이력은 `../gauntlet-progress.md`에 기록한다. 실제 API 스모크는 `../scripts/smoke.mjs`로 재현한다. 백엔드 테스트는 실제 PostgreSQL을 사용하며 프런트엔드 테스트는 문서 편집 경합과 실제 출력 파일을 검사한다. 공급자 응답 fixture는 실제 OAuth·결제·유료 AI 계정 사용 성공으로 간주하지 않는다.

시각 검증은 확정 v6 이미지와 1672px·800px·720px 브라우저 화면을 비교했다. 채팅 응답은 브라우저에만 주입한 명시적 시각 fixture이며 첨부 문서는 실제 로컬 API 리소스다. 실제 제공자 응답을 가장하거나 서버에 가짜 메시지를 저장하지 않았다. macOS 접근성 트리의 레이블과 키보드 포커스를 확인했다. 과거 Aside 캡처 파일이 없어 해당 원본과의 직접 비교는 수행하지 않았다.

## 실제 서비스 검증에 필요한 입력

민감한 값은 채팅이나 저장소에 쓰지 않고 배포 환경 또는 GitHub Actions secrets에 설정한다.

- Google/GitHub OAuth 앱과 리디렉션 URI, 실제 테스트 계정.
- 관리형 OpenAI 계정과 공급자별 테스트 BYOK.
- Stripe 제품·가격·웹훅 및 테스트/운영 계정.
- Google Gmail 제한 범위 검증 및 Calendar 테스트 계정.
- Hostinger VPS, API/업데이트 도메인 DNS.
- Apple Developer ID·공증 권한 및 Tauri 업데이트 서명 키.
- Wanted/Jumpit/JobKorea 자동화 허가 확인.
- Windows 실제 설치·딥링크·CLI·업데이트 실행 검증.
- 20명 베타 참가자와 실제 공고→문서→지원 추적 성공 기록(최소 16명).

이 항목은 로컬 검사나 CI 빌드만으로 완료 처리하지 않는다. 허가 확인 전 채용 사이트 자동 제출은 비활성화한다.
