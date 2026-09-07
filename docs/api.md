# Push API 명세

> 상태: 신규 개발을 위한 v1 설계 계약. 이 문서는 구현 완료 목록이 아니다.
> 기준: `docs/plan.md`, 최종 UI 프로토타입, 사용자 확정 사항(React · ky · Zustand · FSD).
> 개발 방식: 기존 코드 이식 없이 새로 구현한다. API 계약을 먼저 작성하고 FE·BE가 이 계약을 기준으로 병렬 개발한다.

## 1. 범위와 책임

Push는 `탐색 → 역량 보완 → 문서 생성 → 지원 → 면접 → 오퍼`를 지원한다. 서버는 사용자 소유권, 원본 근거, 지원별 데이터 격리, 문서 버전, 승인, 연동 토큰, AI 사용량과 결제를 관리한다. Tauri 앱은 SQLite 오프라인 저장, TipTap 편집, PDF/DOCX 출력, 내장 브라우저 DOM 수집, 설치된 CLI 탐지와 외부 Terminal 실행을 담당한다.

오른쪽 컨텍스트 패널은 만들지 않는다. 왼쪽은 `기능 / 채팅 / 프로젝트`, 문서 기능명은 `내 서류`다. API 데이터는 특정 패널 배치를 전제하지 않는다.

외부 계정·도메인·인증서·베타 참가자는 아직 준비되지 않았다. 공급자 연결이 필요한 기능은 설정 여부를 명시하며, 미설정 응답을 성공이나 샘플 데이터로 대체하지 않는다.

## 2. 공통 계약

### 주소·인증·형식

- 개발: `http://localhost:8080/api/v1`
- 운영: `https://<API_DOMAIN>/api/v1`
- 요청/응답: UTF-8 JSON, 필드명 camelCase.
- ID: UUID 문자열. 시각: RFC3339 UTC. 날짜만 필요한 값: `YYYY-MM-DD`.
- 인증: `Authorization: Bearer <accessToken>`. 예외는 로그인용 OAuth 시작·콜백·교환·갱신, Google 연결 콜백, Stripe 웹훅과 health endpoint뿐이다. Google connect/complete에는 기존 Push 인증이 필요하다.
- 개발 인증은 `APP_ENV=development`에서 명시적으로 지정한 `DEV_AUTH_TOKEN`만 허용한다. 운영 환경에는 개발 인증을 허용하지 않는다.
- 사용자는 OAuth 세션에서 결정한다. 요청 body의 `userId`, `ownerId`, `createdAt`, `revision` 등 서버 관리 필드를 거부한다.
- 모든 조회·변경·참조 ID에 소유권을 검증한다. 타 사용자 리소스는 존재 여부를 노출하지 않고 404로 응답한다.
- 본문 기본 제한 1 MiB. 파일은 서버 JSON에 base64로 넣지 않으며 아래 업로드 계약을 사용한다.
- 알 수 없는 필드, enum, 유효하지 않은 UUID/URL/시각, 허용 길이를 초과한 입력은 400이다.
- URL은 자격증명을 포함하지 않는 HTTP(S)만 허용한다. 외부 원문은 신뢰하지 않는 데이터이며 AI 도구 실행 지시로 취급하지 않는다.

### 응답

성공 응답에 포함된 revision·날짜는 서버 값이며 아래 JSON은 구조 설명용 예시다.

단일 성공:

```json
{"data":{"id":"8a8d04db-6e6f-42fd-8588-1f48b0498b53","revision":1,"createdAt":"2026-09-07T09:00:00Z","updatedAt":"2026-09-07T09:00:00Z"}}
```

목록 성공:

```json
{"data":[],"page":{"nextCursor":null,"hasMore":false}}
```

목록은 `limit`(기본 50, 최대 100), 불투명한 `cursor`를 받는다. 정렬은 `createdAt DESC, id DESC`이며 다음 페이지에서도 같은 필터를 사용한다. 지원에 속한 목록은 `applicationId` 필터를 받는다. 목록의 페이지 제한 때문에 분석·집계·근거 검증 대상이 누락되어서는 안 된다.

오류:

```json
{"error":{"code":"REVISION_CONFLICT","message":"다른 기기에서 문서가 변경되었습니다.","requestId":"req_01","details":{"currentRevision":4}}}
```

오류 `details`는 검증 필드명·현재 revision처럼 복구에 필요한 값만 포함하며 원문·키·토큰을 포함하지 않는다.

| HTTP | 의미 |
| --- | --- |
| 200 | 조회·수정·명시적 액션 성공 |
| 201 | 리소스 생성 |
| 202 | 비동기 작업 접수. 응답 data는 Operation이며 `data.id`로 상태 조회 |
| 204 | 삭제/로그아웃 성공, body 없음 |
| 400 | `VALIDATION_ERROR` |
| 401 | `UNAUTHENTICATED`, `TOKEN_EXPIRED` |
| 403 | `FEATURE_DISABLED`, `INSUFFICIENT_SCOPE` |
| 404 | `NOT_FOUND` |
| 409 | `REVISION_CONFLICT`, `INVALID_TRANSITION`, `APPROVAL_REQUIRED`, `APPROVAL_STALE`, `UNSUPPORTED_CLAIM`, `INTEGRATION_REQUIRED`, `IDEMPOTENCY_CONFLICT` |
| 413 | `PAYLOAD_TOO_LARGE` |
| 422 | `VERIFICATION_FAILED`, `DOCUMENT_NOT_FINALIZED` |
| 429 | `RATE_LIMITED`, `CREDIT_EXHAUSTED`. `Retry-After` 포함 |
| 502 | `PROVIDER_ERROR` |
| 503 | `NOT_CONFIGURED`, `TEMPORARILY_UNAVAILABLE` |

### 동시성·멱등성

- 변경 가능한 리소스는 양의 정수 `revision`을 가진다. PATCH와 버전 생성·확정 등 변경 액션은 `expectedRevision`을 필수로 받는다.
- revision이 다르면 변경하지 않고 409 `REVISION_CONFLICT`와 `currentRevision`을 반환한다. 자동 덮어쓰기는 하지 않는다.
- 생성 및 외부 부작용이 있는 POST는 UUID `Idempotency-Key`를 필수로 받는다. 인증 교환·갱신·로그아웃과 웹훅은 각각 일회용 토큰·세션 회전·이벤트 ID로 중복을 제어하므로 이 헤더에서 제외한다. 동일 사용자·메서드·경로·키·본문의 재요청에는 같은 결과를 반환한다. 같은 키에 다른 본문은 409다.
- 멱등 기록은 최소 7일 보존한다. Stripe 이벤트 ID와 실제 지원 제출 실행 ID는 별도의 영구 중복 방지 키를 사용한다.
- 데이터 변경·승인 소비·사용량 반영은 원자적이다. 트랜잭션 커밋 전에 성공 응답을 보내지 않는다.
- FE의 ky는 조회 재시도만 제한적으로 수행한다. 변경 요청은 타임아웃 후 동일 멱등 키로 상태를 확인하거나 재시도하며 새 키로 자동 재실행하지 않는다.
- 외부 서비스가 요청을 수신했는지 불명확한 경우 `UNKNOWN` 실행 상태를 노출하고 확인 전 재전송을 막는다.

### 공통 필드

