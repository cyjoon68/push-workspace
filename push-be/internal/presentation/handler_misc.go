package presentation

import (
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
	"push-be/internal/infra"
)

func (h *Handler) googleStatus(c echo.Context) error {
	st, err := h.google.Status(c.Request().Context(), userID(c))
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, st)
}

func (h *Handler) googleConnect(c echo.Context) error {
	var req googleConnectReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	callbackURL := c.Scheme() + "://" + c.Request().Host + "/api/v1/integrations/google/callback"
	url, state, expiresAt, err := h.google.Connect(c.Request().Context(), userID(c),
		req.CodeChallenge, req.CodeChallengeMethod, req.RedirectURI, callbackURL)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{
		"authorizationUrl": url, "state": state,
		"expiresAt": expiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) googleCallback(c echo.Context) error {
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	link, err := h.google.HandleCallback(c.Request().Context(),
		c.QueryParam("code"), c.QueryParam("state"))
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, link)
}

func (h *Handler) googleComplete(c echo.Context) error {
	var req googleCompleteReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	st, err := h.google.Complete(c.Request().Context(), userID(c), req.IntegrationCode, req.CodeVerifier)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, st)
}

func (h *Handler) googleDisconnect(c echo.Context) error {
	if err := h.google.Disconnect(c.Request().Context(), userID(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) googleSync(c echo.Context) error {
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	op, err := h.google.Sync(c.Request().Context(), userID(c))
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) googleMessages(c echo.Context) error {
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	if !h.cfg.GmailBeta {
		return domain.FeatureDisabled("gmail sync is not enabled")
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	var appID *uuid.UUID
	if raw := c.QueryParam("applicationId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		appID = &id
	}
	p, err := h.google.ListMessages(c.Request().Context(), userID(c), appID, page)
	if err != nil {
		return err
	}
	return list(c, pageBody(p))
}

func (h *Handler) googleEvents(c echo.Context) error {
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	var from, to *time.Time
	for _, p := range []struct {
		name string
		dst  **time.Time
	}{{"from", &from}, {"to", &to}} {
		if raw := c.QueryParam(p.name); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return domain.ValidationField(p.name, "must be RFC3339")
			}
			*p.dst = &t
		}
	}
	evs, err := h.google.ListEvents(c.Request().Context(), userID(c), from, to, page)
	if err != nil {
		return err
	}
	return list(c, pageBody(evs))
}

func (h *Handler) googleLinkMessage(c echo.Context) error {
	var req linkMessageReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if !h.cfg.GoogleConfigured {
		return domain.NotConfigured("google integration is not configured")
	}
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	m, err := h.google.LinkMessage(c.Request().Context(), userID(c), id, req.ApplicationID)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, m)
}

var jobSiteChecklist = []string{
	"Open the job posting in the built-in browser",
	"Review the extracted fields against the posting",
	"Upload finalized documents manually",
	"Confirm submission in the site UI",
	"Record the submission via POST /applications/:id/submission-drafts",
}

func (h *Handler) jobSites(c echo.Context) error {
	sites := []map[string]any{}
	for _, p := range []string{"WANTED", "JUMPIT", "JOBKOREA"} {
		sites = append(sites, map[string]any{
			"provider": p, "enabled": h.cfg.JobSiteAdapters[p],
			"permissionVerified": false, "mode": "MANUAL_CHECKLIST",
			"supportedFields": []string{}, "checklist": jobSiteChecklist,
		})
	}
	return data(c, http.StatusOK, sites)
}

func (h *Handler) aiModels(c echo.Context) error {
	provider := c.QueryParam("provider")
	credMode := c.QueryParam("credentialMode")
	if provider != "" && !domain.ValidAiProvider(provider) {
		return domain.ValidationField("provider", "unsupported provider")
	}
	if credMode != "" && !domain.ValidCredentialMode(credMode) {
		return domain.ValidationField("credentialMode", "unsupported credentialMode")
	}
	byokOpenAI := false
	if h.cfg.ByokEnabled && h.aiKeys != nil {
		_, err := h.aiKeys.Get(c.Request().Context(), userID(c), "OPENAI")
		byokOpenAI = err == nil
	}
	models := []map[string]any{}
	for _, m := range domain.OpenAIModels {
		if provider != "" && provider != m.Provider {
			continue
		}
		available := h.cfg.ManagedAI || byokOpenAI
		if credMode == "MANAGED" {
			available = h.cfg.ManagedAI
		} else if credMode == "BYOK" {
			available = byokOpenAI
		}
		models = append(models, map[string]any{
			"provider": m.Provider, "model": m.Model, "label": m.Label,
			"available": available, "supportedEfforts": []string{"LOW", "MEDIUM", "HIGH"},
		})
	}
	return data(c, http.StatusOK, models)
}

