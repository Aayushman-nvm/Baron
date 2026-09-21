package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// ProjectEventRepository handles project-wide event queries.
type ProjectEventRepository struct {
	db *sql.DB
}

// NewProjectEventRepository creates a new ProjectEventRepository.
func NewProjectEventRepository(db *sql.DB) *ProjectEventRepository {
	return &ProjectEventRepository{db: db}
}

// ProjectEventRow is a work item event enriched with the work item's display info.
type ProjectEventRow struct {
	model.WorkItemEventWithActor
	ItemNumber int    `json:"item_number"`
	ItemTitle  string `json:"item_title"`
	DisplayID  string `json:"display_id"`
}

// ListByProject returns the most recent events across all work items in a project,
// ordered by created_at DESC. Limit caps results (max 500).
func (r *ProjectEventRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]ProjectEventRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT e.id, e.work_item_id, e.actor_id, e.actor_type, e.event_type,
		       e.field_name, e.old_value, e.new_value, e.metadata, e.visibility, e.created_at,
		       CASE WHEN e.actor_type = 'system_key' THEN e.metadata->>'key_name'
		            ELSE u.display_name
		       END AS display_name,
		       wi.item_number, wi.title, wi.display_id
		FROM work_item_events e
		JOIN work_items wi ON wi.id = e.work_item_id
		LEFT JOIN users u ON e.actor_id = u.id AND e.actor_type = 'user'
		WHERE wi.project_id = $1
		  AND wi.deleted_at IS NULL
		ORDER BY e.created_at ASC
		LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying project events: %w", err)
	}
	defer rows.Close()

	var result []ProjectEventRow
	for rows.Next() {
		var row ProjectEventRow
		var (
			actorID     uuid.NullUUID
			fieldName   sql.NullString
			oldValue    sql.NullString
			newValue    sql.NullString
			metadataRaw []byte
			displayName sql.NullString
		)
		if err := rows.Scan(
			&row.ID, &row.WorkItemID, &actorID, &row.ActorType, &row.EventType,
			&fieldName, &oldValue, &newValue, &metadataRaw, &row.Visibility, &row.CreatedAt,
			&displayName,
			&row.ItemNumber, &row.ItemTitle, &row.DisplayID,
		); err != nil {
			return nil, fmt.Errorf("scanning project event: %w", err)
		}
		if actorID.Valid {
			row.ActorID = &actorID.UUID
		}
		if fieldName.Valid {
			row.FieldName = &fieldName.String
		}
		if oldValue.Valid {
			row.OldValue = &oldValue.String
		}
		if newValue.Valid {
			row.NewValue = &newValue.String
		}
		if displayName.Valid {
			row.ActorDisplayName = &displayName.String
		}
		row.Metadata = make(map[string]interface{})
		if len(metadataRaw) > 0 {
			json.Unmarshal(metadataRaw, &row.Metadata) //nolint:errcheck
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