변경 가능한 엔티티는 아래 공통 필드를 사용한다. 불변 DocumentVersion, GapAnalysis, SubmissionDraft, Message, AiUsage, 원장 항목과 계산/작업 결과는 명시된 필드만 가지며 revision/updatedAt을 자동으로 추가하지 않는다. 그 외 엔티티는 특별한 언급이 없으면 `id`, `revision`, `createdAt`, `updatedAt`을 가진다. `?`는 선택 입력/nullable 응답이며, 배열은 값이 없어도 `[]`다. 금액은 부동소수점이 아닌 통화 최소 단위 정수와 ISO 4217 `currency`로 저장한다. 크레딧은 정수 `microCredits`를 사용한다.

## 3. 인증과 세션 `/auth`

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/auth/:provider/start` | query `codeChallenge`, `codeChallengeMethod=S256`, `redirectUri=push://auth/callback` | `{authorizationUrl,state,expiresAt}` |
| GET `/auth/:provider/callback` | query `code,state` | 일회용 교환 코드를 담은 허용된 앱 딥링크로 302 |
| POST `/auth/exchange` | `{code,codeVerifier}` | `Session` |
| POST `/auth/refresh` | `{refreshToken}` | 교체된 `Session` |
| POST `/auth/logout` | `{refreshToken}` | 204 |
| GET `/auth/me` | — | `{id,displayName,locale,createdAt}` |

`provider=google|github`. Session은 `{accessToken,refreshToken,expiresIn:900,user:{id,displayName,locale}}`다. state는 10분, 딥링크 교환 코드는 60초, refresh token은 최대 30일이며 교환 코드는 1회만 사용한다. PKCE 검증 실패·state 불일치·만료·재사용은 인증 실패다. redirectUri는 서버의 정확한 allowlist와 비교한다.

OAuth 공급자 토큰과 Push 세션은 구분한다. Push access/refresh token은 URL·로그로 전달하지 않는다. 갱신 토큰은 매번 회전하고 이전 토큰을 폐기한다. FE는 동시 401 응답을 하나의 갱신 작업으로 묶고, 갱신 실패 시 다시 로그인하도록 한다. 로그아웃/계정 전환 시 해당 계정의 메모리 상태와 인증 정보를 비운다.

## 4. Career Vault `/career-evidence`

### 모델

`CareerEvidence`:

```ts
type CareerEvidence = {
  id: string; revision: number; createdAt: string; updatedAt: string;
  kind: 'RESUME' | 'GITHUB' | 'CAREER' | 'EDUCATION' | 'SKILL' | 'PROJECT';
  title: string;
  sourceText: string;
  sourceUrl: string | null;
  skills: string[];
  verificationStatus: 'USER_PROVIDED' | 'PENDING' | 'VERIFIED' | 'REJECTED';
  provenance: { sourceId: string | null; projectEvidenceId: string | null; contentHash: string; sourceLocation: { start: number; end: number; unit: 'CODE_POINT' } | null };
};
```

원문·출처·해시는 보존한다. 원문 정정은 새 근거로 생성하고 `supersedesId`로 연결한다. 이미 문서 버전이 인용한 원문을 덮어쓰지 않는다. `USER_PROVIDED`는 사용자 입력임을 의미하며 독립 검증과 혼동하지 않는다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/career-evidence` | `kind?`, `query?`, 페이지 | CareerEvidence 목록 |
| POST `/career-evidence` | `{kind,title,sourceText,sourceUrl?,skills:[],supersedesId?}` | CareerEvidence |
| GET `/career-evidence/:id` | — | CareerEvidence |
| POST `/career-evidence/import` | `{sourceId?,text?,sourceUrl?,contentHash?,format:'TEXT'|'MARKDOWN'|'PDF'|'DOCX'|'GITHUB'}` | 202 Operation |
| POST `/career-evidence/:id/archive` | `{expectedRevision}` | 보관된 CareerEvidence |

제목 1~200자, 원문 1~100,000자, 기술은 항목당 1~100자/최대 100개다. import는 파일 추출·경력/학력/프로젝트/성과 분리를 수행하되 파싱 결과를 사용자가 확인할 수 있게 원문 위치와 연결한다. 파싱할 수 없는 스캔 PDF 등은 `NEEDS_INPUT`으로 반환하고 빈 성공을 만들지 않는다. GitHub 원문은 허용된 GitHub URL/연동 계정으로 수집한다.

근거 사용은 `EVIDENCE_USE` 승인 대상이며 승인 범위는 근거 ID와 사용할 지원 ID의 조합이다. 보관된 근거는 기존 인용 조회를 유지하되 새 생성에 선택하지 않는다.

### 원본 파일 `/sources`

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| POST `/sources` | multipart `file`, `kind` | `{id,fileName,mimeType,size,sha256,status}` |
| GET `/sources/:id` | — | 소유자용 원본 메타데이터 |
| GET `/sources/:id/content` | — | 인증된 원본 다운로드 |
| DELETE `/sources/:id` | — | 204. 인용 중이면 409 |

PDF/DOCX/TXT/Markdown, 파일당 최대 20 MiB. 확장자만 믿지 않고 MIME·파일 형식·크기를 검증한다. 원본은 공개 URL로 제공하지 않고 실행하지 않는다. 데스크톱에서 텍스트를 추출한 경우 `text`와 원본 해시를 import에 전달할 수 있다.

수집 근거의 `provenance.sourceLocation`은 보존된 추출 원문의 코드 포인트 구간 `[start,end)`이다. 수동 입력처럼 별도 원문 위치가 없는 경우 null이다. 바이트·UTF-16 인덱스와 혼용하지 않으며 파일 추출 결과 원문과 contentHash를 함께 보존한다.

## 5. 공고·요구사항 분석 `/jobs`

`JobPosting`: `{id,revision,company,title,sourceKind,sourceUrl,sourceText,requirements:[],preferred:[],keywords:[],risks:[],deadline,language,createdAt,updatedAt}`.

`GapAnalysis`: `{id,applicationId,jobId,jobRevision,evidenceIds,matched:[{requirement,evidenceIds}],missing:[],preferredMissing:[],risks:[],fitScore,method,createdAt}`. 점수는 0~100이며 `method=RULE_BASED|AI_ASSISTED`와 근거를 함께 표시한다. 요구사항이 없으면 점수는 `null`이며 임의로 0점/100점을 만들지 않는다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/jobs` | `query?`, `archived?`, 페이지 | JobPosting 목록 |
| POST `/jobs` | `{company,title,sourceKind:'URL'|'TEXT'|'DOM',sourceUrl?,sourceText,requirements:[],preferred:[],deadline?,language?}` | JobPosting |
| GET `/jobs/:id` | — | JobPosting |
| PATCH `/jobs/:id` | `{expectedRevision,company?,title?,requirements?,preferred?,deadline?}` | JobPosting |
| POST `/jobs/:id/analyze` | `{applicationId,expectedRevision,evidenceIds:[],ai:AiOptions|null}` | 202 Operation → GapAnalysis |
| GET `/jobs/:id/analyses` | 페이지 | 분석 이력 |

분석 전에 지원 리소스를 생성한다. analyze의 applicationId가 해당 jobId에 속하는지, 각 evidenceId에 해당 지원의 EVIDENCE_USE 승인이 있는지 확인한다.

URL/DOM 방식도 수집한 원문과 출처 URL이 필요하다. 서버 임의 URL fetch를 기본 제공하지 않는다. 내장 브라우저는 사용자가 연 공고에서 명시적 수집 액션으로 본문을 읽는다. 역할·필수 기술·우대사항·위험 요소 추출 결과는 사용자가 수정할 수 있다. 공고나 근거 변경 후 과거 분석은 이력으로 남기고 오래된 분석임을 표시한다.

