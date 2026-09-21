package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/repository"
	"github.com/marcoshack/taskwondo/internal/service"
)

// ProjectEventsHandler serves project-wide activity log endpoints.
type ProjectEventsHandler struct {
	repo     *repository.ProjectEventRepository
	projects *service.ProjectService
}

// NewProjectEventsHandler creates a new ProjectEventsHandler.
func NewProjectEventsHandler(repo *repository.ProjectEventRepository, projects *service.ProjectService) *ProjectEventsHandler {
	return &ProjectEventsHandler{repo: repo, projects: projects}
}

// projectEventResponse is the wire format for a single project-scoped event.
type projectEventResponse struct {
	ID          uuid.UUID              `json:"id"`
	WorkItemID  uuid.UUID              `json:"work_item_id"`
	ItemNumber  int                    `json:"item_number"`
	ItemTitle   string                 `json:"item_title"`
	DisplayID   string                 `json:"display_id"`
	EventType   string                 `json:"event_type"`
	Actor       *eventActorResponse    `json:"actor,omitempty"`
	FieldName   *string                `json:"field_name,omitempty"`
	OldValue    *string                `json:"old_value,omitempty"`
	NewValue    *string                `json:"new_value,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Visibility  string                 `json:"visibility"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ListProjectEvents handles GET /api/v1/{namespace}/projects/{projectKey}/events
func (h *ProjectEventsHandler) ListProjectEvents(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	project, err := h.projects.Get(r.Context(), info, projectKey)
	if err != nil {
		handleProjectError(w, r, err, "failed to get project")
		return
	}

	limit := 200
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	rows, err := h.repo.ListByProject(r.Context(), project.ID, limit)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list project events")
		writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	resp := make([]projectEventResponse, len(rows))
	for i, row := range rows {
		e := projectEventResponse{
			ID:         row.ID,
			WorkItemID: row.WorkItemID,
			ItemNumber: row.ItemNumber,
			ItemTitle:  row.ItemTitle,
			DisplayID:  row.DisplayID,
			EventType:  row.EventType,
			FieldName:  row.FieldName,
			OldValue:   row.OldValue,
			NewValue:   row.NewValue,
			Metadata:   row.Metadata,
			Visibility: row.Visibility,
			CreatedAt:  row.CreatedAt,
		}
		if e.Metadata == nil {
			e.Metadata = map[string]interface{}{}
		}
		if row.ActorID != nil {
			actor := &eventActorResponse{ID: *row.ActorID}
			if row.ActorDisplayName != nil {
				actor.DisplayName = *row.ActorDisplayName
			}
			e.Actor = actor
		}
		resp[i] = e
	}

	writeData(w, http.StatusOK, resp)
}
