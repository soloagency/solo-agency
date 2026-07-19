package main

// render.go — in-binary report renderer interface. crm_store's Python version
// shelled out to `python3 report_renderer.py` (three call sites); here the
// renderer is a function call. The full renderer port (MD→HTML + scrub gate +
// PDF via Chrome headless) replaces the body of renderReportFile; until then
// it degrades exactly like Python does on a machine without python3:
// html_rendered=false, markdown still written.

type rendererRequest struct {
	Input        string
	OutputHTML   string
	Title        string
	ClientName   string
	ReportKind   string
	ReportDate   string
	ClientFacing bool
	FailOnScrub  bool
}

type rendererResult struct {
	RC         int // 0 ok, 2 html-ok-pdf-missing, 3 scrub-blocked, 1 failed
	BlindTerms []any
}

// renderReportFile is swapped to the native implementation by the renderer port
// (see renderer.go). The variable indirection keeps crm_reports.go stable.
var renderReportFile = func(req rendererRequest) rendererResult {
	return rendererResult{RC: 1}
}
