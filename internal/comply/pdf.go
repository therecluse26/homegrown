package comply

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// ─── Portfolio PDF ────────────────────────────────────────────────────────────

// renderPortfolioPDF generates a PDF for a learning portfolio.
// Returns the PDF bytes or an error. [14-comply §9.2]
func renderPortfolioPDF(
	portfolio *ComplyPortfolio,
	items []ComplyPortfolioItem,
	studentName string,
	attendance *AttendanceSummaryRow,
) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)

	// ── Cover Page ────────────────────────────────────────────────────────────
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 12, "Learning Portfolio", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 16)
	pdf.CellFormat(0, 8, studentName, "", 1, "C", false, 0, "")
	pdf.Ln(4)

	dateRange := fmt.Sprintf("%s – %s",
		portfolio.DateRangeStart.Format("January 2, 2006"),
		portfolio.DateRangeEnd.Format("January 2, 2006"),
	)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 7, dateRange, "", 1, "C", false, 0, "")
	pdf.Ln(4)

	if portfolio.Organization != "" {
		pdf.SetFont("Arial", "I", 11)
		pdf.CellFormat(0, 7, portfolio.Organization, "", 1, "C", false, 0, "")
	}

	if portfolio.Description != nil && *portfolio.Description != "" {
		pdf.Ln(8)
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 6, *portfolio.Description, "", "C", false)
	}

	// ── Activity Summary ──────────────────────────────────────────────────────
	if len(items) > 0 {
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 10, "Activity Summary", "", 1, "L", false, 0, "")
		pdf.Ln(4)

		// Group items by subject.
		type subjectGroup struct {
			subject string
			items   []ComplyPortfolioItem
		}
		subjectMap := make(map[string]*subjectGroup)
		subjectOrder := []string{}
		for _, item := range items {
			subject := "General"
			if item.CachedSubject != nil && *item.CachedSubject != "" {
				subject = *item.CachedSubject
			}
			if _, ok := subjectMap[subject]; !ok {
				subjectMap[subject] = &subjectGroup{subject: subject}
				subjectOrder = append(subjectOrder, subject)
			}
			subjectMap[subject].items = append(subjectMap[subject].items, item)
		}

		// Header row.
		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(220, 220, 220)
		pdf.CellFormat(50, 7, "Date", "1", 0, "L", true, 0, "")
		pdf.CellFormat(90, 7, "Activity", "1", 0, "L", true, 0, "")
		pdf.CellFormat(0, 7, "Subject", "1", 1, "L", true, 0, "")

		pdf.SetFont("Arial", "", 9)
		pdf.SetFillColor(255, 255, 255)
		for _, subj := range subjectOrder {
			group := subjectMap[subj]
			// Subject header row.
			pdf.SetFont("Arial", "B", 9)
			pdf.SetFillColor(240, 240, 240)
			pdf.CellFormat(0, 6, "  "+group.subject, "1", 1, "L", true, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.SetFillColor(255, 255, 255)
			for _, item := range group.items {
				pdf.CellFormat(50, 6, item.CachedDate.Format("Jan 2, 2006"), "1", 0, "L", false, 0, "")
				title := item.CachedTitle
				if len(title) > 55 {
					title = title[:52] + "..."
				}
				pdf.CellFormat(90, 6, title, "1", 0, "L", false, 0, "")
				pdf.CellFormat(0, 6, group.subject, "1", 1, "L", false, 0, "")
			}
		}

		// Summary count.
		pdf.Ln(4)
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(0, 7, fmt.Sprintf("Total entries: %d", len(items)), "", 1, "L", false, 0, "")
	}

	// ── Attendance Summary ────────────────────────────────────────────────────
	if attendance != nil && portfolio.IncludeAttendance {
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 10, "Attendance Summary", "", 1, "L", false, 0, "")
		pdf.Ln(4)

		totalDays := attendance.PresentFull + attendance.PresentPartial + attendance.Absent + attendance.NotApplicable
		rows := [][]string{
			{"Present (Full Day)", fmt.Sprintf("%d", attendance.PresentFull)},
			{"Present (Partial Day)", fmt.Sprintf("%d", attendance.PresentPartial)},
			{"Absent", fmt.Sprintf("%d", attendance.Absent)},
			{"Not Applicable", fmt.Sprintf("%d", attendance.NotApplicable)},
			{"Total Days Recorded", fmt.Sprintf("%d", totalDays)},
		}
		if attendance.TotalMinutes > 0 {
			hours := float64(attendance.TotalMinutes) / 60.0
			rows = append(rows, []string{"Total Learning Hours", fmt.Sprintf("%.1f", hours)})
		}

		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(220, 220, 220)
		pdf.CellFormat(100, 7, "Metric", "1", 0, "L", true, 0, "")
		pdf.CellFormat(0, 7, "Value", "1", 1, "L", true, 0, "")

		pdf.SetFont("Arial", "", 10)
		for _, row := range rows {
			pdf.CellFormat(100, 7, row[0], "1", 0, "L", false, 0, "")
			pdf.CellFormat(0, 7, row[1], "1", 1, "L", false, 0, "")
		}
	}

	// ── Footer on each page ───────────────────────────────────────────────────
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 10,
			fmt.Sprintf("Generated %s — Homegrown Academy", time.Now().Format("January 2, 2006")),
			"", 0, "C", false, 0, "")
	})

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("comply: render portfolio PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ─── Transcript PDF ───────────────────────────────────────────────────────────

// renderTranscriptPDF generates a PDF for an academic transcript.
// Returns the PDF bytes or an error. [14-comply §9.3]
func renderTranscriptPDF(
	transcript *ComplyTranscript,
	courses []ComplyCourse,
	studentName string,
	unweightedGPA float64,
	weightedGPA float64,
) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ── Header ────────────────────────────────────────────────────────────────
	pdf.SetFont("Arial", "B", 22)
	pdf.CellFormat(0, 12, "Academic Transcript", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, studentName, "", 1, "C", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(0, 7, transcript.Title, "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// ── Course Table ──────────────────────────────────────────────────────────
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Course Record", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Table header.
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(55, 7, "Course Title", "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 7, "Subject", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 7, "Year", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 7, "Grade", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 7, "Credits", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 7, "Level", "1", 0, "C", true, 0, "")
	pdf.CellFormat(0, 7, "GPA Pts", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(255, 255, 255)

	var totalCredits float64
	for _, c := range courses {
		title := c.Title
		if len(title) > 32 {
			title = title[:29] + "..."
		}
		subj := c.Subject
		if len(subj) > 16 {
			subj = subj[:13] + "..."
		}
		grade := ""
		if c.GradeLetter != nil {
			grade = *c.GradeLetter
		}
		gpaPoints := ""
		if c.GradePoints != nil {
			gpaPoints = fmt.Sprintf("%.2f", *c.GradePoints)
		}
		level := c.Level
		if len(level) > 8 {
			level = level[:7] + "."
		}

		pdf.CellFormat(55, 6, title, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, subj, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 6, c.SchoolYear, "1", 0, "C", false, 0, "")
		pdf.CellFormat(15, 6, grade, "1", 0, "C", false, 0, "")
		pdf.CellFormat(15, 6, fmt.Sprintf("%.2f", c.Credits), "1", 0, "C", false, 0, "")
		pdf.CellFormat(15, 6, level, "1", 0, "C", false, 0, "")
		pdf.CellFormat(0, 6, gpaPoints, "1", 1, "C", false, 0, "")
		totalCredits += c.Credits
	}

	// Totals row.
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(55, 6, "TOTAL", "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 6, "", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 6, "", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 6, "", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 6, fmt.Sprintf("%.2f", totalCredits), "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 6, "", "1", 0, "C", true, 0, "")
	pdf.CellFormat(0, 6, "", "1", 1, "C", true, 0, "")

	// ── GPA Summary ───────────────────────────────────────────────────────────
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Grade Point Average", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(80, 7, "GPA Type", "1", 0, "L", true, 0, "")
	pdf.CellFormat(0, 7, "Value", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetFillColor(255, 255, 255)
	pdf.CellFormat(80, 7, "Unweighted GPA (4.0 scale)", "1", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("%.2f", unweightedGPA), "1", 1, "C", false, 0, "")
	pdf.CellFormat(80, 7, "Weighted GPA", "1", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("%.2f", weightedGPA), "1", 1, "C", false, 0, "")
	pdf.CellFormat(80, 7, "Total Credits Earned", "1", 0, "L", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("%.2f", totalCredits), "1", 1, "C", false, 0, "")

	// ── Footer ────────────────────────────────────────────────────────────────
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 10,
			fmt.Sprintf("Generated %s — Homegrown Academy", time.Now().Format("January 2, 2006")),
			"", 0, "C", false, 0, "")
	})

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("comply: render transcript PDF: %w", err)
	}
	return buf.Bytes(), nil
}
