import assert from 'node:assert/strict';
import { randomUUID } from 'node:crypto';
import { setTimeout as delay } from 'node:timers/promises';

const base = new URL(`${(process.env.PUSH_API_URL ?? 'http://127.0.0.1:8080/api/v1').replace(/\/$/, '')}/`);
assert(['127.0.0.1', 'localhost', '[::1]'].includes(base.hostname), 'Smoke checks create data: use a local test API');
assert(process.env.DEV_AUTH_TOKEN, 'Set DEV_AUTH_TOKEN for the local test API');
const request = async (method, path, body, expected = 200, options = {}) => {
  const response = await fetch(new URL(path, base), {
    method,
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': options.key ?? randomUUID(),
      ...(options.anonymous ? {} : { Authorization: `Bearer ${process.env.DEV_AUTH_TOKEN}` }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(15000),
  });
  const result = response.status === 204 ? null : await response.json();
  assert.equal(response.status, expected, `${method} ${path}: ${result?.error?.code ?? response.status}`);
  return result?.data ?? result?.error ?? null;
};
const completed = async (operation) => {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    if (operation.status === 'SUCCEEDED') return operation.result.value;
    assert(!['FAILED', 'CANCELLED', 'NEEDS_INPUT'].includes(operation.status), `Operation ended: ${operation.error?.code ?? operation.status}`);
    await delay(100);
    operation = await request('GET', `operations/${operation.id}`);
  }
  assert.fail('Operation did not finish within 10 seconds');
};
const approve = async (kind, applicationId, targetId, decision = 'APPROVED') => {
  const approval = await request('POST', 'approvals', { kind, applicationId, targetId }, 201);
  return request('POST', `approvals/${approval.id}/decision`, { expectedRevision: approval.revision, decision });
};
const tag = randomUUID();
const sourceText = 'Go API 응답 시간을 20% 줄였습니다.';
const evidence = await request('POST', 'career-evidence', {
  kind: 'CAREER', title: `검증 ${tag}`, sourceText, skills: ['Go'],
}, 201);
const createApplication = async (company) => {
  const payload = {
    company, title: '서버 개발자', sourceText: 'Go PostgreSQL 개발자를 모집합니다.', sourceKind: 'TEXT',
    requirements: ['Go', 'PostgreSQL'], preferred: [],
  };
  const key = randomUUID();
  const job = await request('POST', 'jobs', payload, 201, { key });
  const repeated = await request('POST', 'jobs', payload, 201, { key });
  assert.equal(repeated.id, job.id);
  await request('POST', 'jobs', { ...payload, title: '변경된 본문' }, 409, { key });
  const application = await request('POST', 'applications', { jobId: job.id }, 201);
  return { job, application };
};
const a = await createApplication(`A-${tag}`);
const b = await createApplication(`B-${tag}`);
await approve('EVIDENCE_USE', a.application.id, evidence.id);
const analysis = await completed(await request('POST', `jobs/${a.job.id}/analyze`, {
  applicationId: a.application.id, expectedRevision: a.job.revision, evidenceIds: [evidence.id], ai: null,
}, 202));
assert(analysis.matched.some((entry) => entry.evidenceIds.includes(evidence.id)));
assert(analysis.missing.includes('PostgreSQL'));
const document = await request('POST', 'documents', {
  applicationId: a.application.id, title: '검증 이력서', kind: 'RESUME', template: 'CLASSIC',
}, 201);
const blockId = randomUUID();
const refs = [{ evidenceId: evidence.id, start: 0, end: Array.from(sourceText).length }];
const body = (text) => ({
  content: { type: 'doc', content: [{ type: 'paragraph', attrs: { blockId }, content: [{ type: 'text', text }] }] },
  blocks: [{ id: blockId, text, evidenceRefs: refs }],
});
const fabricated = await request('POST', `documents/${document.id}/versions`, {
  expectedRevision: document.revision, ...body('Go API 응답 시간을 90% 줄였습니다.'),
}, 201);
const fabricatedApproval = await approve('DOCUMENT_FINALIZE', a.application.id, fabricated.version.id);
await request('POST', `documents/${document.id}/finalize`, {
  expectedRevision: fabricated.document.revision, versionId: fabricated.version.id, approvalId: fabricatedApproval.id,
}, 409);
const generated = await completed(await request('POST', `documents/${document.id}/generate`, {
  expectedRevision: fabricated.document.revision, evidenceIds: [evidence.id], analysisId: analysis.id, ai: null,
}, 202));
const finalApproval = await approve('DOCUMENT_FINALIZE', a.application.id, generated.version.id);
const finalized = await request('POST', `documents/${document.id}/finalize`, {
  expectedRevision: generated.document.revision, versionId: generated.version.id, approvalId: finalApproval.id,
});
assert.equal(finalized.finalizedVersionId, generated.version.id);
await request('POST', `documents/${document.id}/versions`, { expectedRevision: 1, ...body(sourceText) }, 409);
assert.equal((await request('GET', `documents?applicationId=${b.application.id}`)).length, 0);
const preparing = await request('PATCH', `applications/${a.application.id}`, { expectedRevision: 1, stage: 'PREPARING' });
const ready = await request('PATCH', `applications/${a.application.id}`, { expectedRevision: preparing.revision, stage: 'READY' });
await request('PATCH', `applications/${a.application.id}`, { expectedRevision: ready.revision, stage: 'APPLIED' }, 409);
const submissionDraft = await request('POST', `applications/${a.application.id}/submission-drafts`, {
  expectedRevision: ready.revision, mode: 'MANUAL_RECORD', documentVersionIds: [generated.version.id], confirmedSubmitted: true,
}, 201);
const deniedSubmission = await approve('APPLICATION_SUBMIT', a.application.id, submissionDraft.id, 'DENIED');
await request('POST', `applications/${a.application.id}/submissions`, {
  expectedRevision: ready.revision, draftId: submissionDraft.id, approvalId: deniedSubmission.id,
}, 409);
const submitApproval = await approve('APPLICATION_SUBMIT', a.application.id, submissionDraft.id);
const submission = await request('POST', `applications/${a.application.id}/submissions`, {
  expectedRevision: ready.revision, draftId: submissionDraft.id, approvalId: submitApproval.id,
}, 201);
assert.equal(submission.status, 'SUCCEEDED');
const projects = await completed(await request('POST', 'projects/blueprints', {
  applicationId: a.application.id, gapAnalysisId: analysis.id, ai: null,
}, 202));
assert.equal(projects.length, 4);
await request('POST', `projects/${projects[0].id}/select`, { expectedRevision: projects[0].revision });
for (const provider of ['CODEX', 'CLAUDE_CODE', 'GROK_BUILD']) {
  const run = await request('POST', `projects/${projects[0].id}/runs`, {
    provider, workingDirectory: `/tmp/push-smoke-${tag}`, prompt: '검증용 프로젝트를 구현하세요.',
  }, 201);
  const denied = await approve('CLI_EXECUTE', a.application.id, run.id, 'DENIED');
  await request('POST', `projects/${projects[0].id}/runs/${run.id}/start`, {
    expectedRevision: run.revision, approvalId: denied.id, detectedVersion: 'test-only', deviceId: randomUUID(),
  }, 409);
}
const scheduledAt = new Date(Date.now() + 86400000).toISOString();
const canonicalTime = (value) => ({ ...value, accessedAt: new Date(value.accessedAt).toISOString() });
const companySource = { sourceUrl: 'https://company.example.invalid/about', sourceText: '검증 회사는 React와 TypeScript로 서비스를 개발합니다.', accessedAt: new Date().toISOString() };
const interview = await request('POST', 'interviews', {
  applicationId: a.application.id, title: `면접 ${tag}`, scheduledAt, durationMinutes: 60, evidenceIds: [evidence.id], companySources: [companySource],
}, 201);
const preparation = await completed(await request('POST', `interviews/${interview.id}/prepare`, {
  expectedRevision: interview.revision, ai: null,
}, 202));
assert(preparation.starAnswers.some((answer) => answer.evidenceIds.includes(evidence.id)));
assert.deepEqual(preparation.research.map(canonicalTime), [{claim: companySource.sourceText, sourceUrl: companySource.sourceUrl, accessedAt: companySource.accessedAt, verificationStatus: 'USER_PROVIDED'}]);
assert.deepEqual((await request('GET', `interviews/${interview.id}`)).companySources.map(canonicalTime), [companySource]);
assert.equal((await request('GET', `interviews?applicationId=${b.application.id}`)).length, 0);
const calendar = await request('GET', `calendar/events?from=${new Date().toISOString()}&to=${new Date(Date.now() + 172800000).toISOString()}&applicationId=${a.application.id}`);
assert(calendar.some((event) => event.id === interview.eventId));
const offer = await request('POST', 'offers', {
  applicationId: a.application.id, company: `A-${tag}`, annualSalaryMinor: 70000000, currency: 'KRW',
}, 201);
const foreignOffer = await request('POST', 'offers', {
  applicationId: b.application.id, company: `B-${tag}`, annualSalaryMinor: 9000000, currency: 'USD',
}, 201);
const comparison = await request('GET', `offers/compare?ids=${offer.id},${foreignOffer.id}`);
assert.equal(comparison.comparison.sameCurrency, false);
const routine = await request('POST', 'routines', {
  applicationId: a.application.id, title: `면접 준비 ${tag}`, kind: 'INTERVIEW_PREP', dueAt: scheduledAt,
}, 201);
await request('PATCH', `routines/${routine.id}`, { expectedRevision: routine.revision, status: 'DONE' }, 409);
const confirmed = await request('PATCH', `routines/${routine.id}`, { expectedRevision: routine.revision, status: 'CONFIRMED' });
assert.equal((await request('PATCH', `routines/${routine.id}`, { expectedRevision: confirmed.revision, status: 'DONE' })).status, 'DONE');
const rejected = await request('PATCH', `applications/${b.application.id}`, {expectedRevision: b.application.revision, stage: 'REJECTED'});
assert.equal(rejected.stage, 'REJECTED');
await request('PATCH', `applications/${b.application.id}`, {expectedRevision: rejected.revision, stage: 'PREPARING'}, 409);
await request('GET', 'jobs', undefined, 401, { anonymous: true });
await request('POST', 'integrations/google/sync', {}, 403);
console.log(JSON.stringify({ result: 'PASS', run: tag, checks: ['idempotency', 'evidence-lineage', 'fabrication-blocked', 'revision-conflict', 'two-job-isolation', 'submission-draft-approval', 'four-blueprints', 'three-cli-denials', 'interview-evidence', 'company-source-research', 'rejection-record', 'calendar-link', 'currency-comparison', 'routine-confirmation', 'auth-required', 'google-flag'] }));
