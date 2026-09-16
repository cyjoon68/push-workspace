package presentation

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

type Handler struct {
	auth          *usecase.AuthService
	evidence      *usecase.EvidenceService
	jobs          *usecase.JobService
	applications  *usecase.ApplicationService
	documents     *usecase.DocumentService
	projects      *usecase.ProjectService
	interviews    *usecase.InterviewService
	calendar      *usecase.CalendarService
	conversations *usecase.ConversationService
	approvals     *usecase.ApprovalService
	operations    *usecase.OperationService
	ai            *usecase.AIService
	google        *usecase.GoogleService
	billing       *usecase.BillingService
	sources       domain.SourceStore
	aiKeys        domain.AiKeyStore
	cfg           ConfigView
	storageDir    string
	masterKey     string
}

type ConfigView struct {
	GoogleConfigured bool
	GmailBeta        bool
	ManagedAI        bool
	ByokEnabled      bool
	StripeConfigured bool
	JobSiteAdapters  map[string]bool
}

func sessionDTO(s *domain.Session) map[string]any {
	return map[string]any{
		"accessToken":  s.AccessToken,
		"refreshToken": s.RefreshToken,
		"expiresIn":    s.ExpiresIn,
		"user": map[string]any{
			"id":          s.User.ID,
			"displayName": s.User.DisplayName,
			"locale":      s.User.Locale,
		},
	}
}

func (h *Handler) oauthStart(c echo.Context) error {
	provider := c.Param("provider")
	callbackURL := c.Scheme() + "://" + c.Request().Host + "/api/v1/auth/" + provider + "/callback"
	url, state, expiresAt, err := h.auth.StartOAuth(c.Request().Context(), provider,
		c.QueryParam("codeChallenge"), c.QueryParam("codeChallengeMethod"), c.QueryParam("redirectUri"), callbackURL)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{
		"authorizationUrl": url, "state": state,
		"expiresAt": expiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) oauthCallback(c echo.Context) error {
	provider := c.Param("provider")
	link, err := h.auth.HandleCallback(c.Request().Context(), provider,
		c.QueryParam("code"), c.QueryParam("state"))
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, link)
}

func (h *Handler) authExchange(c echo.Context) error {
	var req exchangeReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	s, err := h.auth.Exchange(c.Request().Context(), req.Code, req.CodeVerifier)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, sessionDTO(s))
}

func (h *Handler) authRefresh(c echo.Context) error {
	var req refreshReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	s, err := h.auth.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, sessionDTO(s))
}

func (h *Handler) authLogout(c echo.Context) error {
	var req logoutReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if err := h.auth.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) authMe(c echo.Context) error {
	u, _ := c.Get("user").(*domain.User)
	return data(c, http.StatusOK, map[string]any{
		"id": u.ID, "displayName": u.DisplayName, "locale": u.Locale,
		"createdAt": u.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) healthz(c echo.Context) error {
	return data(c, http.StatusOK, map[string]any{"status": "ok"})
}