## 6. 지원 파이프라인 `/applications`

`Application`: `{id,revision,jobId,company,title,stage,notes,appliedAt,nextActionAt,createdAt,updatedAt}`.

```text
DISCOVERED → PREPARING → READY → APPLIED → SCREENING → INTERVIEW → OFFER → ACCEPTED
비종료 단계 → REJECTED | WITHDRAWN
```

일반 PATCH는 현재 단계의 메모 수정 또는 다음 단계 이동만 허용한다. 종료 상태를 변경하지 않는다. 과거 지원 가져오기는 별도 `import`에서 실제 지원일과 사용자 확인을 기록하며 자동 제출과 구분한다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/applications` | `stage?`, `query?`, 페이지 | Application 목록 |
| POST `/applications` | `{jobId,notes?}` | DISCOVERED Application |
| GET `/applications/:id` | — | Application |
| PATCH `/applications/:id` | `{expectedRevision,stage?,notes?,nextActionAt?}` | Application |
| POST `/applications/import` | `{jobId,stage,appliedAt,notes?,confirmed:true}` | 과거 지원 기록 |
| GET `/applications/:id/timeline` | 페이지 | 단계·문서·승인·면접·오퍼 이벤트 |
| GET `/applications/:id/checklist` | — | `{items:[{key,label,completed}],automationEnabled}` |
| POST `/applications/:id/submission-drafts` | `{expectedRevision,mode:'MANUAL_RECORD'|'ADAPTER',adapter?,documentVersionIds:[],confirmedSubmitted?}` | SubmissionDraft |
| POST `/applications/:id/submissions` | `{expectedRevision,draftId,approvalId}` | Submission 또는 202 Operation |
| GET `/applications/:id/submissions` | 페이지 | 제출 실행 이력 |

SubmissionDraft는 `{id,applicationId,applicationRevision,mode,adapter,documentVersionIds,confirmedSubmitted,payloadHash,createdAt}`의 불변 리소스다. 생성 시 서버가 대상 사이트와 제출 문서를 검증하고 hash를 계산한다. APPLICATION_SUBMIT 승인의 targetId는 Application이 아니라 SubmissionDraft ID다. 실행 시 원본 지원 revision과 draft 내용을 다시 확인한다. 바뀌면 새 draft와 승인이 필요하다.

APPLIED 이동은 submissions 경로에서만 수행한다. 수동 제출 기록도 사용자가 실제 외부 제출 완료를 확인해야 한다. `ADAPTER`는 사이트별 기능 플래그와 허가 설정이 모두 활성화된 경우에만 가능하다. 승인에는 해당 지원·불변 문서 버전·대상 사이트·제출 payload hash를 고정한다. 다른 지원 승인, 만료 승인, 변경 전 문서에 대한 승인으로 제출할 수 없다.

`Submission`: `{id,applicationId,mode,adapter,status:'PENDING'|'RUNNING'|'SUCCEEDED'|'FAILED'|'UNKNOWN',documentVersionIds,approvalId,receiptUrl,errorCode,createdAt,updatedAt}`. 외부 제출 결과 불명확 상태를 성공으로 기록하거나 자동 재제출하지 않는다.

## 7. 내 서류 `/documents`

`Document`: `{id,revision,applicationId,title,kind,template,language,status,latestVersionId,finalizedVersionId,createdAt,updatedAt}`.

- `kind=RESUME|PORTFOLIO|COVER_LETTER`
- `template=CLASSIC|MODERN|COMPACT`
- `status=DRAFT|FINALIZED|ARCHIVED`
- `DocumentVersion`: `{id,documentId,applicationId,number,content,blocks,changeNote,quality,createdAt}`. 버전은 불변이다.
- `content`가 유일한 본문 원본이다. 각 텍스트 블록 노드의 `attrs.blockId`와 blocks의 `id`는 일대일로 대응한다. 서버가 해당 노드에서 추출한 텍스트가 block.text와 정확히 일치해야 하며, 미연결 노드·중복 blockId·숨겨진 텍스트를 거부한다. 제목/리스트 내부 텍스트도 같은 검증을 받는다. `claimStatus`는 서버 전용 결과이며 생성 요청에서 받지 않는다.
- `content`는 검증된 TipTap JSON. 허용 노드는 문단·제목·리스트·텍스트·링크이며 임의 HTML/스크립트는 허용하지 않는다.
- `blocks`: `[{id,text,evidenceRefs:[{evidenceId,start,end}],claimStatus:'SUPPORTED'|'NEEDS_REVIEW'|'UNSUPPORTED'}]`.
- 원문 위치 `start/end`는 Unicode code point 기준 `[start,end)`이며 그 범위의 텍스트가 실제 원문과 일치해야 한다.
- `quality`: `{jobFit,evidenceFidelity,readability,ats,method,issues:[{code,severity,blockId,message}]}`. 점수는 0~100 또는 평가 불가 시 null.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/documents` | `applicationId?`, `kind?`, 페이지 | Document 목록 |
| POST `/documents` | `{applicationId,title,kind,template,language?}` | Document |
| GET `/documents/:id` | — | Document |
| PATCH `/documents/:id` | `{expectedRevision,title?,template?,language?}` | Document |
| GET `/documents/:id/versions` | 페이지 | DocumentVersion 목록 |
| GET `/documents/:id/versions/:versionId` | — | 해당 문서 소유 버전 |
| POST `/documents/:id/versions` | `{expectedRevision,content,blocks:BlockInput[],changeNote?}` | `{document,version}` |
| POST `/documents/:id/generate` | `{expectedRevision,evidenceIds:[],analysisId?,ai:AiOptions|null,language?}` | 202 Operation → `{document,version}` |
| POST `/documents/:id/revisions` | `{expectedRevision,versionId,selection:{from,to,text},action,instruction?,ai:AiOptions}` | 202 Operation → 변경 제안 |
| POST `/documents/:id/revisions/:revisionId/apply` | `{expectedRevision}` | `{document,version}` |
| POST `/documents/:id/review` | `{versionId}` | quality와 근거 검증 결과 |
| POST `/documents/:id/finalize` | `{expectedRevision,versionId,approvalId}` | FINALIZED Document |
| POST `/documents/:id/archive` | `{expectedRevision}` | ARCHIVED Document |

문장 수정 action은 `REWRITE|SHORTEN|EMPHASIZE_METRICS|CHANGE_TONE|TAILOR_TO_JOB`이다. AI 버블은 제안 전후 차이와 근거를 보여주고 사용자가 적용해야 새 버전을 만든다. 원래 선택 문구와 버전이 달라지면 적용을 거부한다.

원본 근거 없는 수치·기술·경력을 생성·확정하지 않는다. 수정된 문장도 근거 참조를 유지한다. 자동 사실 검증으로 확정할 수 없는 문장은 `NEEDS_REVIEW`로 남기고 원본 근거 보완을 요구한다. 단순 승인 버튼으로 `UNSUPPORTED`를 통과시키지 않는다. 확정은 모든 block의 근거 확인, 해당 지원의 근거 사용 승인, 정확한 버전의 DOCUMENT_FINALIZE 승인을 원자적으로 검증한다. 새 버전 생성·템플릿 변경은 기존 확정을 해제한다.

