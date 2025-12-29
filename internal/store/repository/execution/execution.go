package execution

import (
	"context"
	"time"

	"github.com/bornholm/oplet/internal/crypto"
	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// CRUD Operations

func (r *Repository) Create(ctx context.Context, execution *store.TaskExecution) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		token, err := crypto.RandomToken(tokenSize)
		if err != nil {
			return errors.WithStack(err)
		}

		execution.RunnerToken = token

		if err := db.Create(execution).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*store.TaskExecution, error) {
	var execution store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload("Task").Preload("User").Preload("Logs").Preload("OutputFiles").First(&execution, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// GetByIDForUser retrieves an execution by ID, ensuring it belongs to the specified user
func (r *Repository) GetByIDForUser(ctx context.Context, id uint, userID uint) (*store.TaskExecution, error) {
	var execution store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload("Task").Preload("User").Preload("Logs").Preload("OutputFiles").
			Where("id = ? AND user_id = ?", id, userID).First(&execution).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *Repository) Update(ctx context.Context, execution *store.TaskExecution) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Save(execution).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Delete(&store.TaskExecution{}, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Query Operations

func (r *Repository) GetByTaskID(ctx context.Context, taskID uint, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("task_id = ?", taskID).Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// GetByTaskIDForUser retrieves executions for a specific task and user
func (r *Repository) GetByTaskIDForUser(ctx context.Context, taskID uint, userID uint, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("user_id = ?", userID).Preload("Task").Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *Repository) GetByContainerID(ctx context.Context, containerID string) (*store.TaskExecution, error) {
	var execution store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("container_id = ?", containerID).Preload("Task").Preload("User").First(&execution).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *Repository) GetRunningExecutions(ctx context.Context) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	runningStatuses := []store.TaskExecutionStatus{
		store.StatusPending,
		store.StatusPullingImage,
		store.StatusImagePulled,
		store.StatusCreatingContainer,
		store.StatusContainerCreated,
		store.StatusUploadingFiles,
		store.StatusFilesUploaded,
		store.StatusStartingContainer,
		store.StatusContainerStarted,
		store.StatusRunning,
		store.StatusDownloadingFiles,
	}

	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("status IN ?", runningStatuses).Preload("Task").Preload("User").Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Preload("Task").Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// ListForUser retrieves executions for a specific user
func (r *Repository) ListForUser(ctx context.Context, userID uint, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("user_id = ?", userID).Preload("Task").Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// Log Operations

func (r *Repository) AddLog(ctx context.Context, executionID uint, log *store.TaskExecutionLog) error {
	log.ExecutionID = executionID
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(log).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) GetLogs(ctx context.Context, executionID uint, limit, offset int) ([]*store.TaskExecutionLog, error) {
	var logs []*store.TaskExecutionLog
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("execution_id = ?", executionID).Order("timestamp ASC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&logs).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *Repository) GetLogsSince(ctx context.Context, executionID uint, since time.Time) ([]*store.TaskExecutionLog, error) {
	var logs []*store.TaskExecutionLog
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("execution_id = ? AND timestamp > ?", executionID, since).Order("timestamp ASC").Find(&logs).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// File Operations

func (r *Repository) AddFile(ctx context.Context, executionID uint, file *store.TaskExecutionFile) error {
	file.ExecutionID = executionID
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(file).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) GetFiles(ctx context.Context, executionID uint, isOutput bool) ([]*store.TaskExecutionFile, error) {
	var files []*store.TaskExecutionFile
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("execution_id = ? AND is_output = ?", executionID, isOutput).Order("filename ASC").Find(&files).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (r *Repository) GetFileByPath(ctx context.Context, executionID uint, filename string) (*store.TaskExecutionFile, error) {
	var file store.TaskExecutionFile
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("execution_id = ? AND filename = ?", executionID, filename).First(&file).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// Status Operations

func (r *Repository) UpdateStatus(ctx context.Context, executionID uint, status store.TaskExecutionStatus) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.TaskExecution{}).Where("id = ?", executionID).Update("status", status).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) SetCompleted(ctx context.Context, executionID uint, exitCode int, errorMsg string) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		updates := map[string]interface{}{
			"exit_code":     &exitCode,
			"error_message": errorMsg,
			"finished_at":   time.Now(),
		}

		if exitCode == 0 && errorMsg == "" {
			updates["status"] = store.StatusSucceeded
		} else {
			updates["status"] = store.StatusFailed
		}

		if err := db.Model(&store.TaskExecution{}).Where("id = ?", executionID).Updates(updates).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Statistics and Monitoring

type ExecutionStats struct {
	TotalExecutions int64
	SuccessfulRuns  int64
	FailedRuns      int64
	AverageRunTime  time.Duration
	LastExecution   *time.Time
}

func (r *Repository) GetExecutionStats(ctx context.Context, taskID uint) (*ExecutionStats, error) {
	stats := &ExecutionStats{}

	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Total executions
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ?", taskID).Count(&stats.TotalExecutions).Error; err != nil {
			return errors.WithStack(err)
		}

		// Successful runs
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ? AND status = ?", taskID, store.StatusSucceeded).Count(&stats.SuccessfulRuns).Error; err != nil {
			return errors.WithStack(err)
		}

		// Failed runs
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ? AND status = ?", taskID, store.StatusFailed).Count(&stats.FailedRuns).Error; err != nil {
			return errors.WithStack(err)
		}

		// Last execution
		var lastExec store.TaskExecution
		if err := db.Where("task_id = ?", taskID).Order("created_at DESC").First(&lastExec).Error; err == nil {
			stats.LastExecution = &lastExec.CreatedAt
		}

		// Average run time
		var avgDuration float64
		if err := db.Model(&store.TaskExecution{}).
			Where("task_id = ? AND started_at IS NOT NULL AND finished_at IS NOT NULL", taskID).
			Select("AVG(JULIANDAY(finished_at) - JULIANDAY(started_at)) * 86400").
			Scan(&avgDuration).Error; err == nil {
			stats.AverageRunTime = time.Duration(avgDuration) * time.Second
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetExecutionStatsForUser retrieves execution statistics for a specific task and user
func (r *Repository) GetExecutionStatsForUser(ctx context.Context, taskID uint, userID uint) (*ExecutionStats, error) {
	stats := &ExecutionStats{}

	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Total executions
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ? AND user_id = ?", taskID, userID).Count(&stats.TotalExecutions).Error; err != nil {
			return errors.WithStack(err)
		}

		// Successful runs
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ? AND user_id = ? AND status = ?", taskID, userID, store.StatusSucceeded).Count(&stats.SuccessfulRuns).Error; err != nil {
			return errors.WithStack(err)
		}

		// Failed runs
		if err := db.Model(&store.TaskExecution{}).Where("task_id = ? AND user_id = ? AND status = ?", taskID, userID, store.StatusFailed).Count(&stats.FailedRuns).Error; err != nil {
			return errors.WithStack(err)
		}

		// Last execution
		var lastExec store.TaskExecution
		if err := db.Where("task_id = ? AND user_id = ?", taskID, userID).Order("created_at DESC").First(&lastExec).Error; err == nil {
			stats.LastExecution = &lastExec.CreatedAt
		}

		// Average run time
		var avgDuration float64
		if err := db.Model(&store.TaskExecution{}).
			Where("task_id = ? AND user_id = ? AND started_at IS NOT NULL AND finished_at IS NOT NULL", taskID, userID).
			Select("AVG(JULIANDAY(finished_at) - JULIANDAY(started_at)) * 86400").
			Scan(&avgDuration).Error; err == nil {
			stats.AverageRunTime = time.Duration(avgDuration) * time.Second
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *Repository) GetRecentExecutions(ctx context.Context, limit int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Preload("Task").Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// GetRecentExecutionsForUser retrieves recent executions for a specific user
func (r *Repository) GetRecentExecutionsForUser(ctx context.Context, userID uint, limit int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("user_id = ?", userID).Preload("Task").Preload("User").Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *Repository) CountByStatus(ctx context.Context, status store.TaskExecutionStatus) (int64, error) {
	var count int64
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.TaskExecution{}).Where("status = ?", status).Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Cleanup Operations

func (r *Repository) CleanupOldExecutions(ctx context.Context, olderThan time.Time) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Delete old logs first (foreign key constraint)
		if err := db.Where("execution_id IN (SELECT id FROM task_executions WHERE created_at < ?)", olderThan).Delete(&store.TaskExecutionLog{}).Error; err != nil {
			return errors.WithStack(err)
		}

		// Delete old files
		if err := db.Where("execution_id IN (SELECT id FROM task_executions WHERE created_at < ?)", olderThan).Delete(&store.TaskExecutionFile{}).Error; err != nil {
			return errors.WithStack(err)
		}

		// Delete old executions
		if err := db.Where("created_at < ?", olderThan).Delete(&store.TaskExecution{}).Error; err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
}

func (r *Repository) CleanupOrphanedLogs(ctx context.Context) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("execution_id NOT IN (SELECT id FROM task_executions)").Delete(&store.TaskExecutionLog{}).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Search and Filter Operations

func (r *Repository) SearchExecutions(ctx context.Context, filters ExecutionFilters, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Preload("Task").Preload("User").Order("created_at DESC")

		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}

		if filters.UserID > 0 {
			query = query.Where("user_id = ?", filters.UserID)
		}

		if filters.TaskID > 0 {
			query = query.Where("task_id = ?", filters.TaskID)
		}

		if filters.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
		}

		if filters.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
				query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour))
			}
		}

		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// SearchExecutionsForUser searches executions for a specific user
