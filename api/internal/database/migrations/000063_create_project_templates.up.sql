-- Project templates: save a snapshot of an existing project's configuration
-- so it can be reused when creating new projects.
-- A template captures: name, description, default_workflow_id,
-- allowed_complexity_values, business_hours, and per-type workflow mappings.

CREATE TABLE project_templates (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                        TEXT NOT NULL,
    description                 TEXT,
    created_by                  UUID NOT NULL REFERENCES users(id),
    default_workflow_id         UUID REFERENCES workflows(id) ON DELETE SET NULL,
    allowed_complexity_values   INTEGER[] NOT NULL DEFAULT '{}',
    business_hours              JSONB,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-type workflow mappings stored in the template
CREATE TABLE project_template_type_workflows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id     UUID NOT NULL REFERENCES project_templates(id) ON DELETE CASCADE,
    work_item_type  TEXT NOT NULL,
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    UNIQUE (template_id, work_item_type)
);

CREATE INDEX idx_project_templates_created_by ON project_templates(created_by);
