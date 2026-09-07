# Push implementation gauntlet

Goal: implement docs/plan.md, replace empty API contract, create private push-dot/push-fe and push-dot/push-be and attach pinned submodules.

## Locked references and checks

- Product contract: docs/plan.md, retrieved 2026-09-07. Final UI decision supersedes initial right-panel requirement.
- Visual reference: docs/no-right-sidebar-prototype-v6.png (1672×941), inspected. Historical Aside captures are missing; no claim of comparison to those captures.
- Domain reference: docs/plan.md and docs/api.md only. Fresh implementation; user forbids legacy/Super Resume code references.
- Comparison mode: champion/challenger against product specification (cross-medium); UI compares rendered candidates against prototype. Each critic receives neutral immutable candidate versions and version-matched checks, never this progress file.
- Acceptance: evidence lineage; two-job isolation; fabricated claims blocked at finalization; PDF/DOCX Korean text, pagination and links; OAuth return and expiration; offline editing/conflicts; webhook idempotency; CLI absent/denied/failure/verified; no unapproved application submission; secrets absent from logs; keyboard/dark/narrow UI; actual beta 16/20 successful journeys.
- Tools: Codex collaboration spawn_agent, fresh critics via fork_turns=none; inherited model settings. FE and BE parallel after API publication, as explicitly requested; only lead spawns. No background automation.

## Tasks

1. Fresh domain implementation and server API — implemented;30 real PostgreSQL tests pass.
2. Tauri desktop shell and company chats without right sidebar — local execution/visual comparison pass.
3. Vault, analysis, editor, immutable versions and PDF/DOCX — local API/UI/export checks pass.
4. Pipeline, calendar, interviews and offers — implemented; company-source and rejection UI/API verification pass.
5. Four blueprints and approved external CLI execution/evidence verification — local subprocess and provider-boundary checks pass.
6. OAuth, Google, AI/BYOK, billing, flagged job adapters — implemented; actual external account verification pending readiness.
7. Release signing/notarization/updater and20-person private beta — local Universal app/DMG ready; external credentials/participants pending.
8. Integrated acceptance and fresh critic convergence — scoped visual/storage/native/billing checks pass; company research and durable ACK coverage checks pass.
9. Commit units, PRs to develop, squash merge and pin submodules — FE/BE squash merged with identical tested trees; final gitlinks pinned, root PR prepared for develop.

## Repository state

Original untracked docs preserved at /Volumes/Untitled/Documents/Github/push-workspace/docs.
Isolated implementation: /Volumes/Untitled/Documents/Github/push-workspace-implementation, feat/implement-push-plan based on develop.
Private repositories created: https://github.com/push-dot/push-fe and https://github.com/push-dot/push-be.
No existing executable code or test suite; baseline consists of README and untracked user docs.

## Evidence and next action

Initial API, server and frontend checkpoints have passed scoped checks recorded below. Full gauntlet convergence remains pending. External integrations must fail closed when credentials/permissions are absent; never label mocks as live integrations.

2026-09-07 user steering: external VPS/OAuth/Stripe/Apple/beta not prepared; proceed with implementation and local verification. External launch criteria stay explicitly pending, not a reason to stop local work.
Backend builder agent: /root/backend_builder, owns push-be. Lead owns docs/api.md. Running actual PostgreSQL on 127.0.0.1:55432/push_test for integration tests.

2026-09-07 user steering: React HTTP MUST use ky; FSD structure with explicitly invoked feature-sliced-design skill. Loaded /Users/cyjoon/.agents/skills/feature-sliced-design/SKILL.md and layer/framework references. FSD v2.1 pages-first, index public APIs, no same-layer slice imports; no speculative layers. Frontend task must enforce this and test request auth/errors/no mutation retries.
2026-09-07 user steering: state management MUST use Zustand (쥬스턴드). Required FE stack: React + ky + Zustand + FSD v2.1, Tauri v2, SQLite, TipTap. Place stores at owning layer; no app-store upward imports from pages.
2026-09-07 user explicitly authorized FE/BE parallel after API doc. Initial contract exists; overrides sequential implementer rule. Spawned /root/frontend_builder owning push-fe, while /root/backend_builder owns push-be; lead owns docs/api.md. Shared interface changes coordinated between builders. Lead owns root files, integration tests, immutable comparisons.

