package store

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	ImageRef       string `gorm:"unique"`
	Author         string `gorm:"index"`
	Name           string `gorm:"index"`
	Description    string
	Configurations []*TaskConfiguration `gorm:"constraint:OnDelete:CASCADE;"`

	Executions []*TaskExecution `gorm:"constraint:OnDelete:CASCADE;"`
}

type TaskExecution struct {
	gorm.Model

	// Relationships
	Task   *Task
	TaskID uint
	User   *User
	UserID uint

	// Execution Details
	ContainerID  string              `gorm:"index"`
	Status       TaskExecutionStatus `gorm:"index"`
	ExitCode     *int                // Nullable until completion
	ErrorMessage string              `gorm:"type:text"`

	RunnerToken string `gorm:"unique"`

	// Timing
	StartedAt  *time.Time
	FinishedAt *time.Time

	// Input Parameters (JSON)
	InputParameters string `gorm:"type:text"` // JSON of form inputs

	// Logs and Files
	Logs        []TaskExecutionLog  `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE;"`
	OutputFiles []TaskExecutionFile `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE;"`
}

type TaskExecutionStatus string

const (
	StatusPending           TaskExecutionStatus = "pending"
	StatusPullingImage      TaskExecutionStatus = "pulling_image"
	StatusImagePulled       TaskExecutionStatus = "image_pulled"
	StatusCreatingContainer TaskExecutionStatus = "creating_container"
	StatusContainerCreated  TaskExecutionStatus = "container_created"
	StatusUploadingFiles    TaskExecutionStatus = "uploading_files"
	StatusFilesUploaded     TaskExecutionStatus = "files_uploaded"
	StatusStartingContainer TaskExecutionStatus = "starting_container"
	StatusContainerStarted  TaskExecutionStatus = "container_started"
	StatusRunning           TaskExecutionStatus = "running"
	StatusFinished          TaskExecutionStatus = "finished"
	StatusDownloadingFiles  TaskExecutionStatus = "downloading_files"
	StatusFilesDownloaded   TaskExecutionStatus = "files_downloaded"
	StatusSucceeded         TaskExecutionStatus = "succeeded"
	StatusFailed            TaskExecutionStatus = "failed"
	StatusKilled            TaskExecutionStatus = "killed"
)

type TaskExecutionLog struct {
	gorm.Model
	Execution   *TaskExecution
	ExecutionID uint   `gorm:"index:task_execution_log,unique"`
	Timestamp   int64  `gorm:"index:task_execution_log,unique"`
	Source      string `gorm:"index:task_execution_log,unique"` // "container", "system"
	Clock       uint   `gorm:"index:task_execution_log,unique"`
	Message     string
}

type TaskExecutionFile struct {
	gorm.Model
	Execution   *TaskExecution
	ExecutionID uint `gorm:"index"`
	Filename    string
	FilePath    string // Filesystem path
	FileSize    int64
	MimeType    string
	IsOutput    bool // true for output files, false for input files
}

type TaskConfiguration struct {
	gorm.Model

	Task   *Task
	TaskID uint `gorm:"index:task_config_index,unique"`

	Name  string `gorm:"index:task_config_index,unique"`
	Value string `gorm:"index:task_config_index,unique"`
}