### PDF/DOCX 출력

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| POST `/documents/:id/exports` | `{versionId,format:'PDF'|'DOCX',rendererVersion}` | `{id,documentId,versionId,format,template,language,content,blocks,contentHash,status:'READY_TO_RENDER'}` |
| POST `/documents/:id/exports/:exportId/result` | `{sha256,byteLength,pageCount?,validation:{koreanText,links,atsText},status:'SUCCEEDED'|'FAILED',errorCode?}` | 출력 기록 |
| GET `/documents/:id/exports` | 페이지 | 출력 이력 |

이력서·포트폴리오는 PDF, 자기소개서는 DOCX가 기본 형식이다. 확정된 해당 문서 버전만 출력한다. Tauri가 실제 파일을 생성하고 저장한다. 파일 생성·텍스트 검사 실패는 실패로 표시한다. 서버 출력 기록은 클라이언트 관측 결과이며 서버가 파일 렌더를 독립 검증했다는 뜻이 아니다. 한국어 글꼴은 재배포 가능한 라이선스를 포함하고 페이지 분할·클릭 가능한 링크·ATS 텍스트 추출을 검사한다.

## 8. 프로젝트 블루프린트와 CLI `/projects`

`ProjectBlueprint`: `{id,revision,applicationId,gapAnalysisId,title,skills,problem,solution,tasks,completionCriteria,metrics,estimatedEffort,state,createdAt,updatedAt}`.

`CliProvider=CODEX|CLAUDE_CODE|GROK_BUILD`.

`CliRunState=DRAFT|APPROVAL_REQUIRED|RUNNING|VERIFYING|VERIFIED|FAILED`.

`CliRun`: `{id,revision,projectId,applicationId,provider,workingDirectory,executable,arguments,prompt,payloadHash,state,approvalId,startedAt,finishedAt,failureReason,createdAt,updatedAt}`.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/projects` | `applicationId?`, 페이지 | ProjectBlueprint 목록 |
| POST `/projects/blueprints` | `{applicationId,gapAnalysisId,ai:AiOptions|null}` | 202 Operation → 정확히 4개 블루프린트 |
| GET `/projects/:id` | — | ProjectBlueprint |
| POST `/projects/:id/select` | `{expectedRevision}` | 선택한 블루프린트와 복사할 파일 manifest |
| GET `/projects/:id/runs` | 페이지 | CliRun 목록 |
| POST `/projects/:id/runs` | `{provider,workingDirectory,prompt}` | APPROVAL_REQUIRED CliRun |
| POST `/projects/:id/runs/:runId/start` | `{expectedRevision,approvalId,detectedVersion,deviceId}` | RUNNING CliRun |
| POST `/projects/:id/runs/:runId/launch` | `{expectedRevision,deviceId,payloadHash,launchStatus:'CLAIMED'|'STARTED'|'UNKNOWN',process?:{pid,startedAt}}` | CliRun |
| POST `/projects/:id/runs/:runId/recover` | `{expectedRevision,deviceId,decision:'REATTACH'|'MARK_FAILED',process?:{pid,startedAt},failureReason?}` | CliRun |
| POST `/projects/:id/runs/:runId/result` | `{expectedRevision,exitCode,commitSha?,stdoutHash,stderrHash}` | VERIFYING 또는 FAILED CliRun |
| POST `/projects/:id/evidence` | `{runId,commitUrl,testCommand,testOutput,exitCode,metrics:[{name,value,unit}],summary}` | 검증 대기 ProjectEvidence |
| POST `/projects/:id/evidence/:evidenceId/verify` | `{expectedRevision}` | 202 Operation → ProjectEvidence |
| GET `/projects/:id/evidence` | 페이지 | ProjectEvidence 목록 |

로컬 앱은 설치 경로와 버전을 실제로 검사한다. `codex`, `claude`, Grok Build의 실행 파일과 인자 형식은 공급자 버전별 지원 목록으로 관리하며 미설치를 성공으로 표시하지 않는다. 작업 폴더·명령·인자·프롬프트를 보여준 후 해당 run의 CLI_EXECUTE 승인을 요청한다. 폴더는 사용자가 선택한 별도 폴더이며 경로 이탈·심볼릭 링크 탈출·기존 파일 덮어쓰기를 차단한다.

start는 deviceId를 원자적으로 고정하며 다른 기기에서 같은 run을 시작할 수 없다. launch 보고는 고정된 deviceId와 payloadHash가 일치해야 한다. 전이는 `NOT_CLAIMED → CLAIMED → STARTED`, `CLAIMED|STARTED → UNKNOWN`, 결과 수신 후 `STARTED → FINISHED`다. result는 RUNNING 상태와 STARTED launchStatus에서만 받는다. recover는 명시적 사용자 확인 후 UNKNOWN에서만 가능하다. REATTACH는 동일 pid/OS 프로세스 시작 시각을 native가 확인한 기존 프로세스에만 연결하여 STARTED로 복구하며 새 spawn을 하지 않는다. MARK_FAILED는 비어 있지 않은 failureReason과 함께 run을 FAILED로 닫는다. 설치 오류 등 spawn 전 실패도 launch UNKNOWN 후 MARK_FAILED로 종료한다.

native는 runId를 unique key로 한 SQLite 실행 레코드를 트랜잭션에서 먼저 CLAIMED로 기록한 뒤 spawn한다. 서버 승인 payloadHash와 실제 executable/arguments/workingDirectory/prompt의 hash를 비교하고 동일 runId를 다시 spawn하지 않는다. 실행 프로세스 식별자를 기록하기 전에 앱이 종료되면 launchStatus를 UNKNOWN으로 취급한다. 이 경우 자동 재실행하지 않고 사용자가 외부 Terminal/프로세스를 확인하여 기존 실행에 재연결하거나 실패로 닫도록 한다. 서버의 멱등 RUNNING 응답 재수신은 새 로컬 실행 권한이 아니다. 서버 CliRun에 `launchStatus: NOT_CLAIMED|CLAIMED|STARTED|UNKNOWN|FINISHED`를 함께 저장한다.

명령은 shell 문자열 결합으로 만들지 않는다. 인자와 프롬프트를 안전하게 전달하며 v1은 외부 Terminal을 사용한다. 내장 PTY는 구현하지 않는다. 거부·실행 파일 없음·프로세스 실패는 실행하지 않거나 FAILED로 기록한다.

`ProjectEvidence`: `{id,revision,projectId,runId,commitUrl,commitSha,testResults,metrics,summary,status:'PENDING'|'VERIFIED'|'REJECTED',verificationMethod,verifiedAt,careerEvidenceId}`. 클라이언트가 입력한 성공 문자열/exitCode만으로 VERIFIED로 바꾸지 않는다. 독립 검증은 원격 커밋 존재, 그 커밋에 연결된 성공 CI/check, 테스트 결과와 측정 artifact를 검증한다. 검증 접근 권한이 없으면 PENDING이다. 검증 후에도 EVIDENCE_USE 승인 전 문서 근거로 사용하지 않는다.

## 9. 채팅 작업공간 `/conversations`

회사별 대화는 지원 ID에 연결된다. 다른 지원의 문서/분석을 자동으로 합치지 않는다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/conversations` | `applicationId?`, 페이지 | `{id,revision,applicationId,title,pinned,createdAt,updatedAt}` 목록 |
| POST `/conversations` | `{applicationId,title?}` | 새 대화 |
| PATCH `/conversations/:id` | `{expectedRevision,title?,pinned?}` | 대화 |
| GET `/conversations/:id/messages` | 페이지 | Message 목록 |
| POST `/conversations/:id/messages` | `{text,context:{documentId?,versionId?,evidenceIds:[]},ai:AiOptions,accessMode}` | 202 Operation |
| POST `/conversations/:id/archive` | `{expectedRevision}` | 보관된 대화 |