Backend candidate checkpoint e7a80fe saved immutably in .artifacts/backend-a. Lead independently ran TEST_DATABASE_URL=local PostgreSQL go test -race -count=1 ./... on that snapshot: PASS (1.675s). This establishes phase-1 tests only, not complete server/runtime or external integration success. Next: full executable + auth/billing tests and challenger snapshot.

Priority correction from user: API must be written in original workspace docs/api.md FIRST; fresh development, no existing-code/Super Resume references. Both builders paused while lead authored full contract and then resumed parallel. Original API document is now present and mirrored to implementation worktree. User requests highest code quality; no lower quality gate.
API preliminary independent check found 8 blocking ambiguities; repaired all in authoritative document. Fresh critic /root/api_spec_critic compared immutable docs: A=revised contract, B=earlier contract; verdict A, gap native launch/recovery reporting absent. Added explicit launch/recover DTO and device-bound state transitions. API docs still design, not claims of implemented routes.

Authoritative API now includes async 202 full Operation in data (poll data.id), explicit source-excerpt ai:null document generation, and exact native launch/recovery reporting. Both builders notified. FE reports 4 tests green (not yet independently verified); first runnable UI pending. Lead rewrote scripts/smoke.mjs against authoritative contract; node syntax check passes, live execution pending contract-ready server. API original file is visible at /Volumes/Untitled/Documents/Github/push-workspace/docs/api.md.

Live API smoke PASS run 4172f759-0677-4e34-ab61-13f431e3ffd5 against immutable binary .artifacts/runtime/push-api-next; exact binary/script hashes and scope in .artifacts/runtime/smoke.json. Auth, idempotency, evidence, unsupported claims, revision conflict, application isolation, submission draft approval, 4 blueprints, 3 CLI denials, Google flag checked over HTTP. Not complete frontend/native/provider evidence.

Frontend immutable checkpoint .artifacts/frontend-a: production build and 8 tests passed. Real browser at 1672×941, API connected: evidence creation, job/application creation, scoped evidence approval, React/TypeScript analysis and source-excerpt document v1 saved successfully. Approval review currently lacks human-readable payload and was returned to FE for correction. Browser journey continuing; no final visual/accessibility/export pass claimed. Native builder reports 8 Rust checks; independent build/runtime verification pending.

Second immutable backend acb52d8: lead go test -race PASS (1.686s), expanded live smoke PASS 3af5acbc-d64d-405f-8765-5c4ed8ba4a0e including interview evidence, calendar link, cross-currency offer comparison, routine confirmation. Exact hashes in .artifacts/runtime/smoke-b.json. Immutable FE 0414fe0 build/FSD/10 tests PASS, but PDF raster inspection found missing glyphs despite text extraction passing. Returned font embedding defect to builder; no PDF visual pass yet.

Independent FE critic comparison A=e1ec421 vs B=0414fe0 chose A but retained blockers: actual delayed save/load erases newer edits, Asia/Seoul calendar month bounds wrong. FE assigned fixes plus remaining pin/NEEDS_INPUT/offline memo/calendar features. Root e1ec421 immutable build/FSD/12 tests pass. PDF raster now visibly correct. DOCX rendered via bundled LibreOffice using task-local FONTCONFIG_FILE with licensed NanumGothic: Korean and links visibly correct; font embedding portability improvement assigned.
Independent BE/native critic A98b5461 vs Bacb52d8 chose A with blockers: native result missing commitSha, server-start/local-claim crash dead end, Gemini missing grounded prompt, AI worker restart recovery absent. Builders assigned corrections; not a convergence verdict. Root backend98b5461 race tests/vet pass. Actual Tauri0219d1e shell launched, duplicated titlebar fixed, maximize/restore observed. Universal build/signing and remaining interactions not yet certified.

