package repositories

import (
	"juki-engine/pkg/data/models"
)

const maxLogsPerProject = 1000

// CreateConsoleLog inserts a log entry and enforces the rolling 1000-entry limit.
func (r *repository) CreateConsoleLog(entry *models.ConsoleLog) error {
	if err := r.store.Create(entry).Error; err != nil {
		return err
	}

	// Rolling retention: delete oldest entries beyond the limit
	var count int64
	r.store.Model(&models.ConsoleLog{}).Where("project_id = ?", entry.ProjectID).Count(&count)
	if count > maxLogsPerProject {
		excess := count - maxLogsPerProject
		r.store.Exec(`
			DELETE FROM console_logs
			WHERE id IN (
				SELECT id FROM console_logs
				WHERE project_id = ?
				ORDER BY created_at ASC
				LIMIT ?
			)`, entry.ProjectID, excess)
	}

	return nil
}

// GetConsoleLogs returns logs filtered by project + optional session + optional level.
func (r *repository) GetConsoleLogs(projectID, sessionID string, limit, offset int, level string) ([]*models.ConsoleLog, int64, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	q := r.store.Model(&models.ConsoleLog{}).Where("project_id = ?", projectID)
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	if level != "" && level != "unspecified" {
		q = q.Where("level = ?", level)
	}

	var total int64
	q.Count(&total)

	var entries []*models.ConsoleLog
	err := q.Order("timestamp_ms DESC").Limit(limit).Offset(offset).Find(&entries).Error
	return entries, total, err
}

// ClearConsoleLogs deletes all logs for a project (and optionally a session).
func (r *repository) ClearConsoleLogs(projectID, sessionID string) (int64, error) {
	q := r.store.Where("project_id = ?", projectID)
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	result := q.Delete(&models.ConsoleLog{})
	return result.RowsAffected, result.Error
}
