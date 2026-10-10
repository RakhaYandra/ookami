package model

import "context"

type Category string

const (
	CategorySystem      Category = "system"
	CategoryDevelopment Category = "development"
	CategoryStorage     Category = "storage"
	CategoryNetwork     Category = "network"
	CategoryGPU         Category = "gpu"
	CategoryServices    Category = "services"
)

type Severity string

const (
	SeverityPass     Severity = "pass"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityUnknown  Severity = "unknown"
)

type Remediation struct {
	Description  string
	Command      string
	Args         []string
	Safe         bool
	RequiresSudo bool
}

type Result struct {
	ID          string
	Category    Category
	Severity    Severity
	Title       string
	Message     string
	Details     map[string]any
	Remediation *Remediation
}

type CheckMetadata struct {
	ID          string
	Name        string
	Description string
	Category    Category
	Optional    bool
}

type Check interface {
	ID() string
	Category() Category
	Metadata() CheckMetadata
	Run(ctx context.Context) Result
}
