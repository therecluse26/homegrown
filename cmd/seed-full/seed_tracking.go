package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ─── Activity Logs ────────────────────────────────────────────────────────────

func (s *seeder) seedActivityLogs() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 {
				continue
			}

			// Use family's primary methodology UUID for activity logs.
			methID := methodologyUUIDs[f.PrimaryMethodology]

			// Activity level * 6 logs per student.
			logsPerStudent := f.ActivityLevel * 6
			for _, st := range f.Students {
				earliest := maxTime(schoolStart, f.CreatedAt)
				for j := 0; j < logsPerStudent; j++ {
					seq++
					at := activityTemplates[s.rng.Intn(len(activityTemplates))]
					date := randomDate(s.rng, earliest, s.now)
					contentID := s.activityDefIDs[s.rng.Intn(len(s.activityDefIDs))]

					rows = append(rows, []any{
						uid(dActLog, seq), f.ID, st.ID,
						at.Title, at.Description,
						"{" + strings.Join(at.Subjects, ",") + "}",
						contentID, methID, at.Duration,
						date.Format("2006-01-02"),
						date, date,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO learn_activity_logs (id, family_id, student_id, title, description, subject_tags, content_id, methodology_id, duration_minutes, activity_date, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Journal Entries ──────────────────────────────────────────────────────────

func (s *seeder) seedJournalEntries() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		natures := []string{"a beautiful cardinal", "wildflowers blooming", "a hawk circling overhead", "tadpoles in the creek", "mushrooms on a log"}
		weathers := []string{"sunny and warm", "cool and breezy", "overcast but pleasant", "crisp autumn air", "warm spring morning"}
		drawings := []string{"the cardinal", "a wildflower cluster", "the hawk", "the tadpoles", "the mushroom cap"}
		topics := []string{"ancient Rome", "the water cycle", "photosynthesis", "the Civil War", "multiplication patterns"}
		facts := []string{"they built aqueducts", "water can be a solid, liquid, or gas", "plants make their own food", "it lasted four years", "they repeat in interesting ways"}
		desires := []string{"Roman engineering", "weather patterns", "how trees communicate", "Abraham Lincoln", "prime numbers"}
		hobbies := []string{"drawing", "building things", "reading", "gardening", "cooking"}
		reasons := []string{"it helps me relax", "I learn new things", "I can use my imagination", "I love being outside", "it tastes delicious"}
		goals := []string{"get better at shading", "build a birdhouse", "read 50 books this year", "grow a pumpkin", "make dinner for my family"}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 {
				continue
			}

			entriesPerStudent := f.ActivityLevel * 2
			for _, st := range f.Students {
				earliest := maxTime(schoolStart, f.CreatedAt)
				for j := 0; j < entriesPerStudent; j++ {
					seq++
					jt := journalTemplates[s.rng.Intn(len(journalTemplates))]
					date := randomDate(s.rng, earliest, s.now)

					// Fill template placeholders.
					content := fmt.Sprintf(jt.Content,
						natures[s.rng.Intn(len(natures))],
						weathers[s.rng.Intn(len(weathers))],
						drawings[s.rng.Intn(len(drawings))],
					)
					// For templates with different placeholder types, use appropriate fillers.
					switch s.rng.Intn(3) {
					case 1:
						content = fmt.Sprintf(jt.Content,
							topics[s.rng.Intn(len(topics))],
							facts[s.rng.Intn(len(facts))],
							desires[s.rng.Intn(len(desires))],
						)
					case 2:
						content = fmt.Sprintf(jt.Content,
							hobbies[s.rng.Intn(len(hobbies))],
							reasons[s.rng.Intn(len(reasons))],
							goals[s.rng.Intn(len(goals))],
						)
					}

					rows = append(rows, []any{
						uid(dJournal, seq), f.ID, st.ID,
						jt.Type, jt.Title, content,
						"{reading,language_arts}",
						date.Format("2006-01-02"),
						date, date,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO learn_journal_entries (id, family_id, student_id, entry_type, title, content, subject_tags, entry_date, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Reading Progress ─────────────────────────────────────────────────────────

func (s *seeder) seedReadingProgress() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 || len(s.readingItemIDs) == 0 {
				continue
			}

			booksPerStudent := f.ActivityLevel * 3
			for _, st := range f.Students {
				bookIdxs := pickN(s.rng, len(s.readingItemIDs), min(booksPerStudent, len(s.readingItemIDs)))
				for _, bIdx := range bookIdxs {
					seq++
					status := "in_progress"
					var completedAt any = nil
					startedAt := randomDate(s.rng, f.CreatedAt, s.now)

					if s.rng.Float64() < 0.5 {
						status = "completed"
						completedAt = randomDate(s.rng, startedAt, s.now)
					} else if s.rng.Float64() < 0.2 {
						status = "to_read"
						startedAt = time.Time{} // zero time means not started
					}

					var startedAtVal any = startedAt
					if startedAt.IsZero() {
						startedAtVal = nil
					}

					rows = append(rows, []any{
						uid(dReadProg, seq), f.ID, st.ID,
						s.readingItemIDs[bIdx], nil,
						status, startedAtVal, completedAt, "",
						s.now, s.now,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO learn_reading_progress (id, family_id, student_id, reading_item_id, reading_list_id, status, started_at, completed_at, notes, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Assessment Results ───────────────────────────────────────────────────────

func (s *seeder) seedAssessmentResults() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 || len(s.assessmentDefIDs) == 0 {
				continue
			}

			resultsPerStudent := f.ActivityLevel
			for _, st := range f.Students {
				for j := 0; j < resultsPerStudent; j++ {
					seq++
					defIdx := s.rng.Intn(len(s.assessmentDefIDs))
					score := 60 + s.rng.Intn(41) // 60–100
					date := randomDate(s.rng, maxTime(schoolStart, f.CreatedAt), s.now)

					rows = append(rows, []any{
						uid(dAssResult, seq), f.ID, st.ID,
						s.assessmentDefIDs[defIdx],
						score, 100, 1.0, "",
						date.Format("2006-01-02"),
						date, date,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO learn_assessment_results (id, family_id, student_id, assessment_def_id, score, max_score, weight, notes, assessment_date, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Compliance: State Configs, Attendance, Courses ───────────────────────────

func (s *seeder) seedCompliance() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		now := s.now

		// ── State configs (all 50 states + DC) ──
		var stateRows [][]any
		regLevels := map[string]string{
			"TX": "low", "CA": "moderate", "FL": "low", "NC": "moderate",
			"VA": "high", "OH": "moderate", "PA": "high", "GA": "low",
			"NY": "high", "IL": "moderate",
		}
		for code, name := range stateNames {
			level := regLevels[code]
			if level == "" {
				level = "moderate"
			}
			notifReq := level == "high" || level == "moderate"
			assessReq := level == "high"
			attendReq := level == "high" || level == "moderate"
			attendDays := 180
			attendHours := 900
			reqSubjects := "{english,mathematics,science,social_studies}"

			stateRows = append(stateRows, []any{
				code, name, notifReq, "File with local school district",
				reqSubjects, assessReq, "Annual standardized testing",
				true, "Maintain attendance records",
				attendReq, attendDays, attendHours, "180 instructional days required",
				false, "", level, now, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO comply_state_configs (state_code, state_name, notification_required, notification_details, required_subjects, assessment_required, assessment_details, record_keeping_required, record_keeping_details, attendance_required, attendance_days, attendance_hours, attendance_details, umbrella_school_available, umbrella_school_details, regulation_level, synced_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (state_code) DO NOTHING",
			stateRows,
		); err != nil {
			return fmt.Errorf("state configs: %w", err)
		}

		// ── Family configs (~60% of families) ──
		var cfgRows [][]any
		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if s.rng.Float64() > 0.60 {
				continue
			}
			cfgRows = append(cfgRows, []any{
				f.ID, f.StateCode,
				"2025-08-15", "2026-05-30", 180,
				now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO comply_family_configs (family_id, state_code, school_year_start, school_year_end, total_school_days, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (family_id) DO NOTHING",
			cfgRows,
		); err != nil {
			return fmt.Errorf("family configs: %w", err)
		}

		// ── Attendance records (active families with students) ──
		var attendRows [][]any
		attendSeq := 0
		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 || f.ActivityLevel < 2 {
				continue
			}

			// Generate attendance for ~20 school days per student.
			daysToGenerate := f.ActivityLevel * 5
			for _, st := range f.Students {
				startDate := time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC)
				dayOffset := 0
				generated := 0
				for generated < daysToGenerate && dayOffset < 200 {
					d := startDate.AddDate(0, 0, dayOffset)
					dayOffset++
					// Skip weekends.
					if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
						continue
					}
					// Skip if after now.
					if d.After(s.now) {
						break
					}
					generated++
					attendSeq++

					status := "present_full"
					duration := 360 // 6 hours
					r := s.rng.Float64()
					if r < 0.05 {
						status = "absent"
						duration = 0
					} else if r < 0.10 {
						status = "present_partial"
						duration = 180
					}

					attendRows = append(attendRows, []any{
						uid(dAttend, attendSeq), f.ID, st.ID,
						d.Format("2006-01-02"), status, duration,
						"", false, false, d, d,
					})
				}
			}
		}
		if err := execBatch(tx,
			"INSERT INTO comply_attendance (id, family_id, student_id, attendance_date, status, duration_minutes, notes, is_auto, manual_override, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			attendRows,
		); err != nil {
			return fmt.Errorf("attendance: %w", err)
		}

		// ── Assessment records ──
		var compAssRows [][]any
		caSeq := 0
		assTypes := []string{"test", "quiz", "project", "assignment", "portfolio_piece"}
		subjects := []string{"Language Arts", "Mathematics", "Science", "History", "Latin", "Art", "Music"}
		grades := []struct {
			letter string
			points float64
		}{
			{"A", 4.0}, {"A-", 3.7}, {"B+", 3.3}, {"B", 3.0}, {"B-", 2.7},
			{"C+", 2.3}, {"C", 2.0},
		}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 || f.ActivityLevel < 2 {
				continue
			}

			for _, st := range f.Students {
				numAss := f.ActivityLevel
				for j := 0; j < numAss; j++ {
					caSeq++
					assType := assTypes[s.rng.Intn(len(assTypes))]
					subj := subjects[s.rng.Intn(len(subjects))]
					score := 70 + s.rng.Intn(31)
					gradeIdx := s.rng.Intn(len(grades))
					date := randomDate(s.rng, maxTime(time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC), f.CreatedAt), s.now)

					compAssRows = append(compAssRows, []any{
						uid(dComplyAss, caSeq), f.ID, st.ID,
						fmt.Sprintf("%s %s", subj, assType), subj, assType,
						score, 100, grades[gradeIdx].letter, grades[gradeIdx].points,
						score >= 70, nil,
						date.Format("2006-01-02"), "",
						date, date,
					})
				}
			}
		}
		if err := execBatch(tx,
			"INSERT INTO comply_assessment_records (id, family_id, student_id, title, subject, assessment_type, score, max_score, grade_letter, grade_points, is_passing, source_activity_id, assessment_date, notes, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			compAssRows,
		); err != nil {
			return fmt.Errorf("comply assessments: %w", err)
		}

		// ── Courses (for active families, ~3 courses per student) ──
		var courseRows [][]any
		courseSeq := 0
		courseSubjects := []string{
			"Language Arts", "Mathematics", "Science", "History",
			"Latin", "Art", "Music", "Physical Education",
		}
		levels := []string{"regular", "regular", "regular", "honors"}
		semesters := []string{"fall", "spring", "full_year"}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if len(f.Students) == 0 || f.ActivityLevel < 3 {
				continue
			}

			for _, st := range f.Students {
				numCourses := 2 + s.rng.Intn(4) // 2–5 courses
				usedSubjects := pickN(s.rng, len(courseSubjects), min(numCourses, len(courseSubjects)))
				for _, sIdx := range usedSubjects {
					courseSeq++
					subj := courseSubjects[sIdx]
					gradeIdx := s.rng.Intn(len(grades))
					level := levels[s.rng.Intn(len(levels))]
					sem := semesters[s.rng.Intn(len(semesters))]
					credits := 0.5
					if sem == "full_year" {
						credits = 1.0
					}

					courseRows = append(courseRows, []any{
						uid(dCourse, courseSeq), f.ID, st.ID, nil,
						subj + " " + st.GradeLevel, subj, gradeToNum(st.GradeLevel),
						credits, grades[gradeIdx].letter, grades[gradeIdx].points,
						level, "2025-2026", sem,
						now, now,
					})
				}
			}
		}
		if err := execBatch(tx,
			"INSERT INTO comply_courses (id, family_id, student_id, transcript_id, title, subject, grade_level, credits, grade_letter, grade_points, level, school_year, semester, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			courseRows,
		); err != nil {
			return fmt.Errorf("courses: %w", err)
		}

		return nil
	})
}

// ─── Schedule Items ───────────────────────────────────────────────────────────

func (s *seeder) seedScheduleItems() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		scheduleTitles := []string{
			"Morning Basket", "Math Lesson", "Read Aloud", "Nature Study",
			"Latin Practice", "Copywork", "Science Experiment", "History Reading",
			"Art Time", "Music Appreciation", "PE / Outdoor Play", "Free Reading",
			"Handwriting Practice", "Poetry Recitation", "Geography",
		}
		times := [][2]string{
			{"08:00", "08:30"}, {"08:30", "09:15"}, {"09:15", "09:45"},
			{"09:45", "10:30"}, {"10:30", "11:00"}, {"11:00", "11:30"},
			{"13:00", "13:45"}, {"13:45", "14:30"}, {"14:30", "15:00"},
		}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if f.ActivityLevel < 2 {
				continue
			}

			// Generate schedule items for the next 5 school days.
			for day := 0; day < 5; day++ {
				date := s.now.AddDate(0, 0, day)
				if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
					continue
				}

				numItems := 3 + s.rng.Intn(5) // 3–7 items per day
				for j := 0; j < numItems && j < len(times); j++ {
					seq++
					title := scheduleTitles[s.rng.Intn(len(scheduleTitles))]
					cat := scheduleCategories[s.rng.Intn(len(scheduleCategories))]
					timeSlot := times[j]

					// Assign to specific student or whole family.
					var studentID any = nil
					if len(f.Students) > 0 && s.rng.Float64() < 0.7 {
						studentID = f.Students[s.rng.Intn(len(f.Students))].ID
					}

					completed := date.Before(s.now)

					rows = append(rows, []any{
						uid(dSchedItem, seq), f.ID, studentID,
						title, "", date.Format("2006-01-02"),
						timeSlot[0], timeSlot[1], 30,
						cat, nil, "", completed, nil, nil, nil, nil, "",
						s.now, s.now,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO plan_schedule_items (id, family_id, student_id, title, description, start_date, start_time, end_time, duration_minutes, category, subject_id, color, is_completed, completed_at, linked_activity_id, linked_event_id, recurrence_rule, notes, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Notifications ────────────────────────────────────────────────────────────

func (s *seeder) seedNotifications() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]

			// 2–6 notifications per family.
			numNotifs := 2 + s.rng.Intn(5)
			for j := 0; j < numNotifs; j++ {
				seq++
				nt := notifTemplates[s.rng.Intn(len(notifTemplates))]
				created := randomDate(s.rng, f.CreatedAt, s.now)
				isRead := s.rng.Float64() < 0.6

				rows = append(rows, []any{
					uid(dNotif, seq), f.ID, nt.Type, nt.Category,
					nt.Title, nt.Body, "", "{}", isRead, created,
				})
			}
		}

		// Extra notifications for seed family (rich inbox).
		for j := 0; j < 15; j++ {
			seq++
			nt := notifTemplates[s.rng.Intn(len(notifTemplates))]
			created := randomDate(s.rng, s.now.AddDate(0, -1, 0), s.now)
			rows = append(rows, []any{
				uid(dNotif, seq), existSeedFamilyID, nt.Type, nt.Category,
				nt.Title, nt.Body, "", "{}", j > 5, created,
			})
		}

		return execBatch(tx,
			"INSERT INTO notify_notifications (id, family_id, notification_type, category, title, body, action_url, metadata, is_read, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?::JSONB, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Billing ──────────────────────────────────────────────────────────────────

func (s *seeder) seedBilling() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var custRows [][]any
		var subRows [][]any
		var txnRows [][]any
		subSeq := 0
		txnSeq := 0

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if f.SubscriptionTier != "premium" {
				continue
			}
			subSeq++
			txnSeq++

			hsCustomerID := fmt.Sprintf("cus_full_%06d", subSeq)
			hsSubID := fmt.Sprintf("sub_full_%06d", subSeq)

			custRows = append(custRows, []any{
				f.ID, hsCustomerID, nil, f.CreatedAt, f.CreatedAt,
			})

			interval := "monthly"
			amount := 999
			if s.rng.Float64() < 0.3 {
				interval = "annual"
				amount = 9990
			}
			periodStart := s.now.AddDate(0, -1, 0)
			periodEnd := s.now.AddDate(0, 1, 0)

			hsPriceID := fmt.Sprintf("price_premium_%s_%06d", interval, subSeq)
			subRows = append(subRows, []any{
				uid(dSub, subSeq), f.ID, hsSubID, hsCustomerID,
				"premium", "active", interval,
				periodStart, periodEnd, false, nil,
				amount, "usd", hsPriceID, f.CreatedAt, f.CreatedAt,
			})

			txnRows = append(txnRows, []any{
				uid(dTxn, txnSeq), f.ID, "subscription_payment", "succeeded",
				amount, "usd", nil, nil, "Premium subscription payment", "{}",
				f.CreatedAt,
			})
		}

		if err := execBatch(tx,
			"INSERT INTO bill_hyperswitch_customers (family_id, hyperswitch_customer_id, default_payment_method_id, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?)",
			"ON CONFLICT (family_id) DO NOTHING",
			custRows,
		); err != nil {
			return fmt.Errorf("billing customers: %w", err)
		}

		if err := execBatch(tx,
			"INSERT INTO bill_subscriptions (id, family_id, hyperswitch_subscription_id, hyperswitch_customer_id, tier, status, billing_interval, current_period_start, current_period_end, cancel_at_period_end, canceled_at, amount_cents, currency, hyperswitch_price_id, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			subRows,
		); err != nil {
			return fmt.Errorf("billing subscriptions: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO bill_transactions (id, family_id, transaction_type, status, amount_cents, currency, hyperswitch_payment_id, hyperswitch_invoice_id, description, metadata, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?::JSONB, ?)",
			"ON CONFLICT DO NOTHING",
			txnRows,
		)
	})
}

// ─── Recommendations ──────────────────────────────────────────────────────────

func (s *seeder) seedRecommendations() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var signalRows [][]any
		var recRows [][]any
		var prefRows [][]any
		sigSeq := 0
		recSeq := 0

		signalTypes := []string{"activity_logged", "book_completed", "purchase_completed"}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if f.ActivityLevel < 2 {
				continue
			}

			// Signals: 2–5 per family.
			numSignals := f.ActivityLevel
			for j := 0; j < numSignals; j++ {
				sigSeq++
				sigType := signalTypes[s.rng.Intn(len(signalTypes))]
				sigDate := randomDate(s.rng, f.CreatedAt, s.now)

				var studentID any = nil
				if len(f.Students) > 0 {
					studentID = f.Students[s.rng.Intn(len(f.Students))].ID
				}

				payload, _ := json.Marshal(map[string]string{
					"type": sigType,
				})

				signalRows = append(signalRows, []any{
					uid(dRecsSignl, sigSeq), f.ID, studentID,
					sigType, f.PrimaryMethodology,
					string(payload), sigDate.Format("2006-01-02"), sigDate,
				})
			}

			// Recommendations: 1–3 per active family.
			numRecs := 1 + s.rng.Intn(min(3, f.ActivityLevel))
			for j := 0; j < numRecs; j++ {
				recSeq++
				recType := "marketplace_content"
				targetID := s.listingIDs[s.rng.Intn(len(s.listingIDs))]
				targetLabel := "Recommended listing"
				sourceSignal := "methodology_match"

				switch s.rng.Intn(4) {
				case 1:
					recType = "reading_suggestion"
					targetID = s.readingItemIDs[s.rng.Intn(len(s.readingItemIDs))]
					targetLabel = "Recommended book"
					sourceSignal = "reading_history"
				case 2:
					recType = "activity_idea"
					targetID = s.activityDefIDs[s.rng.Intn(len(s.activityDefIDs))]
					targetLabel = "Suggested activity"
					sourceSignal = "progress_gap"
				case 3:
					recType = "community_group"
					if len(s.groupIDs) > 0 {
						targetID = s.groupIDs[s.rng.Intn(len(s.groupIDs))]
					}
					targetLabel = "Suggested group"
					sourceSignal = "methodology_match"
				}

				score := 0.5 + s.rng.Float64()*0.5
				expires := s.now.AddDate(0, 0, 14)

				recRows = append(recRows, []any{
					uid(dRec, recSeq), f.ID, nil,
					recType, targetID, targetLabel,
					sourceSignal, methodologyDisplayNames[f.PrimaryMethodology],
					score, "active", expires, s.now, s.now,
				})
			}

			// Preferences for each family.
			prefRows = append(prefRows, []any{
				uid(dRecsPref, famIdx), f.ID,
				"{marketplace_content,reading_suggestion,activity_idea,community_group}",
				"occasional", f.CreatedAt, f.CreatedAt,
			})
		}

		if err := execBatch(tx,
			"INSERT INTO recs_signals (id, family_id, student_id, signal_type, methodology_slug, payload, signal_date, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?::JSONB, ?, ?)",
			"ON CONFLICT DO NOTHING",
			signalRows,
		); err != nil {
			return fmt.Errorf("recs signals: %w", err)
		}

		if err := execBatch(tx,
			"INSERT INTO recs_recommendations (id, family_id, student_id, recommendation_type, target_entity_id, target_entity_label, source_signal, source_label, score, status, expires_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			recRows,
		); err != nil {
			return fmt.Errorf("recs recommendations: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO recs_preferences (id, family_id, enabled_types, exploration_frequency, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			prefRows,
		)
	})
}