`Message`: `{id,conversationId,role:'USER'|'ASSISTANT'|'SYSTEM',text,attachments:MessageAttachment[],operationId,createdAt}`. `MessageAttachment`는 `{type:'DOCUMENT_VERSION',id:string,documentId:string,title:string}` 또는 `{type:'EVIDENCE',id:string,title:string}` 또는 `{type:'APPROVAL',id:string}`이다. id는 각 버전·근거·승인 리소스의 ID이며 서버가 확인한 참조만 포함한다. 첨부 리소스도 소유권과 지원 범위를 확인한다. 응답은 진행 상태·분석 결과·문서 제안·승인 요청을 구조화된 attachment로 포함할 수 있다. 임의 도구 이름이나 shell 명령을 그대로 실행하는 API는 없다.

채팅 context의 문서·버전은 작업을 넣는 시점의 불변 버전으로 고정한다. documentId만 있으면 해당 문서의 최신 버전을 선택하고 버전이 없으면 명확한 입력 오류를 반환한다. versionId만 있으면 소유 문서를 유도한다. 버전의 근거 참조도 해당 지원의 근거 사용 승인을 확인하며, 문서의 미검증 제안 문장을 사실 근거로 승격하지 않는다. 채팅 응답만으로 문서를 변경·확정하거나 승인을 생성·소비하지 않는다. 첨부 카드의 변경 검토·확정은 기존 문서/승인 API와 사용자 동작을 사용한다.

`effort=LOW|MEDIUM|HIGH`; `accessMode=SUGGEST|CONFIRM_ACTIONS`. 이 설정은 개별 승인 게이트를 해제하지 않는다. 고정한 공고·문서는 `/pins`의 `GET`, `POST {resourceType,resourceId}`, `DELETE /:id`로 관리한다. 열린 탭과 임시 입력 상태는 로컬 UI 상태다.

## 10. 캘린더·면접·오퍼·루틴

### 캘린더 `/calendar/events`

`CalendarEvent`: `{id,revision,applicationId,type:'INTERVIEW'|'DEADLINE'|'FOLLOW_UP'|'CUSTOM',title,startsAt,endsAt,timeZone,source:'LOCAL'|'GOOGLE',externalId,notes,createdAt,updatedAt}`.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/calendar/events` | `from`, `to`, `applicationId?`, 페이지 | 일정 목록 |
| POST `/calendar/events` | `{applicationId?,type,title,startsAt,endsAt,timeZone,notes?}` | 일정 |
| PATCH `/calendar/events/:id` | `{expectedRevision,title?,startsAt?,endsAt?,timeZone?,notes?}` | 일정 |
| DELETE `/calendar/events/:id` | `If-Match: <revision>` | 204 |

조회 기간은 최대 366일, 끝 시각은 시작 이후다. IANA timeZone을 보관한다. Google 일정 삭제/수정은 scope와 명시적 사용자 동작이 있어야 하며 로컬 삭제를 자동 전파하지 않는다.

### 면접 `/interviews`

`InterviewSession`: `{id,revision,applicationId,title,scheduledAt,durationMinutes,eventId,evidenceIds,companySources,notes,reflection,createdAt,updatedAt}`.

- `GET /interviews`: 지원·기간별 목록.
- `POST /interviews`: `{applicationId,title,scheduledAt,durationMinutes?,evidenceIds:[],companySources?:CompanySource[],notes?}` → InterviewSession과 연결된 내부 일정.
- `GET /interviews/:id`: 상세.
- `PATCH /interviews/:id`: `{expectedRevision,title?,scheduledAt?,companySources?:CompanySource[],notes?,reflection?}` → 수정된 세션.
- `POST /interviews/:id/prepare`: `{expectedRevision,ai:AiOptions|null}` → 202 Operation → `{questions:[{question,requirement,evidenceIds}],starAnswers:[{evidenceIds,situation,task,action,result,needsInput:[]}],research:[{claim,sourceUrl,accessedAt,verificationStatus:'USER_PROVIDED'}]}`.

`CompanySource`는 `{sourceUrl,sourceText,accessedAt}`이며 사용자가 확인한 회사 자료의 HTTPS URL, 원문(1~20,000자), 조회 시각(RFC3339, 미래 불가)을 최대 10개 보관한다. URL은 출처 링크이며 서버가 자동 조회했다고 표시하지 않는다. 회사 조사 결과는 저장된 원문에서 정확히 발췌하고 출처·조회 시각·USER_PROVIDED 상태를 함께 표시한다. 원문이 없으면 자료 입력을 요청한다. AI 작업도 요청 당시 자료를 고정하며 모델이 만든 회사 사실·URL로 교체하지 않는다. 회사 조사에는 출처를 붙인다. 확인하지 않은 회사 사실이나 사용자 경험은 생성하지 않고 추가 입력으로 남긴다. 면접 회고는 해당 지원에만 연결한다.

### 오퍼 `/offers`

- `GET /offers`: 지원별 목록.
- `POST /offers`: `{applicationId,company,annualSalaryMinor,currency,equity?,benefits?,deadline?,notes?}` → Offer.
- `GET /offers/:id`, `PATCH /offers/:id {expectedRevision,annualSalaryMinor?,currency?,equity?,benefits?,deadline?,notes?}`.
- `GET /offers/compare?ids=<id,id>` → `{offers:[],comparison:{sameCurrency,fields:[]}}`. 2~10개를 비교한다.

금액은 0 이상의 안전한 정수다. 다른 통화의 금액을 환율 없이 합산·순위화하지 않는다. 오퍼 기록은 법적 수락이나 외부 회사에 대한 의사 전달이 아니다.

### 루틴 `/routines`

`Routine`: `{id,revision,applicationId,title,kind,dueAt,status,reason,createdAt,updatedAt}`.

- `kind=DEADLINE|FOLLOW_UP|INTERVIEW_PREP|FOLLOW_UP_EMAIL`
- `status=SUGGESTED|CONFIRMED|DONE|DISMISSED`
- `GET /routines`, `POST /routines {applicationId,title,kind,dueAt}`.
- `POST /routines/suggest {}` → 중복되지 않는 제안 목록. 마감 72시간 이내, 지원 후 7일 응답 없음, 면접 72시간 이내를 기본 기준으로 한다.
- `PATCH /routines/:id {expectedRevision,status}` → `SUGGESTED → CONFIRMED|DISMISSED`, `CONFIRMED → DONE|DISMISSED`만 허용한다.

자동 외부 발송은 하지 않는다. 후속 메일은 초안 제안까지만 하며 전송 동작은 별도 명시적 사용자 확인 없이는 제공하지 않는다.

## 11. 승인 `/approvals`

```ts
type Approval = {
  id: string; revision: number;
  kind: 'EVIDENCE_USE' | 'DOCUMENT_FINALIZE' | 'APPLICATION_SUBMIT' | 'CLI_EXECUTE';
  applicationId: string;
  targetId: string;
  targetRevision: number | null;
  payloadHash: string;
  status: 'PENDING' | 'APPROVED' | 'DENIED' | 'EXPIRED' | 'CONSUMED';
  expiresAt: string | null;
  decidedAt: string | null;
  consumedAt: string | null;
  createdAt: string; updatedAt: string;
};
```

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/approvals` | `applicationId?`, `status?`, 페이지 | Approval 목록 |
| POST `/approvals` | `{kind,applicationId,targetId,targetRevision?}` | 서버가 대상 내용을 조회해 hash를 계산한 Approval |
| GET `/approvals/:id` | — | Approval과 사용자 검토용 대상 요약 |
| POST `/approvals/:id/decision` | `{expectedRevision,decision:'APPROVED'|'DENIED'}` | 결정된 Approval |

