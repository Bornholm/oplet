package store

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	ImageRef    string `gorm:"unique"`
	Author      string `gorm:"index"`
	Name        string `gorm:"index"`
	Description string

	Executions []*TaskExecution
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

	// Timing
	StartedAt  *time.Time
	FinishedAt *time.Time

	// Input Parameters (JSON)
	InputParameters string `gorm:"type:text"` // JSON of form inputs

	// Logs and Files
	Logs        []TaskExecutionLog  `gorm:"foreignKey:ExecutionID"`
	OutputFiles []TaskExecutionFile `gorm:"foreignKey:ExecutionID"`
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
)

type TaskExecutionLog struct {
	gorm.Model
	ExecutionID uint   `gorm:"index"`
	Timestamp   int64  `gorm:"index"`
	Source      string // "container", "system"
	Message     string
}

type TaskExecutionFile struct {
	gorm.Model
	ExecutionID uint `gorm:"index"`
	Filename    string
	FilePath    string // Filesystem path
	FileSize    int64
	MimeType    string
	IsOutput    bool // true for output files, false for input files
}
