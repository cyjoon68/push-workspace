package presentation

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

type Deps struct {
	Auth          *usecase.AuthService
	Evidence      *usecase.EvidenceService
	Jobs          *usecase.JobService
	Applications  *usecase.ApplicationService
	Documents     *usecase.DocumentService
	Projects      *usecase.ProjectService
	Interviews    *usecase.InterviewService
	Calendar      *usecase.CalendarService
	Conversations *usecase.ConversationService
	Approvals     *usecase.ApprovalService
	Operations    *usecase.OperationService
	AI            *usecase.AIService
	Google        *usecase.GoogleService
	Billing       *usecase.BillingService
	Sources       domain.SourceStore
	AiKeys        domain.AiKeyStore
	Idempotency   domain.IdempotencyStore
	Config        ConfigView
	StorageDir    string
	MasterKey     string
}

func NewServer(d Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = ErrorHandler

	e.Use(requestIDMiddleware())
	e.Use(bodyLimitMiddleware(1 << 20))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:5173", "http://127.0.0.1:5173",
			"tauri://localhost", "http://tauri.localhost", "https://tauri.localhost",
		},
		AllowHeaders: []string{
			echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept,
			echo.HeaderAuthorization, "Idempotency-Key", "X-Request-Id",
		},
		AllowMethods: []string{
			echo.GET, echo.POST, echo.PATCH, echo.PUT, echo.DELETE, echo.OPTIONS,
		},
	}))

	h := &Handler{
		auth: d.Auth, evidence: d.Evidence, jobs: d.Jobs, applications: d.Applications,
		documents: d.Documents, projects: d.Projects, interviews: d.Interviews,
		approvals: d.Approvals, operations: d.Operations, calendar: d.Calendar,
		conversations: d.Conversations, sources: d.Sources,
		aiKeys: d.AiKeys, ai: d.AI, google: d.Google, billing: d.Billing,
		cfg: d.Config, storageDir: d.StorageDir, masterKey: d.MasterKey,
	}

	e.GET("/healthz", h.healthz)
	e.GET("/api/v1/healthz", h.healthz)

	pub := e.Group("/api/v1")
	pub.GET("/auth/:provider/start", h.oauthStart)
	pub.GET("/auth/:provider/callback", h.oauthCallback)
	pub.POST("/auth/exchange", h.authExchange)
	pub.POST("/auth/refresh", h.authRefresh)
	pub.POST("/auth/logout", h.authLogout)
	pub.GET("/integrations/google/callback", h.googleCallback)
	pub.POST("/billing/webhook", h.billingWebhook)

	g := e.Group("/api/v1", authMiddleware(d.Auth), NewIdempotencyMiddleware(d.Idempotency).Handle)

	g.GET("/auth/me", h.authMe)

	g.GET("/career-evidence", h.listEvidence)
	g.POST("/career-evidence", h.createEvidence)
	g.GET("/career-evidence/:id", h.getEvidence)
	g.POST("/career-evidence/import", h.importEvidence)
	g.POST("/career-evidence/:id/archive", h.archiveEvidence)

	g.POST("/sources", h.createSource)
	g.GET("/sources/:id", h.getSource)
	g.GET("/sources/:id/content", h.getSourceContent)
	g.DELETE("/sources/:id", h.deleteSource)

	g.GET("/jobs", h.listJobs)
	g.POST("/jobs", h.createJob)
	g.GET("/jobs/:id", h.getJob)
	g.PATCH("/jobs/:id", h.patchJob)
	g.POST("/jobs/:id/analyze", h.analyzeJob)
	g.GET("/jobs/:id/analyses", h.listAnalyses)

	g.GET("/applications", h.listApplications)
	g.POST("/applications", h.createApplication)
	g.POST("/applications/import", h.importApplication)
	g.GET("/applications/:id", h.getApplication)
	g.PATCH("/applications/:id", h.patchApplication)
	g.GET("/applications/:id/timeline", h.applicationTimeline)
	g.GET("/applications/:id/checklist", h.applicationChecklist)
	g.POST("/applications/:id/submission-drafts", h.createSubmissionDraft)
	g.POST("/applications/:id/submissions", h.submitApplication)
	g.GET("/applications/:id/submissions", h.listSubmissions)

	g.GET("/documents", h.listDocuments)
	g.POST("/documents", h.createDocument)
	g.GET("/documents/:id", h.getDocument)
	g.PATCH("/documents/:id", h.patchDocument)
	g.GET("/documents/:id/versions", h.listVersions)
	g.GET("/documents/:id/versions/:versionId", h.getVersion)
	g.POST("/documents/:id/versions", h.createVersion)
	g.POST("/documents/:id/generate", h.generateDocument)
	g.POST("/documents/:id/revisions", h.createRevision)
	g.POST("/documents/:id/revisions/:revisionId/apply", h.applyRevision)
	g.POST("/documents/:id/review", h.reviewDocument)
	g.POST("/documents/:id/finalize", h.finalizeDocument)
	g.POST("/documents/:id/archive", h.archiveDocument)
	g.POST("/documents/:id/exports", h.createExport)
	g.POST("/documents/:id/exports/:exportId/result", h.recordExportResult)
	g.GET("/documents/:id/exports", h.listExports)

	g.GET("/projects", h.listProjects)
	g.POST("/projects/blueprints", h.generateBlueprints)
	g.GET("/projects/:id", h.getProject)
	g.POST("/projects/:id/select", h.selectProject)
	g.GET("/projects/:id/runs", h.listRuns)
	g.POST("/projects/:id/runs", h.createRun)
	g.POST("/projects/:id/runs/:runId/start", h.startRun)
	g.POST("/projects/:id/runs/:runId/launch", h.launchRun)
	g.POST("/projects/:id/runs/:runId/recover", h.recoverRun)
	g.POST("/projects/:id/runs/:runId/result", h.resultRun)
	g.POST("/projects/:id/evidence", h.createProjectEvidence)
	g.POST("/projects/:id/evidence/:evidenceId/verify", h.verifyProjectEvidence)
	g.GET("/projects/:id/evidence", h.listProjectEvidence)

	g.GET("/interviews", h.listInterviews)
	g.POST("/interviews", h.createInterview)
	g.GET("/interviews/:id", h.getInterview)
	g.PATCH("/interviews/:id", h.patchInterview)
	g.POST("/interviews/:id/prepare", h.prepareInterview)

	g.GET("/conversations", h.listConversations)
	g.POST("/conversations", h.createConversation)
	g.PATCH("/conversations/:id", h.patchConversation)
	g.GET("/conversations/:id/messages", h.listMessages)
	g.POST("/conversations/:id/messages", h.postMessage)
	g.POST("/conversations/:id/archive", h.archiveConversation)

	g.GET("/calendar/events", h.listCalendarEvents)
	g.POST("/calendar/events", h.createCalendarEvent)
	g.PATCH("/calendar/events/:id", h.patchCalendarEvent)
	g.DELETE("/calendar/events/:id", h.deleteCalendarEvent)

	g.GET("/approvals", h.listApprovals)
	g.POST("/approvals", h.createApproval)
	g.GET("/approvals/:id", h.getApproval)
	g.POST("/approvals/:id/decision", h.decideApproval)

	g.GET("/operations/:id", h.getOperation)
	g.POST("/operations/:id/input", h.operationInput)
	g.POST("/operations/:id/cancel", h.cancelOperation)
	g.GET("/operations/:id/events", h.operationEvents)

	g.GET("/integrations/google", h.googleStatus)
	g.POST("/integrations/google/connect", h.googleConnect)
	g.POST("/integrations/google/complete", h.googleComplete)
	g.DELETE("/integrations/google", h.googleDisconnect)
	g.POST("/integrations/google/sync", h.googleSync)
	g.GET("/integrations/google/messages", h.googleMessages)
	g.GET("/integrations/google/events", h.googleEvents)
	g.POST("/integrations/google/messages/:id/link", h.googleLinkMessage)
	g.GET("/integrations/job-sites", h.jobSites)

	g.GET("/ai/models", h.aiModels)
	g.GET("/ai/keys", h.aiKeysList)
	g.PUT("/ai/keys/:provider", h.aiKeyPut)
	g.DELETE("/ai/keys/:provider", h.aiKeyDelete)
	g.POST("/ai/keys/:provider/test", h.aiKeyTest)
	g.POST("/ai/generate", h.aiGenerate)
	g.GET("/ai/usage", h.aiUsage)

	g.GET("/billing", h.billingStatus)
	g.POST("/billing/checkout", h.billingCheckout)
	g.POST("/billing/portal", h.billingPortal)
	g.GET("/billing/ledger", h.billingLedger)

	return e
}
