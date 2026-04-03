package plan

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// renderCalendarPDF generates a PDF for the given schedule items, grouped by day.
// Follows the same PDF generation pattern as comply/pdf.go. [17-planning §13.2]
func renderCalendarPDF(start, end time.Time, items []ScheduleItem) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)

	// ── Title Page ────────────────────────────────────────────────────────────
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 22)
	pdf.CellFormat(0, 12, "Schedule", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	dateRange := fmt.Sprintf("%s - %s",
		start.Format("January 2, 2006"),
		end.Add(-24*time.Hour).Format("January 2, 2006"),
	)
	pdf.SetFont("Arial", "", 13)
	pdf.CellFormat(0, 8, dateRange, "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// ── Group items by date ───────────────────────────────────────────────────
	grouped := make(map[string][]ScheduleItem)
	for _, item := range items {
		key := truncateToDate(item.StartDate).Format("2006-01-02")
		grouped[key] = append(grouped[key], item)
	}

	// ── Render each day ──────────────────────────────────────────────────────
	for d := truncateToDate(start); d.Before(truncateToDate(end)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dayItems := grouped[key]

		// Day header
		pdf.SetFont("Arial", "B", 12)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(0, 8, d.Format("Monday, January 2"), "B", 1, "L", true, 0, "")
		pdf.Ln(2)

		if len(dayItems) == 0 {
			pdf.SetFont("Arial", "I", 10)
			pdf.CellFormat(0, 6, "No scheduled items", "", 1, "L", false, 0, "")
		} else {
			for _, item := range dayItems {
				// Time column
				timeStr := ""
				if item.StartTime != nil {
					timeStr = *item.StartTime
					if item.EndTime != nil {
						timeStr += " - " + *item.EndTime
					}
				}

				pdf.SetFont("Arial", "", 10)
				if timeStr != "" {
					pdf.CellFormat(35, 6, timeStr, "", 0, "L", false, 0, "")
				} else {
					pdf.CellFormat(35, 6, "", "", 0, "L", false, 0, "")
				}

				// Title
				pdf.SetFont("Arial", "", 10)
				pdf.CellFormat(90, 6, item.Title, "", 0, "L", false, 0, "")

				// Category
				pdf.SetFont("Arial", "I", 9)
				pdf.CellFormat(0, 6, string(item.Category), "", 1, "L", false, 0, "")

				// Completed indicator
				if item.IsCompleted {
					pdf.SetFont("Arial", "I", 8)
					pdf.CellFormat(35, 5, "", "", 0, "L", false, 0, "")
					pdf.CellFormat(0, 5, "Completed", "", 1, "L", false, 0, "")
				}
			}
		}
		pdf.Ln(3)
	}

	// ── Output ────────────────────────────────────────────────────────────────
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("plan: render calendar PDF: %w", err)
	}
	return buf.Bytes(), nil
}
