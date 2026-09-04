package model

import (
	"time"

	"github.com/google/uuid"
)

// ProjectTemplate is a saved snapshot of a project's configuration that can be
// applied when creating a new project.
type ProjectTemplate struct {
	ID                      uuid.UUID            `json:"id"`
	Name                    string               `json:"name"`
	Description             *string              `json:"description,omitempty"`
	CreatedBy               uuid.UUID            `json:"created_by"`
	DefaultWorkflowID       *uuid.UUID           `json:"default_workflow_id,omitempty"`
	AllowedComplexityValues []int                `json:"allowed_complexity_values"`
	BusinessHours           *BusinessHoursConfig `json:"business_hours,omitempty"`
	TypeWorkflows           []TemplateTypeWorkflow `json:"type_workflows,omitempty"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
}

// TemplateTypeWorkflow maps a work item type to a workflow within a template.
type TemplateTypeWorkflow struct {
	WorkItemType string    `json:"work_item_type"`
	WorkflowID   uuid.UUID `json:"workflow_id"`
}