func (r *Repository) SearchExecutionsForUser(ctx context.Context, userID uint, filters ExecutionFilters, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("user_id = ?", userID).Preload("Task").Preload("User").Order("created_at DESC")

		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}

		if filters.TaskID > 0 {
			query = query.Where("task_id = ?", filters.TaskID)
		}

		if filters.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
		}

		if filters.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
				query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour))
			}
		}

		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

// SearchExecutionsForUserByTask searches executions for a specific user and task with filters
func (r *Repository) SearchExecutionsForUserByTask(ctx context.Context, userID uint, taskID uint, filters ExecutionFilters, limit, offset int) ([]*store.TaskExecution, error) {
	var executions []*store.TaskExecution
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Where("user_id = ? AND task_id = ?", userID, taskID).Preload("User").Order("created_at DESC")

		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}

		if filters.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
		}

		if filters.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
				query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour))
			}
		}

		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		if err := query.Find(&executions).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return executions, nil
}

type ExecutionFilters struct {
	Status   string
	UserID   uint
	TaskID   uint
	DateFrom string
	DateTo   string
}

func (r *Repository) CountExecutions(ctx context.Context, filters ExecutionFilters) (int64, error) {
	var count int64
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&store.TaskExecution{})

		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}

		if filters.UserID > 0 {
			query = query.Where("user_id = ?", filters.UserID)
		}

		if filters.TaskID > 0 {
			query = query.Where("task_id = ?", filters.TaskID)
		}

		if filters.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
		}

		if filters.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
				query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour))
			}
		}

		if err := query.Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountExecutionsForUser counts executions for a specific user with filters
func (r *Repository) CountExecutionsForUser(ctx context.Context, userID uint, filters ExecutionFilters) (int64, error) {
	var count int64
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Model(&store.TaskExecution{}).Where("user_id = ?", userID)

		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}

		if filters.TaskID > 0 {
			query = query.Where("task_id = ?", filters.TaskID)
		}

		if filters.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
				query = query.Where("created_at >= ?", dateFrom)
			}
		}

		if filters.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
				query = query.Where("created_at <= ?", dateTo.Add(24*time.Hour))
			}
		}

		if err := query.Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