클라이언트가 임의 hash·소유권·승인 완료 상태를 지정하지 않는다. 승인 유효 기간은 기본 24시간이다. 원문 불변 근거 사용 승인은 해당 지원 범위에 지속 적용할 수 있으나 보관/폐기 시 새 사용을 막는다. 문서 확정·제출·CLI 실행 승인은 정확한 대상에 1회 소비한다. 내용을 바꾸면 이전 승인은 `APPROVAL_STALE`이다. DENIED를 재승인하려면 사용자가 새 승인 요청을 시작해야 하며 시스템이 자동 재요청하지 않는다.

## 12. AI·BYOK `/ai`

`provider=OPENAI|CLAUDE|GEMINI|GROK`. 관리형 AI는 OpenAI만 지원한다. 모델 목록은 서버의 실제 지원 구성에서 반환하며 없는 모델명을 기본값으로 꾸미지 않는다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/ai/models` | `provider?`, `credentialMode?` | `{provider,model,label,available,supportedEfforts:[]}` 목록 |
| GET `/ai/keys` | — | `{provider,lastFour,configured,updatedAt}` 목록 |
| PUT `/ai/keys/:provider` | `{key}` | 키 메타데이터만 반환 |
| DELETE `/ai/keys/:provider` | — | 204 |
| POST `/ai/keys/:provider/test` | `{}` | `{valid,checkedAt,errorCode?}` |
| POST `/ai/generate` | `{ai:AiOptions,prompt,applicationId,evidenceIds:[]}` | 202 Operation → `{text,citations:[],usage}` |
| GET `/ai/usage` | `from?`, `to?`, 페이지 | AiUsage 목록 |

`AiUsage`: `{id,operationId,provider,model,managed,inputTokens,outputTokens,costMicroCredits,status:'RESERVED'|'SETTLED'|'RELEASED',createdAt}`.

BYOK는 AES-256-GCM으로 암호화하고 레코드별 무작위 nonce, 사용자·공급자·키 버전의 AAD를 사용한다. 마스터 키는 저장소/DB와 분리한 환경 secret이다. 키 평문을 응답·로그·분석 이벤트에 기록하지 않는다. 삭제된 키는 새 호출에 사용할 수 없다.

관리형 요청은 활성 구독과 잔여 크레딧을 확인하고 최대 예상 비용을 예약한다. 실제 공급자 사용량으로 1회 정산하고 사용하지 않은 예약은 반환한다. 공급자 실패·타임아웃의 비용 확인이 불가능하면 이력을 남겨 조정하며 비용을 임의로 0으로 만들지 않는다. BYOK는 공급자 비용을 사용자 계정이 부담하며 관리형 크레딧을 차감하지 않는다.

AI 응답은 제안이다. 모델 출력만으로 사실 검증·문서 확정·지원 제출·CLI 실행을 통과시키지 않는다.

## 13. 결제 `/billing`

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/billing` | — | `{subscriptionStatus,plan,periodEndsAt,balanceMicroCredits,reservedMicroCredits}` |
| POST `/billing/checkout` | `{planId}` | `{url,expiresAt}` |
| POST `/billing/portal` | `{}` | `{url}` |
| GET `/billing/ledger` | 페이지 | `{id,type,amountMicroCredits,balanceAfter,referenceId,createdAt}` 목록 |
| POST `/billing/webhook` | Stripe 원문 body + `Stripe-Signature` | `{received:true}` |

`balanceMicroCredits`는 사용 가능한 잔액이며 환불이 이미 예약·사용된 크레딧을 초과하면 음수가 될 수 있다. 음수는 미정산 크레딧 부채를 뜻하며 새 관리형 AI 요청을 허용하지 않는다. 예약 취소·정산으로 잔액이 조정되어도 전체 환불 차감액과 원장 보존 관계를 유지한다.

가격·return URL은 서버 allowlist로 선택한다. 클라이언트가 청구액·사용자 ID·크레딧 지급량을 결정하지 않는다. 웹훅은 raw body HMAC과 5분 timestamp 허용 범위를 검증한다. 이벤트 ID unique 제약, 구독 소유권, 결제 상태를 검증한 뒤 원장과 구독을 원자적으로 갱신한다. 중복·역순 웹훅으로 크레딧을 중복 지급하거나 취소 상태를 되돌리지 않는다. 결제 실패/환불/구독 취소를 반영하고 실제 공급자 확인이 안 되면 완료로 표시하지 않는다.

## 14. Google와 채용 사이트 연동

### Google `/integrations/google`

- `GET /integrations/google` → `{enabled,connected,scopes:[],gmailStatus,calendarStatus,lastSyncedAt}`.
- `POST /integrations/google/connect {codeChallenge,codeChallengeMethod:'S256',redirectUri:'push://integrations/google/callback'}` → `{authorizationUrl,state}`. 필요한 scope에 별도로 동의한다.
- `GET /integrations/google/callback?code=...&state=...`: 공급자 콜백. 최초 로그인 사용자 ID·state·PKCE challenge에 연결한 60초짜리 integrationCode만 담아 허용된 딥링크로 302한다.
- `POST /integrations/google/complete {integrationCode,codeVerifier}`: 인증된 시작 사용자와 PKCE가 일치할 때에만 공급자 토큰을 연결하고 GET 상태와 같은 응답을 반환한다. 로그인용 `/auth/exchange`와 별개이며 다른 Push 계정으로 연결할 수 없다.
- `DELETE /integrations/google` → 204. 공급자 토큰 폐기와 향후 동기화 중지.
- `POST /integrations/google/sync {}` → 202 Operation.
- `GET /integrations/google/messages?applicationId=...` → 저장한 지원 관련 메일 메타데이터 목록.
- `GET /integrations/google/events?from=...&to=...` → 연동 일정 목록.
- `POST /integrations/google/messages/:id/link {applicationId}` → 사용자가 확인한 지원 연결.

앱 실행 시 증분 동기화한다. Gmail historyId와 Calendar syncToken/pageToken은 서버에서 관리하며 외부 응답을 전부 저장한 후 체크포인트를 갱신한다. 페이지 누락·부분 실패 시 커서를 앞당기지 않는다. 만료된 커서는 제한된 재동기화로 복구한다. 취소/삭제 이벤트도 반영한다. 회사명 유사성만으로 메일을 다른 지원에 확정 연결하지 않는다.

Gmail 제한 범위 검증 전에는 `GOOGLE_GMAIL_BETA_ENABLED=false`가 기본이다. Calendar도 실제 scope가 있어야 호출한다. 비활성은 403, 연결 없음은 409, 공급자 오류는 502다. 메일 본문은 로그·분석 이벤트·테스트 픽스처로 쓰지 않는다.

### 채용 사이트 `/integrations/job-sites`

`GET /integrations/job-sites` → `[{provider:'WANTED'|'JUMPIT'|'JOBKOREA',enabled,permissionVerified,mode:'MANUAL_CHECKLIST'|'ASSISTED_FILL',supportedFields:[],checklist:[]}]`.

