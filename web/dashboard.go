// Package web provides the server-rendered HTML dashboard for GoQueue.
package web

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/betim/goqueue/queue"
)

//go:embed templates/*.html
var templateFS embed.FS

// Dashboard handles the web UI routes.
type Dashboard struct {
	manager       *queue.Manager
	dashboardTmpl *template.Template
	jobDetailTmpl *template.Template
}

// NewDashboard parses templates and returns a ready Dashboard.
func NewDashboard(manager *queue.Manager) *Dashboard {
	dashboard := template.Must(
		template.ParseFS(templateFS, "templates/layout.html", "templates/dashboard.html"),
	)
	jobDetail := template.Must(
		template.ParseFS(templateFS, "templates/layout.html", "templates/job_detail.html"),
	)

	return &Dashboard{
		manager:       manager,
		dashboardTmpl: dashboard,
		jobDetailTmpl: jobDetail,
	}
}

// RegisterRoutes adds dashboard routes to the given mux.
func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", d.Index)
	mux.HandleFunc("/jobs/", d.JobDetail)
}

// Index renders the main dashboard page.
func (d *Dashboard) Index(w http.ResponseWriter, r *http.Request) {
	// Only handle exact "/" path — let other routes fall through.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Stats map[string]int
		Jobs  []*queue.Job
	}{
		Stats: d.manager.Stats(),
		Jobs:  d.manager.ListJobs(""),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	d.dashboardTmpl.Execute(w, data)
}

// JobDetail renders a single job's detail page.
func (d *Dashboard) JobDetail(w http.ResponseWriter, r *http.Request) {
	// Handle cancel action
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/cancel") {
		id := strings.TrimPrefix(r.URL.Path, "/jobs/")
		id = strings.TrimSuffix(id, "/cancel")
		d.manager.DeleteJob(id)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	id = strings.TrimRight(id, "/")

	job, err := d.manager.GetJob(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Job *queue.Job
	}{
		Job: job,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	d.jobDetailTmpl.Execute(w, data)
}
