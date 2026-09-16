package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type ConversationService struct {
	conversations domain.ConversationStore
	applications  domain.ApplicationStore
	documents     domain.DocumentStore
	evidence      domain.EvidenceStore
	ops           domain.OperationStore
	uow           domain.UnitOfWork
	ai            *AIGate
	usage         domain.AiUsageStore
}

func NewConversationService(conversations domain.ConversationStore, applications domain.ApplicationStore, documents domain.DocumentStore, evidence domain.EvidenceStore, ops domain.OperationStore, uow domain.UnitOfWork, ai *AIGate, usage domain.AiUsageStore) *ConversationService {
	return &ConversationService{
		conversations: conversations, applications: applications, documents: documents,
		evidence: evidence, ops: ops, uow: uow, ai: ai, usage: usage,
	}
}

func (s *ConversationService) List(ctx context.Context, userID uuid.UUID, f domain.ConversationFilter) (domain.Page[domain.Conversation], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.conversations.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.Conversation]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(v domain.Conversation) domain.Cursor {
		return domain.Cursor{CreatedAt: v.CreatedAt, ID: v.ID}
	}), nil
}

func (s *ConversationService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Conversation, error) {
	v, err := s.conversations.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return v, nil
}

type CreateConversationInput struct {
	ApplicationID *uuid.UUID
	Title         string
}

func (s *ConversationService) Create(ctx context.Context, userID uuid.UUID, in CreateConversationInput) (*domain.Conversation, error) {
	if len(in.Title) > 300 {
		return nil, domain.ValidationField("title", "title must be at most 300 characters")
	}
	if in.ApplicationID != nil {
		if _, err := s.applications.Get(ctx, userID, *in.ApplicationID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NotFound()
			}
			return nil, domain.Internal()
		}
	}
	now := time.Now().UTC()
	c := &domain.Conversation{
		ID: uuid.New(), UserID: userID, Revision: 1,
		ApplicationID: in.ApplicationID, Title: in.Title,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.conversations.Create(ctx, c); err != nil {
		return nil, domain.Internal()
	}
	return c, nil
}

type PatchConversationInput struct {
	ExpectedRevision int64
	Title            *string
	Pinned           *bool
}

func (s *ConversationService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchConversationInput) (*domain.Conversation, error) {
	var out *domain.Conversation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		v, err := s.conversations.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if v.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(v.Revision)
		}
		if in.Title != nil {
			if len(*in.Title) > 300 {
				return domain.ValidationField("title", "title must be at most 300 characters")
			}
			v.Title = *in.Title
		}
		if in.Pinned != nil {
			v.Pinned = *in.Pinned
		}
		v.UpdatedAt = time.Now().UTC()
		if err := s.conversations.Update(ctx, v, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		v.Revision = in.ExpectedRevision + 1
		out = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ConversationService) Archive(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) (*domain.Conversation, error) {
	var out *domain.Conversation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		v, err := s.conversations.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if v.Revision != expectedRevision {
			return domain.RevisionConflict(v.Revision)
		}
		v.Archived = true
		v.UpdatedAt = time.Now().UTC()
		if err := s.conversations.Update(ctx, v, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		v.Revision = expectedRevision + 1
		out = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ConversationService) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, page domain.PageRequest) (domain.Page[domain.Message], error) {
	if _, err := s.Get(ctx, userID, conversationID); err != nil {
		return domain.Page[domain.Message]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.conversations.ListMessages(ctx, userID, conversationID, page, limit+1)
	if err != nil {
		return domain.Page[domain.Message]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(m domain.Message) domain.Cursor {
		return domain.Cursor{CreatedAt: m.CreatedAt, ID: m.ID}
	}), nil
}

type MessageContext struct {
	DocumentID  *uuid.UUID
	VersionID   *uuid.UUID
	EvidenceIDs []uuid.UUID
}

type PostMessageInput struct {
	Text       string
	Context    MessageContext
	AI         *domain.AiOptions
	AccessMode domain.AccessMode
}

func (s *ConversationService) resolveAttachment(ctx context.Context, userID uuid.UUID, conv *domain.Conversation, in MessageContext) (*domain.MessageAttachment, error) {
	if in.DocumentID == nil && in.VersionID == nil {
		return nil, nil
	}
	var doc *domain.Document
	var version *domain.DocumentVersion
	if in.DocumentID != nil {
		d, err := s.documents.Get(ctx, userID, *in.DocumentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ValidationField("context.documentId", "document not found")
			}
			return nil, domain.Internal()
		}
		doc = d
		if conv.ApplicationID != nil && doc.ApplicationID != *conv.ApplicationID {
			return nil, domain.ValidationField("context.documentId", "document belongs to another application")
		}
		if in.VersionID != nil {
			v, err := s.documents.GetVersion(ctx, userID, doc.ID, *in.VersionID)
			if err != nil {
				return nil, domain.ValidationField("context.versionId", "version not found in document")
			}
			version = v
		} else {
			if doc.LatestVersionID == nil {
				return nil, domain.ValidationField("context.documentId", "document has no versions")
			}
			v, err := s.documents.GetVersion(ctx, userID, doc.ID, *doc.LatestVersionID)
			if err != nil {
				return nil, domain.Internal()
			}
			version = v
		}
	} else {
		v, err := s.documents.GetVersionByID(ctx, userID, *in.VersionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ValidationField("context.versionId", "version not found")
			}
			return nil, domain.Internal()
		}
		version = v
		d, err := s.documents.Get(ctx, userID, version.DocumentID)
		if err != nil {
			return nil, domain.Internal()
		}
		doc = d
	}
	if conv.ApplicationID != nil && doc.ApplicationID != *conv.ApplicationID {
		return nil, domain.ValidationField("context.documentId", "document belongs to another application")
	}
	return &domain.MessageAttachment{
		Type: domain.AttachDocumentVersion, ID: version.ID,
		DocumentID: &doc.ID, Title: doc.Title,
	}, nil
}