서비스 약관/공식 허가가 확인되지 않으면 자동 입력·제출은 기본 비활성이다. 수동 체크리스트는 항상 제공한다. 활성화된 DOM 자동 입력도 사용자가 검토할 필드별 차이를 반환하고 최종 제출은 해당 지원의 승인을 별도로 요구한다. 원격 채용 페이지는 Tauri의 로컬 파일·키·CLI IPC 권한을 갖지 않는다.

## 15. 비동기 작업 `/operations`

`Operation`: `{id,type,applicationId,status:'QUEUED'|'RUNNING'|'NEEDS_INPUT'|'SUCCEEDED'|'FAILED'|'CANCELLED',progress,result,error,inputRequest,createdAt,updatedAt}`. inputRequest는 NEEDS_INPUT 외에는 null이다. `progress`는 실제 단계의 0~100 값 또는 null이다.

- `GET /operations/:id`: 상태와 완료 결과.
- `POST /operations/:id/input {fields:{[name:string]:string},sourceId?}`: NEEDS_INPUT의 inputRequest에 명시된 필드만 받는다. 소유권/입력 형식을 검증한 뒤 같은 작업을 QUEUED로 전환하며 완료 작업에는 409를 반환한다.
- `POST /operations/:id/cancel {}`: 취소 요청. 이미 완료된 작업은 결과를 유지한다.
- `GET /operations/:id/events`: 인증된 SSE, `Last-Event-ID`로 재연결.

SSE 이벤트는 `progress`, `delta`, `result`, `error`이며 각 이벤트에 `id`가 있다. SSE는 관찰 수단이다. 연결이 끊겨도 작업을 중복 시작하지 않고 GET으로 최종 상태를 확인한다. 종료 이벤트는 1회 논리 결과를 나타낸다. 취소가 이미 발생한 외부 부작용을 되돌렸다고 주장하지 않는다.

## 16. 오프라인 저장과 동기화 `/sync`

SQLite는 계정별로 분리하며 문서 초안과 outbox를 트랜잭션으로 함께 저장한다. Zustand는 실행 중 상태를 제공하고, 영속 데이터의 원본은 SQLite/서버다. 인증 토큰과 BYOK 키를 일반 Zustand persist/localStorage에 저장하지 않는다.

| 메서드·경로 | 요청 | 응답 |
| --- | --- | --- |
| GET `/sync/changes` | `cursor?`, `limit?` | `{changes:[{sequence,resourceType,resourceId,revision,deleted,data}],nextCursor,hasMore}` |
| POST `/sync/mutations` | `{clientId,mutations:[{mutationId,resourceType,resourceId,expectedRevision,action,payload}]}` | `{results:[{mutationId,status,resource?,error?}]}` |

배치는 최대 50개이며 각 mutation을 개별 원자 처리한다. 같은 사용자/clientId/mutationId는 중복 실행하지 않는다. 실패한 항목을 성공처럼 제거하지 않는다. 삭제는 tombstone으로 전달한다. 커서 만료는 409 `SYNC_CURSOR_EXPIRED`로 전체 재조회 필요를 알린다.

동기화 대상은 명시적으로 허용된 로컬 편집(문서 초안·메모·내부 일정)만이다. 승인·제출·CLI 시작·결제·외부 메일 전송을 오프라인 outbox가 자동 실행하지 않는다. 충돌 시 로컬 초안과 서버 버전을 함께 보존하고 사용자가 비교·병합 후 최신 revision으로 저장한다.

## 17. FE·네이티브 경계

- React HTTP는 ky, 상태 관리는 Zustand, 구조는 FSD v2.1이다.
- ky 인스턴스·공통 DTO·오류 해석은 `shared/api`, 인증 세션은 `shared/auth`, SQLite/native bridge는 용도별 shared 세그먼트에 둔다.
- 화면 전용 요청 조합과 상태는 `pages/<slice>`에 둔다. 실제 여러 화면에서 쓰는 승인·문서 편집 동작만 `features`로 추출한다.
- public `index.ts`로만 슬라이스 외부에 공개하고 상위 계층/동일 계층 다른 슬라이스 import를 금지한다.
- 허용 CORS origin은 개발 Vite와 Tauri origin의 명시 allowlist다. credentials와 와일드카드 origin을 결합하지 않는다.
- native command는 파일 선택·안전한 파일 출력·SQLite·CLI 탐지·승인된 실행·딥링크 처리로 제한한다. 원격 브라우저 내용은 native command를 호출할 수 없다.
- 클라이언트 validation은 UX를 위한 것이며 모든 권한·승인·금액·상태·근거 검증은 서버에서 다시 수행한다.

## 18. 정확한 공통 DTO와 작업 결과

### AI 선택

```ts
type AiOptions = {
  provider: 'OPENAI' | 'CLAUDE' | 'GEMINI' | 'GROK';
  model: string;
  credentialMode: 'MANAGED' | 'BYOK';
  effort: 'LOW' | 'MEDIUM' | 'HIGH';
};
```

AI 실행 경로는 이 값을 필수로 받는다. `ai:null`을 허용한다고 명시한 공고 분석·문서 원문 발췌 생성·블루프린트·면접 준비만 결정적 규칙 기반 동작을 한다. MANAGED는 OPENAI만 허용한다. BYOK 키 누락 시 409 INTEGRATION_REQUIRED, 관리형 설정 누락 시 503 NOT_CONFIGURED다. 다른 자격증명이나 결제 모드로 자동 fallback하지 않는다. 문서 생성의 ai:null은 사용자가 선택한 원문을 그대로 발췌해 템플릿에 배치하는 로컬 검증 가능한 경로이며 AI 재작성이라고 표시하지 않는다. AiUsage의 managed는 credentialMode에서 서버가 계산한 값이다.

### 문서·블루프린트·오퍼 DTO

```ts
type EvidenceRef = { evidenceId: string; start: number; end: number };
type BlockInput = { id: string; text: string; evidenceRefs: EvidenceRef[] };
type RevisionProposal = {
  id: string; documentId: string; sourceVersionId: string; sourceRevision: number;
  selection: { from: number; to: number; text: string };
  replacement: string; evidenceRefs: EvidenceRef[];
  claimStatus: 'SUPPORTED' | 'NEEDS_REVIEW' | 'UNSUPPORTED';
  createdAt: string;
};
type BlueprintTask = { id: string; title: string; description: string; acceptance: string[] };
type BlueprintMetric = { name: string; unit: string; measurement: string; target: number | null };
type ProjectManifest = {
  projectId: string; blueprintRevision: number;
  files: { path: string; encoding: 'utf8'; content: string; sha256: string }[];
};
type Offer = {
  id: string; revision: number; applicationId: string; company: string;
  annualSalaryMinor: number; currency: string; equity: string | null;
  benefits: string[]; deadline: string | null; notes: string;
  createdAt: string; updatedAt: string;
};
```

ProjectBlueprint의 tasks는 BlueprintTask[], metrics는 BlueprintMetric[], estimatedEffort는 `{minHours:number,maxHours:number}`, state는 `DRAFT|SELECTED|IN_PROGRESS|VERIFIED|ARCHIVED`다. select 응답은 `{blueprint:ProjectBlueprint,manifest:ProjectManifest}`다. manifest 경로는 상대 경로만 허용하고 `..`, 절대 경로, 중복 경로를 거부한다. 파일 수 최대 100개, 전체 UTF-8 크기 최대 5 MiB다. 프로젝트 파일은 아직 실행하지 않은 생성 결과이며 완료 경험으로 기록하지 않는다.

