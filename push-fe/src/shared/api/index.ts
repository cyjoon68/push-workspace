export { api } from './client'
export {
  ApiError,
  NETWORK_ERROR_MESSAGE,
  request,
  toApiError,
} from './envelope'
export type {
  ApiErrorBody,
  DataEnvelope,
  ListEnvelope,
  ListParams,
  PageInfo,
} from './envelope'
export {
  exchangeCode,
  fetchMe,
  logout,
  startOAuth,
} from './auth'
export type { AuthProvider, AuthUser, OAuthStart } from './auth'
export {
  createApplication,
  getApplication,
  listApplications,
  patchApplication,
} from './applications'
export type { Application, ApplicationStage } from './applications'
export {
  createJob,
  getJob,
  listJobAnalyses,
  listJobs,
} from './jobs'
export type { GapAnalysis, JobPosting } from './jobs'
export {
  createDocument,
  createDocumentExport,
  getDocument,
  getDocumentVersion,
  listDocuments,
  listDocumentVersions,
} from './documents'
export type {
  ClaimStatus,
  DocumentBlock,
  DocumentExport,
  DocumentKind,
  DocumentStatus,
  DocumentTemplate,
  DocumentVersion,
  EvidenceRef,
  PushDocument,
} from './documents'
export {
  createCareerEvidence,
  listCareerEvidence,
} from './career-evidence'
export type {
  CareerEvidence,
  EvidenceKind,
  VerificationStatus,
} from './career-evidence'
export {
  createConversation,
  listConversations,
  listMessages,
  sendMessage,
} from './conversations'
export type {
  AccessMode,
  AiOptions,
  Conversation,
  Message,
  MessageAttachment,
  MessageRole,
  Operation,
  OperationStatus,
} from './conversations'
export {
  decideApproval,
  getApproval,
  listApprovals,
} from './approvals'
export type {
  Approval,
  ApprovalKind,
  ApprovalStatus,
  ApprovalSummary,
} from './approvals'
export {
  createProjectRun,
  listProjectEvidence,
  listProjectRuns,
  listProjects,
} from './projects'
export type {
  BlueprintMetric,
  BlueprintTask,
  CliProvider,
  CliRun,
  CliRunState,
  ProjectBlueprint,
  ProjectEvidence,
} from './projects'
export { listCalendarEvents, syncGoogle } from './calendar'
export type { CalendarEvent } from './calendar'
export {
  createInterview,
  listInterviews,
  prepareInterview,
} from './interviews'
export type { InterviewSession } from './interviews'
export { fetchBilling, listAiModels } from './ai'
export type { AiModel, BillingSummary } from './ai'
export { getOperation, waitForOperation } from './operations'
export { uploadSource } from './sources'
export type { SourceFile } from './sources'