Root immutable FE4ec3b7e: build/FSD19 unit+export tests PASS,3 actual API tests PASS against both acb52d8 and a68a139 (export/import resume/sync+pins). Light theme at800×700 corrected visually (output/playwright/frontend-d-light-800.png); all tested buttons named, Tab outline2px, 720px settings no horizontal overflow. Embedded DOCX rendered through bundled LibreOffice+licensed font configuration,4pages Korean+links visually readable. Native02730a8 Universal executable independently lipo-verified x86_64 arm64.
New FE critic A4ec3b7e vsBe1ec421 choseA but actual handler reproduction found document-switch Acontent→Bsave contamination while Bload pending; builder reproduced with realReact component and is repairing. Native critic A02730a8 vsB0219d1e choseA but rejected-start PREPARED retry and directory replacement before spawn remain; builder repairing. No convergence claimed.
Remote feature checkpoints pushed; BE f7b63f4 GitHub CI PASS34105847763; FE4ec3b7e CI PASS34106321717. PRs deferred until completion per project git rule. Root expandedHTTPsmoke againsta68a139 PASS1f0ef12d-3e25-4a73-9de8-51b7ac0fe538.

Final local checkpoints (2026-09-07): BE29d8efc independently passed28 PostgreSQL race tests/vet/build; actualHTTP smoke bbe3fb74-dc35-4153-a7fe-b2203fea15f4 passed all14 lifecycle checks. BE4d68329 adds sensitive request log sentinel coverage,29 race tests/vet passed. Provider tests use controlled HTTP fixtures, never actual-account claims.
Billing fresh critic A476ee32 vsBaf1bd0b returned gap:null within refund conservation/equal-second entitlement ordering. Native fresh critic A238a10e vsB02730a8 returned gap:null for durable attempt/recovery/directory identity/actual subprocess scope. Root6b7fe1d adds official singleton deep-link integration and removes unused SQL plugin;25Rust tests/clippy passed. Independent actualsecondprocess exits0 in0.030s, sameGUI restored; --push-run captures actualexit7, replayrejected with unchangedmarker.
FE iterative critics exposed document-switch contamination, inactive ACK metadata loss, native read-await lostedit, then stale list overwriting sync cache. Fixed d112700/7e18cdc/042d312/95f1625 respectively with real React/TipTap and asynchronous native-storage regression checks. Chat comparison ABE29d8efc/FE6e61110 vsB found versionID lostonopen;777cf1e preserves exactimmutableversion and dirtylatestdraft. Fresh A777cf1e vsB6e61110 check8/8 vs2/8, gap:null within cache/version/draft scope.
Root visual800px found Sendoutsideviewport. b6d1909 wraps composer controls; actual720px DOM: Send375..418, composer scrollWidth=clientWidth438, tools406; screenshots recorded. FEb6d1909 immutable .artifacts/frontend-g:28tests+FSD+buildPASS;3 realAPI export/sync/operation testsPASS againstBE29d8efc. PDF/DOCX validation checks use actualgeneratedfiles; export lazychunk warning retained, notsilenced.
Nativeb6d1909 Universal app+DMG buildPASS; mountedread-onlyDMG contains exactsame app, Applications symlink, currentCSS/JS, lipo x86_64+arm64. DMG SHA256 ffc226fdaca48d53d3e9dadc63eb0c8f23b3376abcb0f1762bb8702e4284906e. This is an unsigned/localvalidation artifact, not notarizedrelease. FEPR1 andBEPR1 created todevelop; latestnativeCI/visualcoverage/plancompletioncritics pending beforemerge.