Offer 비교의 fields는 `[{key:'annualSalaryMinor'|'equity'|'benefits'|'deadline',values:[{offerId,value:string|null}]}]`다. 빈 notes는 빈 문자열, 빠진 benefits는 []로 정규화한다. 문서 selection은 원본 TipTap/ProseMirror 문서 위치이며 from < to, 선택 원문 text 일치를 검증한다.

### 오프라인 초안과 mutation

```ts
type DocumentDraft = {
  id: string; revision: number; documentId: string; applicationId: string;
  baseDocumentRevision: number; content: object; blocks: BlockInput[];
  createdAt: string; updatedAt: string;
};
type SyncMutation =
  | { mutationId: string; resourceType: 'DOCUMENT_DRAFT'; resourceId: string; expectedRevision: 0; action: 'CREATE'; payload: { documentId: string; baseDocumentRevision: number; content: object; blocks: BlockInput[] } }
  | { mutationId: string; resourceType: 'DOCUMENT_DRAFT'; resourceId: string; expectedRevision: number; action: 'UPDATE'; payload: { baseDocumentRevision: number; content: object; blocks: BlockInput[] } }
  | { mutationId: string; resourceType: 'APPLICATION'; resourceId: string; expectedRevision: number; action: 'UPDATE_NOTES'; payload: { notes: string } }
  | { mutationId: string; resourceType: 'CALENDAR_EVENT'; resourceId: string; expectedRevision: number; action: 'UPDATE_LOCAL'; payload: { title?: string; startsAt?: string; endsAt?: string; timeZone?: string; notes?: string } };
type SyncResult = {
  mutationId: string; status: 'APPLIED' | 'CONFLICT' | 'REJECTED';
  resource: DocumentDraft | Application | CalendarEvent | null;
  error: { code: string; message: string; currentRevision?: number } | null;
};
```

클라이언트가 만든 draft UUID는 해당 사용자 범위에서 CREATE가 1회만 허용된다. CREATE의 expectedRevision은 0, UPDATE는 1 이상이다. 온라인 대응 경로는 `GET /documents/:id/drafts`, `POST /documents/:id/drafts {id,baseDocumentRevision,content,blocks}`, `PATCH /documents/:id/drafts/:draftId {expectedRevision,baseDocumentRevision,content,blocks}`다. 새 draft 응답 revision은 1이다. 저장은 동일한 본문/근거/소유권 검증을 수행하되 미지원 문장은 검토 상태로 보존한다. draft 저장은 DocumentVersion 생성/확정이 아니며, 명시적으로 versions API에 적용할 때 baseDocumentRevision 충돌을 확인한다. 다른 mutation은 정의하지 않은 action/필드를 거부한다. GOOGLE 일정은 UPDATE_LOCAL로 변경할 수 없다.

### Operation·SSE

Operation.type은 아래 결과 kind와 같다. 성공 전 result는 null, 성공 후에는 아래 tagged 결과만 반환한다. error는 실패 시 `{code,message,retryable}`이며 그 외에는 null이다. applicationId는 파일 import처럼 아직 지원 범위가 없는 작업만 null이다.

| type / result.kind | result.value |
| --- | --- |
| EVIDENCE_IMPORT | `{evidence:CareerEvidence[],warnings:[{code,message,sourceLocation:string|null}]}` |
| JOB_ANALYSIS | GapAnalysis |
| DOCUMENT_GENERATE | `{document:Document,version:DocumentVersion}` |
| DOCUMENT_REVISE | RevisionProposal |
| PROJECT_BLUEPRINTS | ProjectBlueprint[] (4개) |
| PROJECT_VERIFY | ProjectEvidence |
| INTERVIEW_PREPARE | 10절 prepare의 questions/starAnswers/research 객체 |
| AI_GENERATE | `{text:string,citations:EvidenceRef[],usage:AiUsage}` |
| CHAT_MESSAGE | `{userMessage:Message,assistantMessage:Message,approvalIds:string[]}` |
| GOOGLE_SYNC | `{messagesUpserted:number,eventsUpserted:number,eventsDeleted:number,completedAt:string}` |
| APPLICATION_SUBMIT | Submission |

SSE data는 `{operationId,sequence,payload}`다. progress payload는 `{status,progress,step:string}`, delta는 `{text:string}`, result는 위 result 객체, error는 위 error 객체다. sequence는 작업 내 증가하는 정수이고 SSE id는 `operationId:sequence`다. 재연결 이벤트는 중복될 수 있으므로 FE가 id로 중복 제거한다. 복구 가능한 이벤트 보존 기간은 24시간이며 그 이후에는 GET operation으로 최종 상태를 확인한다. NEEDS_INPUT은 `{code,message,fields:[{name,label,type:'TEXT'|'FILE'}]}`의 inputRequest를 함께 반환한다.

## 19. 필수 수용 테스트

| ID | 시나리오 | 통과 조건 |
| --- | --- | --- |
| API-01 | 타 사용자 및 다른 지원의 ID로 조회·문서/버전 참조 | 404 또는 범위 오류, 기존 데이터 유지 |
| API-02 | 같은 문서 revision을 두 기기에서 수정 | 하나만 성공, 다른 초안은 충돌로 보존 |
| API-03 | 원문 20%를 AI가 90%로 바꾸고 확정 | UNSUPPORTED_CLAIM, 승인해도 통과 불가 |
| API-04 | 근거 → 공고 분석 → 문서 → 프로젝트 → 면접 | 불변 원문·ID·출처 역추적 가능 |
| API-05 | 지원 A 승인으로 B 제출/변경된 문서 제출 | 거부, 외부 요청 없음 |
| API-06 | 같은 멱등 키 재요청·다른 body | 동일 결과/409, 중복 부작용 없음 |
| API-07 | OAuth state/PKCE/코드 재사용/refresh 회전 | 올바른 왕복만 성공, 이전 토큰 거부 |
| API-08 | Stripe 중복·역순·위조 웹훅 | 원장 1회, 잘못된 서명 거부 |
| API-09 | 세 CLI 각각 미설치·거부·실패·실제 검증 | 무단 실행 없음, 실제 증빙 없이 VERIFIED 불가 |
| API-10 | Gmail/Calendar 다중 페이지·부분 실패·커서 만료 | 누락 없이 복구, 성공 전 cursor 이동 없음 |
| API-11 | 로그에 키·토큰·원문·메일 sentinel 입력 | 로그/분석 이벤트에서 발견되지 않음 |
| API-12 | PDF/DOCX 한글·링크·페이지·ATS | 실제 생성 파일의 렌더/텍스트 검사 통과 |
| API-13 | 오프라인 재시작·계정 전환·outbox 재전송 | 초안 보존, 계정 격리, 중복 없음 |
| API-14 | 공급자 미설정/기능 플래그 꺼짐 | 명시 오류, 가짜 성공이나 임의 과금 없음 |
| API-15 | 트랜잭션 commit 실패 | 성공 응답 없음, 중간 변경 없음 |

로컬 자동 테스트와 실제 공급자 검증은 구분한다. Apple 서명·공증, 운영 업데이트, 실제 OAuth/결제/메일, 20명 베타는 자격증명과 참가자가 준비된 후 별도로 검증한다.
