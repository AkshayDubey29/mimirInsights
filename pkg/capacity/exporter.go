package capacity

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"time"
)

// Exporter handles capacity report exports
type Exporter struct{}

// NewExporter creates a new report exporter
func NewExporter() *Exporter {
	return &Exporter{}
}

// ExportReport exports a capacity report in the specified format
func (e *Exporter) ExportReport(report *CapacityReport, format ExportFormat) ([]byte, string, error) {
	switch format {
	case ExportFormatJSON:
		return e.exportJSON(report)
	case ExportFormatCSV:
		return e.exportCSV(report)
	case ExportFormatHTML:
		return e.exportHTML(report)
	case ExportFormatPDF:
		return e.exportPDF(report)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports report as JSON
func (e *Exporter) exportJSON(report *CapacityReport) ([]byte, string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	filename := fmt.Sprintf("capacity_report_%s_%s.json",
		report.ReportType,
		report.GeneratedAt.Format("2006-01-02"))

	return data, filename, nil
}

// exportCSV exports report as CSV
func (e *Exporter) exportCSV(report *CapacityReport) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"Tenant", "Ingestion Rate", "Active Series", "CPU Usage %", "Memory Usage %",
		"Storage Usage GB", "Error Rate %", "Risk Level", "Growth Rate %", "Primary Bottleneck",
	}
	if err := writer.Write(header); err != nil {
		return nil, "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write tenant data
	for _, tenant := range report.TenantReports {
		record := []string{
			tenant.TenantName,
			fmt.Sprintf("%.2f", tenant.CurrentCapacity.IngestionRate),
			strconv.FormatInt(tenant.CurrentCapacity.ActiveSeries, 10),
			fmt.Sprintf("%.1f", tenant.CurrentCapacity.CPUUsage),
			fmt.Sprintf("%.1f", tenant.CurrentCapacity.MemoryUsage),
			fmt.Sprintf("%.2f", tenant.CurrentCapacity.StorageUsage),
			fmt.Sprintf("%.2f", tenant.CurrentCapacity.ErrorRate),
			tenant.RiskLevel,
			fmt.Sprintf("%.2f", tenant.UtilizationTrend.GrowthRate*100),
			tenant.BottleneckAnalysis.PrimaryBottleneck,
		}
		if err := writer.Write(record); err != nil {
			return nil, "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", fmt.Errorf("CSV writer error: %w", err)
	}

	filename := fmt.Sprintf("capacity_report_%s_%s.csv",
		report.ReportType,
		report.GeneratedAt.Format("2006-01-02"))

	return buf.Bytes(), filename, nil
}

// exportHTML exports report as HTML
func (e *Exporter) exportHTML(report *CapacityReport) ([]byte, string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Capacity Planning Report - {{.ReportType}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .summary-card { background-color: #e9ecef; padding: 15px; border-radius: 5px; }
        .tenant-table { width: 100%; border-collapse: collapse; margin-bottom: 30px; }
        .tenant-table th, .tenant-table td { border: 1px solid #dee2e6; padding: 8px; text-align: left; }
        .tenant-table th { background-color: #f8f9fa; }
        .risk-critical { background-color: #f8d7da; }
        .risk-high { background-color: #fff3cd; }
        .risk-medium { background-color: #d1ecf1; }
        .risk-low { background-color: #d4edda; }
        .recommendations { background-color: #e2f3ff; padding: 15px; border-radius: 5px; }
        ul { margin: 10px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Capacity Planning Report</h1>
        <p><strong>Report Type:</strong> {{.ReportType}}</p>
        <p><strong>Generated:</strong> {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p><strong>Period:</strong> {{.Period.StartDate.Format "2006-01-02"}} to {{.Period.EndDate.Format "2006-01-02"}}</p>
    </div>

    <div class="summary">
        <div class="summary-card">
            <h3>Total Tenants</h3>
            <p>{{.Summary.TotalTenants}}</p>
        </div>
        <div class="summary-card">
            <h3>Total Ingestion Rate</h3>
            <p>{{printf "%.2f" .Summary.TotalIngestionRate}} samples/sec</p>
        </div>
        <div class="summary-card">
            <h3>Total Active Series</h3>
            <p>{{.Summary.TotalActiveSeries}}</p>
        </div>
        <div class="summary-card">
            <h3>Average Utilization</h3>
            <p>{{printf "%.1f" .Summary.AverageUtilization}}%</p>
        </div>
    </div>

    <h2>Tenant Capacity Analysis</h2>
    <table class="tenant-table">
        <thead>
            <tr>
                <th>Tenant</th>
                <th>Ingestion Rate</th>
                <th>Active Series</th>
                <th>CPU Usage</th>
                <th>Memory Usage</th>
                <th>Risk Level</th>
                <th>Growth Rate</th>
                <th>Primary Bottleneck</th>
            </tr>
        </thead>
        <tbody>
            {{range .TenantReports}}
            <tr class="risk-{{.RiskLevel}}">
                <td>{{.TenantName}}</td>
                <td>{{printf "%.2f" .CurrentCapacity.IngestionRate}}</td>
                <td>{{.CurrentCapacity.ActiveSeries}}</td>
                <td>{{printf "%.1f" .CurrentCapacity.CPUUsage}}%</td>
                <td>{{printf "%.1f" .CurrentCapacity.MemoryUsage}}%</td>
                <td>{{.RiskLevel}}</td>
                <td>{{printf "%.2f" (mul .UtilizationTrend.GrowthRate 100)}}%</td>
                <td>{{.BottleneckAnalysis.PrimaryBottleneck}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <h2>Risk Assessment</h2>
    <p><strong>Overall Risk Level:</strong> {{.RiskAssessment.OverallRiskLevel}}</p>
    
    {{if .RiskAssessment.RiskFactors}}
    <h3>Risk Factors</h3>
    <ul>
        {{range .RiskAssessment.RiskFactors}}
        <li><strong>{{.Factor}}</strong> ({{.Severity}}): {{.Description}}</li>
        {{end}}
    </ul>
    {{end}}

    <h2>Forecasting Summary</h2>
    <p><strong>Global Trend:</strong> {{.Forecasting.GlobalTrend}}</p>
    <p><strong>Predicted Growth Rate:</strong> {{printf "%.2f" (mul .Forecasting.PredictedGrowthRate 100)}}%</p>
    {{if .Forecasting.CapacityExhaustionDate}}
    <p><strong>Estimated Capacity Exhaustion:</strong> {{.Forecasting.CapacityExhaustionDate.Format "2006-01-02"}}</p>
    {{end}}

    <div class="recommendations">
        <h2>Recommendations</h2>
        <ul>
            {{range .Recommendations}}
            <li>{{.}}</li>
            {{end}}
        </ul>
    </div>

    {{if .Forecasting.ScalingRecommendations}}
    <div class="recommendations">
        <h3>Scaling Recommendations</h3>
        <ul>
            {{range .Forecasting.ScalingRecommendations}}
            <li>{{.}}</li>
            {{end}}
        </ul>
    </div>
    {{end}}
</body>
</html>`

	// Custom template functions
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return nil, "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	filename := fmt.Sprintf("capacity_report_%s_%s.html",
		report.ReportType,
		report.GeneratedAt.Format("2006-01-02"))

	return buf.Bytes(), filename, nil
}

// exportPDF exports report as PDF (simplified - would use a PDF library in production)
func (e *Exporter) exportPDF(report *CapacityReport) ([]byte, string, error) {
	// For this example, we'll generate a text-based PDF content
	// In production, you would use a library like gofpdf or wkhtmltopdf

	content := fmt.Sprintf(`
CAPACITY PLANNING REPORT
========================

Report Type: %s
Generated: %s
Period: %s to %s

SUMMARY
-------
Total Tenants: %d
Total Ingestion Rate: %.2f samples/sec
Total Active Series: %d
Average Utilization: %.1f%%

TENANT ANALYSIS
---------------
`,
		report.ReportType,
		report.GeneratedAt.Format("2006-01-02 15:04:05"),
		report.Period.StartDate.Format("2006-01-02"),
		report.Period.EndDate.Format("2006-01-02"),
		report.Summary.TotalTenants,
		report.Summary.TotalIngestionRate,
		report.Summary.TotalActiveSeries,
		report.Summary.AverageUtilization,
	)

	for _, tenant := range report.TenantReports {
		content += fmt.Sprintf(`
Tenant: %s
- Ingestion Rate: %.2f samples/sec
- Active Series: %d
- CPU Usage: %.1f%%
- Memory Usage: %.1f%%
- Risk Level: %s
- Growth Rate: %.2f%%
- Primary Bottleneck: %s

`,
			tenant.TenantName,
			tenant.CurrentCapacity.IngestionRate,
			tenant.CurrentCapacity.ActiveSeries,
			tenant.CurrentCapacity.CPUUsage,
			tenant.CurrentCapacity.MemoryUsage,
			tenant.RiskLevel,
			tenant.UtilizationTrend.GrowthRate*100,
			tenant.BottleneckAnalysis.PrimaryBottleneck,
		)
	}

	content += "\nRECOMMENDATIONS\n---------------\n"
	for i, rec := range report.Recommendations {
		content += fmt.Sprintf("%d. %s\n", i+1, rec)
	}

	filename := fmt.Sprintf("capacity_report_%s_%s.pdf",
		report.ReportType,
		report.GeneratedAt.Format("2006-01-02"))

	return []byte(content), filename, nil
}

// ScheduledExport represents a scheduled export configuration
type ScheduledExport struct {
	ID           string       `json:"id"`
	ReportType   string       `json:"report_type"`
	Format       ExportFormat `json:"format"`
	Schedule     string       `json:"schedule"` // cron format
	Recipients   []string     `json:"recipients"`
	Enabled      bool         `json:"enabled"`
	LastExported time.Time    `json:"last_exported"`
	NextExport   time.Time    `json:"next_export"`
}

// EmailService interface for sending reports
type EmailService interface {
	SendReport(recipients []string, subject string, body string, attachment []byte, filename string) error
}

// Scheduler handles scheduled exports
type Scheduler struct {
	planner      *Planner
	exporter     *Exporter
	emailService EmailService
	exports      map[string]*ScheduledExport
}

// NewScheduler creates a new export scheduler
func NewScheduler(planner *Planner, exporter *Exporter, emailService EmailService) *Scheduler {
	return &Scheduler{
		planner:      planner,
		exporter:     exporter,
		emailService: emailService,
		exports:      make(map[string]*ScheduledExport),
	}
}

// AddScheduledExport adds a new scheduled export
func (s *Scheduler) AddScheduledExport(export *ScheduledExport) {
	s.exports[export.ID] = export
}

// RemoveScheduledExport removes a scheduled export
func (s *Scheduler) RemoveScheduledExport(id string) {
	delete(s.exports, id)
}

// GetScheduledExports returns all scheduled exports
func (s *Scheduler) GetScheduledExports() map[string]*ScheduledExport {
	return s.exports
}