func stubReply(text string) string {
	runes := []rune(text)
	if len(runes) > 80 {
		text = string(runes[:80])
	}
	return fmt.Sprintf("(로컬 스텁) 메시지를 받았습니다: %s", text)
}

func (s *ConversationService) PostMessage(ctx context.Context, userID, conversationID uuid.UUID, in PostMessageInput) (*domain.Operation, error) {
	if in.Text == "" || domain.CodePointLen(in.Text) > 20000 {
		return nil, domain.ValidationField("text", "text must be 1-20000 characters")
	}
	if !domain.ValidAccessMode(in.AccessMode) {
		return nil, domain.ValidationField("accessMode", "must be SUGGEST or CONFIRM_ACTIONS")
	}
	var completion *domain.AICompletion
	if in.AI != nil {
		c, err := s.ai.Complete(ctx, userID, in.AI, "", in.Text)
		if err != nil {
			return nil, err
		}
		completion = c
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		conv, err := s.conversations.Get(ctx, userID, conversationID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if conv.Archived {
			return domain.InvalidTransition("conversation is archived")
		}
		attachments := []domain.MessageAttachment{}
		docAttach, err := s.resolveAttachment(ctx, userID, conv, in.Context)
		if err != nil {
			return err
		}
		if docAttach != nil {
			attachments = append(attachments, *docAttach)
		}
		for _, eid := range in.Context.EvidenceIDs {
			e, err := s.evidence.Get(ctx, userID, eid)
			if err != nil {
				return domain.ValidationField("context.evidenceIds", "evidence "+eid.String()+" not found")
			}
			attachments = append(attachments, domain.MessageAttachment{
				Type: domain.AttachEvidence, ID: e.ID, Title: e.Title,
			})
		}
		now := time.Now().UTC()
		opID := uuid.New()
		userMsg := &domain.Message{
			ID: uuid.New(), UserID: userID, ConversationID: conv.ID,
			Role: domain.RoleUser, Text: in.Text, Attachments: attachments,
			OperationID: &opID, CreatedAt: now,
		}
		reply := stubReply(in.Text)
		if completion != nil {
			reply = completion.Text
		}
		assistantMsg := &domain.Message{
			ID: uuid.New(), UserID: userID, ConversationID: conv.ID,
			Role: domain.RoleAssistant, Text: reply,
			Attachments: []domain.MessageAttachment{},
			OperationID: &opID, CreatedAt: now,
		}
		if err := s.conversations.CreateMessage(ctx, userMsg); err != nil {
			return err
		}
		if err := s.conversations.CreateMessage(ctx, assistantMsg); err != nil {
			return err
		}
		res, _ := json.Marshal(map[string]any{
			"userMessage": userMsg, "assistantMessage": assistantMsg,
			"approvalIds": []string{},
		})
		op = &domain.Operation{
			ID: opID, UserID: userID, Type: domain.OpChatMessage,
			ApplicationID: conv.ApplicationID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpChatMessage), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.ops.Create(ctx, op); err != nil {
			return err
		}
		if completion != nil && s.usage != nil {
			return RecordUsage(ctx, s.usage, userID, opID, in.AI, completion)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}
