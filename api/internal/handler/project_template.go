package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
	"github.com/marcoshack/taskwondo/internal/service"
)

// ProjectTemplateHandler handles project template endpoints.
type ProjectTemplateHandler struct {
	templates *service.ProjectTemplateService
}

// NewProjectTemplateHandler creates a new ProjectTemplateHandler.
func NewProjectTemplateHandler(templates *service.ProjectTemplateService) *ProjectTemplateHandler {
	return &ProjectTemplateHandler{templates: templates}
}

// --- Request DTOs ---

type createTemplateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// --- Response DTOs ---

type templateResponse struct {
	ID                      uuid.UUID                    `json:"id"`
	Name                    string                       `json:"name"`
	Description             *string                      `json:"description,omitempty"`
	CreatedBy               uuid.UUID                    `json:"created_by"`
	DefaultWorkflowID       *uuid.UUID                   `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues []int                        `json:"allowed_complexity_values"`
	BusinessHours           *model.BusinessHoursConfig   `json:"business_hours,omitempty"`
	TypeWorkflows           []model.TemplateTypeWorkflow `json:"type_workflows"`
}

func toTemplateResponse(t *model.ProjectTemplate) templateResponse {
	acv := t.AllowedComplexityValues
	if acv == nil {
		acv = []int{}
	}
	tws := t.TypeWorkflows
	if tws == nil {
		tws = []model.TemplateTypeWorkflow{}
	}
	return templateResponse{
		ID:                      t.ID,
		Name:                    t.Name,
		Description:             t.Description,
		CreatedBy:               t.CreatedBy,
		DefaultWorkflowID:       t.DefaultWorkflowID,
		AllowedComplexityValues: acv,
		BusinessHours:           t.BusinessHours,
		TypeWorkflows:           tws,
	}
}

// --- Handlers ---

// List handles GET /api/v1/project-templates
func (h *ProjectTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templates.List(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to list project templates")
		writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	resp := make([]templateResponse, len(templates))
	for i := range templates {
		resp[i] = toTemplateResponse(&templates[i])
	}
	writeData(w, http.StatusOK, resp)
}

// CreateFromProject handles POST /api/v1/{namespace}/projects/{projectKey}/save-as-template
func (h *ProjectTemplateHandler) CreateFromProject(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	projectKey := chi.URLParam(r, "projectKey")

	var req createTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, CodeValidationError, "name is required")
		return
	}

	t, err := h.templates.CreateFromProjectWithMappingsViaKey(r.Context(), info, projectKey, req.Name, req.Description)
	if err != nil {
		handleTemplateError(w, r, err, "failed to create project template")
		return
	}

	writeData(w, http.StatusCreated, toTemplateResponse(t))
}

// Delete handles DELETE /api/v1/project-templates/{templateId}
func (h *ProjectTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	info := model.AuthInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "not authenticated")
		return
	}

	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template ID")
	if !ok {
		return
	}

	if err := h.templates.Delete(r.Context(), info, templateID); err != nil {
		handleTemplateError(w, r, err, "failed to delete project template")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleTemplateError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, CodeNotFound, "template not found")
		return
	}
	if errors.Is(err, model.ErrForbidden) {
		writeError(w, http.StatusForbidden, CodeForbidden, "insufficient permissions")
		return
	}
	if errors.Is(err, model.ErrValidation) {
		writeErrorFromService(w, http.StatusBadRequest, CodeValidationError, err)
		return
	}
	log.Ctx(r.Context()).Error().Err(err).Msg(logMsg)
	writeError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
}
