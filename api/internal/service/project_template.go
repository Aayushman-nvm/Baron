package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectTemplateRepository defines persistence operations for project templates.
type ProjectTemplateRepository interface {
	Create(ctx context.Context, t *model.ProjectTemplate) error
	List(ctx context.Context) ([]model.ProjectTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ProjectTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectTemplateService handles project template business logic.
type ProjectTemplateService struct {
	templates     ProjectTemplateRepository
	projects      ProjectRepository
	members       ProjectMemberRepository
	typeWorkflows ProjectTypeWorkflowRepository
}

// NewProjectTemplateService creates a new ProjectTemplateService.
func NewProjectTemplateService(
	templates ProjectTemplateRepository,
	projects ProjectRepository,
	members ProjectMemberRepository,
	typeWorkflows ProjectTypeWorkflowRepository,
) *ProjectTemplateService {
	return &ProjectTemplateService{
		templates:     templates,
		projects:      projects,
		members:       members,
		typeWorkflows: typeWorkflows,
	}
}

// CreateFromProjectWithMappingsViaKey saves a project's current configuration as a
// template, snapshotting the per-type workflow mappings automatically.
func (s *ProjectTemplateService) CreateFromProjectWithMappingsViaKey(
	ctx context.Context,
	info *model.AuthInfo,
	projectKey, templateName string,
	templateDesc *string,
) (*model.ProjectTemplate, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	// Require owner or admin role on the source project, or global admin.
	if info.GlobalRole != model.RoleAdmin {
		member, err := s.members.GetByProjectAndUser(ctx, project.ID, info.UserID)
		if err != nil {
			return nil, model.ErrForbidden
		}
		if member.Role != model.ProjectRoleOwner && member.Role != model.ProjectRoleAdmin {
			return nil, model.ErrForbidden
		}
	}

	if templateName == "" {
		return nil, fmt.Errorf("template name is required: %w", model.ErrValidation)
	}

	// Snapshot the project's per-type workflow mappings.
	mappings, err := s.typeWorkflows.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("reading type workflows: %w", err)
	}
	typeWorkflows := make([]model.TemplateTypeWorkflow, len(mappings))
	for i, m := range mappings {
		typeWorkflows[i] = model.TemplateTypeWorkflow{
			WorkItemType: m.WorkItemType,
			WorkflowID:   m.WorkflowID,
		}
	}

	acv := project.AllowedComplexityValues
	if acv == nil {
		acv = []int{}
	}

	t := &model.ProjectTemplate{
		ID:                      uuid.New(),
		Name:                    templateName,
		Description:             templateDesc,
		CreatedBy:               info.UserID,
		DefaultWorkflowID:       project.DefaultWorkflowID,
		AllowedComplexityValues: acv,
		BusinessHours:           project.BusinessHours,
		TypeWorkflows:           typeWorkflows,
	}

	if err := s.templates.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("saving template: %w", err)
	}

	return s.templates.GetByID(ctx, t.ID)
}

// List returns all available project templates.
func (s *ProjectTemplateService) List(ctx context.Context) ([]model.ProjectTemplate, error) {
	return s.templates.List(ctx)
}

// GetByID returns a single template by ID.
func (s *ProjectTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.ProjectTemplate, error) {
	return s.templates.GetByID(ctx, id)
}

// Delete removes a template. Only the creator or a global admin can delete.
func (s *ProjectTemplateService) Delete(ctx context.Context, info *model.AuthInfo, id uuid.UUID) error {
	t, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if t.CreatedBy != info.UserID && info.GlobalRole != model.RoleAdmin {
		return model.ErrForbidden
	}

	return s.templates.Delete(ctx, id)
}
