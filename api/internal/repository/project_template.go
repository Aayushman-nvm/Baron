package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectTemplateRepository handles project template persistence.
type ProjectTemplateRepository struct {
	db *sql.DB
}

// NewProjectTemplateRepository creates a new ProjectTemplateRepository.
func NewProjectTemplateRepository(db *sql.DB) *ProjectTemplateRepository {
	return &ProjectTemplateRepository{db: db}
}

// Create inserts a new project template and its type-workflow mappings.
func (r *ProjectTemplateRepository) Create(ctx context.Context, t *model.ProjectTemplate) error {
	var bhJSON []byte
	if t.BusinessHours != nil {
		var err error
		bhJSON, err = json.Marshal(t.BusinessHours)
		if err != nil {
			return fmt.Errorf("marshalling business_hours: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO project_templates
		 (id, name, description, created_by, default_workflow_id,
		  allowed_complexity_values, business_hours, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,now(),now())`,
		t.ID, t.Name, t.Description, t.CreatedBy, t.DefaultWorkflowID,
		pq.Array(t.AllowedComplexityValues), nullableJSON(bhJSON),
	)
	if err != nil {
		return fmt.Errorf("inserting project template: %w", err)
	}

	for _, tw := range t.TypeWorkflows {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO project_template_type_workflows
			 (id, template_id, work_item_type, workflow_id)
			 VALUES ($1,$2,$3,$4)`,
			uuid.New(), t.ID, tw.WorkItemType, tw.WorkflowID,
		)
		if err != nil {
			return fmt.Errorf("inserting template type workflow: %w", err)
		}
	}

	return nil
}

// List returns all project templates.
func (r *ProjectTemplateRepository) List(ctx context.Context) ([]model.ProjectTemplate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, created_by, default_workflow_id,
		        allowed_complexity_values, business_hours, created_at, updated_at
		 FROM project_templates
		 ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing project templates: %w", err)
	}
	defer rows.Close()

	var templates []model.ProjectTemplate
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating project templates: %w", err)
	}

	// Load type-workflow mappings for each template
	for i := range templates {
		tws, err := r.listTypeWorkflows(ctx, templates[i].ID)
		if err != nil {
			return nil, err
		}
		templates[i].TypeWorkflows = tws
	}

	return templates, nil
}

// GetByID returns a single project template by ID.
func (r *ProjectTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ProjectTemplate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_by, default_workflow_id,
		        allowed_complexity_values, business_hours, created_at, updated_at
		 FROM project_templates
		 WHERE id = $1`, id)

	t, err := scanTemplateRow(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting project template: %w", err)
	}

	tws, err := r.listTypeWorkflows(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.TypeWorkflows = tws

	return t, nil
}

// Delete removes a project template by ID.
func (r *ProjectTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM project_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting project template: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// --- helpers ---

func (r *ProjectTemplateRepository) listTypeWorkflows(ctx context.Context, templateID uuid.UUID) ([]model.TemplateTypeWorkflow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT work_item_type, workflow_id
		 FROM project_template_type_workflows
		 WHERE template_id = $1
		 ORDER BY work_item_type`, templateID)
	if err != nil {
		return nil, fmt.Errorf("listing template type workflows: %w", err)
	}
	defer rows.Close()

	var tws []model.TemplateTypeWorkflow
	for rows.Next() {
		var tw model.TemplateTypeWorkflow
		if err := rows.Scan(&tw.WorkItemType, &tw.WorkflowID); err != nil {
			return nil, fmt.Errorf("scanning template type workflow: %w", err)
		}
		tws = append(tws, tw)
	}
	return tws, rows.Err()
}

type templateScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(s templateScanner) (*model.ProjectTemplate, error) {
	var t model.ProjectTemplate
	var acv pq.GenericArray
	acv.A = &[]int{}
	var bhRaw []byte
	if err := s.Scan(
		&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.DefaultWorkflowID,
		&acv, &bhRaw, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scanning project template: %w", err)
	}
	if vals, ok := acv.A.(*[]int); ok {
		t.AllowedComplexityValues = *vals
	}
	if t.AllowedComplexityValues == nil {
		t.AllowedComplexityValues = []int{}
	}
	if len(bhRaw) > 0 {
		var bh model.BusinessHoursConfig
		if err := json.Unmarshal(bhRaw, &bh); err == nil {
			t.BusinessHours = &bh
		}
	}
	return &t, nil
}

func scanTemplateRow(row *sql.Row) (*model.ProjectTemplate, error) {
	var t model.ProjectTemplate
	var acv pq.GenericArray
	acv.A = &[]int{}
	var bhRaw []byte
	if err := row.Scan(
		&t.ID, &t.Name, &t.Description, &t.CreatedBy, &t.DefaultWorkflowID,
		&acv, &bhRaw, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if vals, ok := acv.A.(*[]int); ok {
		t.AllowedComplexityValues = *vals
	}
	if t.AllowedComplexityValues == nil {
		t.AllowedComplexityValues = []int{}
	}
	if len(bhRaw) > 0 {
		var bh model.BusinessHoursConfig
		if err := json.Unmarshal(bhRaw, &bh); err == nil {
			t.BusinessHours = &bh
		}
	}
	return &t, nil
}

// nullableJSON returns nil when b is empty, otherwise the raw bytes.
func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