Root source-plan audit found company research permanentlyempty and REJECTEDnormalUI unreachable. API updatedfirst with persistent userprovidedCompanySource provenance; BE5f99281 implements exactsource research andAI snapshots,30PostgreSQL race tests/vet/build independentlyPASS. ExpandedactualHTTPsmoke8661cb5c-4bb5-4e7b-9b7d-850d6fa5b01f passes16checks including companysource persistence andterminalrejection.
Visual A f833bbb vsB b6d1909 freshcritic returnsgap:null withinreference shell/card/approval/composer scope. Root720px actually reviewed andapproved attachedversion; API finalizedandapproval consumed. Lighttheme720 labelsreadable, allbuttons named, Taboutline2px. Sourcecomparison excludes missing historicalAsidecaptures, andexplicitbrowser-onlyvisualresponses are notactualAI evidence.

Interview companysource audit caught cached→fresh sources deletion;8125709 binds pristine/dirtybaseline. Next critic found PATCHsuccess+GETfailure revertsdisplay;430c20a retainsACK but newcritic found page-remount fallbacktocachedoldrevision.3ae09e4 persistsACK viaexistingaccountcachequeue/localUpdate andremovespage-onlyoverlay. FreshA3ae09e4 vsB430c20a critic6/6vs4/6, gap:null withinACK/remount/realstoragequeue/nextSave+Prepare/accountscope. Root independently reproduced failureon430 andpassedunchanged actualAPI+browser GETabort+twoSave+page-remount checkon3ae; logs .artifacts/interview-ack-remount-{before,after}.log. Userenteredtestnotesonly, no realpersonaldata.
Root3ae09e4 immutable FE40tests+FSD+buildPASS. Liveplan test exposed equivalentRFC3339 fractionalformat .490Z vs.49Z;9aad318 normalizesboth tosameinstant andusesdeterministictrailing-zero regression, productcode unchanged. ExpandedHTTPsmoke12669053-1124-4828-a299-f5e0b5d76446 PASS16checks. BackendPR1 squash merged d53b69f; gitdiff5f99281..d53b69fempty, branchdeleted. FinalFEPR1/rootintegration andplatformCIpending; no additionalproductgapsremainincompletedboundedreviews.

Finalproduct9aad318 (test-only timestamp normalization over3ae09e4), BE d53b69f. Local40FE tests,4actualAPItests,30BEPostgreSQLrace,25Rust/clippy,16HTTPchecks pass. Actual browser failedGET+twoacknowledgedsaves+remount passeson3ae withsameprobe thatfailed430. Universal9aad318 app/DMG mountedandverifiedbotharchitectures+currentdist; DMGSHA256ef0321521d9ee3f442b1f1f902a949d83965121f0f6952fbcb24de57f778ac71. Evidence /tmp/push-final-universal-9aad318.json.
Freshprivate clone --recurse-submodules /tmp/push-final-clean.Uk5n0s atrootc1c2296 FE9aad318 BEd53b69f: npmci246packages/no sourceor node_modules symlinks,40FEtests+FSD+build,30BE-race+vet+build pass; allthreeworktreesclean. Fourlivechecks intentionallydisabledthere, independentlypassedbyrootwithactualAPI. RootPR1 created; exactfinalFEplatformCI+merge/pin/rootdevelopintegration pending.

Final FE CI run34110973822 at9aad318: macOS Universal DMG, Windows NSIS, export-tests PASS; both frontendverify andpolicy PASS. FE PR1 squashmerged5bec651f3bb205fadf0bcd3a6a785ad0f20e7e89; gitdiff9aad318..5bec651empty, featurebranchdeleted. BE remainsd53b69fdc88a2912ac208528bb635214e17056fb. Root pins these exactdevelopcommits. All observed movable gaps in completed scoped gauntlet comparisons areclosed; actualexternal rollout criteria remainpending peruserinstruction.