func (h *Handler) aiKeysList(c echo.Context) error {
	keys, err := h.aiKeys.List(c.Request().Context(), userID(c))
	if err != nil {
		return domain.Internal()
	}
	out := []map[string]any{}
	for _, k := range keys {
		out = append(out, map[string]any{
			"provider": k.Provider, "lastFour": k.LastFour, "configured": true,
			"updatedAt": k.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return data(c, http.StatusOK, out)
}

func (h *Handler) aiKeyPut(c echo.Context) error {
	provider := c.Param("provider")
	if !domain.ValidAiProvider(provider) {
		return domain.ValidationField("provider", "unsupported provider")
	}
	if !h.cfg.ByokEnabled {
		return domain.NotConfigured("BYOK encryption is not configured")
	}
	var req putAiKeyReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if len(req.Key) < 8 {
		return domain.ValidationField("key", "key is too short")
	}
	cipher, err := infra.NewKeyCipher(h.masterKey)
	if err != nil {
		return domain.NotConfigured("BYOK encryption is not configured")
	}
	ct, nonce, err := cipher.Encrypt(req.Key, userID(c), provider)
	if err != nil {
		return domain.Internal()
	}
	k := &domain.AiKey{
		UserID: userID(c), Provider: provider,
		LastFour: req.Key[len(req.Key)-4:], Ciphertext: ct, Nonce: nonce,
		UpdatedAt: time.Now().UTC(),
	}
	if err := h.aiKeys.Put(c.Request().Context(), k); err != nil {
		return domain.Internal()
	}
	return data(c, http.StatusOK, map[string]any{
		"provider": k.Provider, "lastFour": k.LastFour, "configured": true,
		"updatedAt": k.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) aiKeyDelete(c echo.Context) error {
	provider := c.Param("provider")
	if !domain.ValidAiProvider(provider) {
		return domain.ValidationField("provider", "unsupported provider")
	}
	if err := h.aiKeys.Delete(c.Request().Context(), userID(c), provider); err != nil {
		return domain.NotFound()
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) aiKeyTest(c echo.Context) error {
	provider := c.Param("provider")
	if !domain.ValidAiProvider(provider) {
		return domain.ValidationField("provider", "unsupported provider")
	}
	_, err := h.aiKeys.Get(c.Request().Context(), userID(c), provider)
	if err != nil {
		return data(c, http.StatusOK, map[string]any{
			"valid": false, "checkedAt": time.Now().UTC().Format(time.RFC3339),
			"errorCode": "KEY_MISSING",
		})
	}
	return data(c, http.StatusOK, map[string]any{
		"valid": false, "checkedAt": time.Now().UTC().Format(time.RFC3339),
		"errorCode": "PROVIDER_UNREACHABLE",
	})
}

func (h *Handler) aiGenerate(c echo.Context) error {
	var req aiGenerateReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.ai.Generate(c.Request().Context(), userID(c), usecase.AIGenerateInput{
		AI: req.AI, Prompt: req.Prompt,
		ApplicationID: req.ApplicationID, EvidenceIDs: req.EvidenceIDs,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) aiUsage(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.AiUsageFilter{Page: page}
	if v := c.QueryParam("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return domain.ValidationField("from", "must be RFC3339")
		}
		f.From = &t
	}
	if v := c.QueryParam("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return domain.ValidationField("to", "must be RFC3339")
		}
		f.To = &t
	}
	p, err := h.ai.ListUsage(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) billingStatus(c echo.Context) error {
	u, _ := c.Get("user").(*domain.User)
	st, err := h.billing.Status(c.Request().Context(), u)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, st)
}

func (h *Handler) billingCheckout(c echo.Context) error {
	var req checkoutReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	u, _ := c.Get("user").(*domain.User)
	sess, err := h.billing.Checkout(c.Request().Context(), u, req.PlanID)
	if err != nil {
		return err
	}
	out := map[string]any{"url": sess.URL, "expiresAt": nil}
	if sess.ExpiresAt != nil {
		out["expiresAt"] = sess.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return data(c, http.StatusOK, out)
}

func (h *Handler) billingPortal(c echo.Context) error {
	u, _ := c.Get("user").(*domain.User)
	url, err := h.billing.Portal(c.Request().Context(), u)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"url": url})
}

func (h *Handler) billingLedger(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.billing.Ledger(c.Request().Context(), userID(c), page)
	if err != nil {
		return err
	}
	return list(c, pageBody(p))
}

func (h *Handler) billingWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return domain.Validation("unreadable body")
	}
	if err := h.billing.HandleWebhook(c.Request().Context(), body, c.Request().Header.Get("Stripe-Signature")); err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"received": true})
}
