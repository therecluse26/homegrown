package main

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ─── Notification Preferences ─────────────────────────────────────────────────

func (s *seeder) seedNotificationPreferences() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		type prefTemplate struct {
			notifType string
			channel   string
		}
		templates := []prefTemplate{
			{"message_received", "email"},
			{"milestone_achieved", "in_app"},
			{"activity_streak", "email"},
		}
		digestFreqs := []string{"immediate", "daily", "weekly"}

		var rows [][]any
		seq := 0
		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			for j, tmpl := range templates {
				seq++
				enabled := s.rng.Float64() > 0.2
				freq := digestFreqs[s.rng.Intn(len(digestFreqs))]
				// Higher-activity families prefer immediate delivery.
				if f.ActivityLevel >= 4 {
					freq = "immediate"
				}
				rows = append(rows, []any{
					uid(dNotifPref, seq), f.ID,
					tmpl.notifType, tmpl.channel,
					enabled, freq,
					// Use j to vary defaults per-type
					j,
				})
				// Trim the unused j arg — rows must match the INSERT columns exactly.
				rows[len(rows)-1] = rows[len(rows)-1][:6]
			}
		}

		return execBatch(tx,
			"INSERT INTO notify_preferences (id, family_id, notification_type, channel, enabled, digest_frequency) VALUES ",
			"(?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (family_id, notification_type, channel) DO NOTHING",
			rows,
		)
	})
}

// ─── Transcripts ──────────────────────────────────────────────────────────────

func (s *seeder) seedTranscripts() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		highSchoolGrades := map[string]bool{
			"9th": true, "10th": true, "11th": true, "12th": true,
		}

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			// Only families with at least one high-school-age student.
			var hsStudents []student
			for _, st := range f.Students {
				if highSchoolGrades[st.GradeLevel] {
					hsStudents = append(hsStudents, st)
				}
			}
			if len(hsStudents) == 0 {
				continue
			}
			// ~75% of eligible families get a transcript.
			if s.rng.Float64() > 0.75 {
				continue
			}

			st := hsStudents[0]
			seq++
			gradeLevel := st.GradeLevel
			rows = append(rows, []any{
				uid(dTranscr, seq), f.ID, st.ID,
				fmt.Sprintf("%s Academic Transcript", st.Name),
				st.Name,
				"{" + gradeLevel + "}",
				"configuring",
				f.CreatedAt, f.CreatedAt,
			})
		}

		return execBatch(tx,
			"INSERT INTO comply_transcripts (id, family_id, student_id, title, student_name, grade_levels, status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Portfolios ───────────────────────────────────────────────────────────────

func (s *seeder) seedPortfolios() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		schoolYearStart := "2025-08-15"
		schoolYearEnd := "2026-06-15"

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if f.ActivityLevel < 3 || len(f.Students) == 0 {
				continue
			}

			st := f.Students[0]
			seq++
			rows = append(rows, []any{
				uid(dPortfolio, seq), f.ID, st.ID,
				fmt.Sprintf("%s Portfolio 2025-2026", st.Name),
				"Academic portfolio for the 2025-2026 school year",
				"chronological",
				schoolYearStart, schoolYearEnd,
				true, true,
				"configuring",
				f.CreatedAt, f.CreatedAt,
			})
		}

		return execBatch(tx,
			"INSERT INTO comply_portfolios (id, family_id, student_id, title, description, organization, date_range_start, date_range_end, include_attendance, include_assessments, status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Standardized Tests ───────────────────────────────────────────────────────

func (s *seeder) seedStandardizedTests() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		testNames := []string{"Iowa Assessments", "CAT", "Stanford Achievement Test"}
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if s.rng.Float64() > 0.25 {
				continue
			}
			for _, st := range f.Students {
				seq++
				testName := testNames[s.rng.Intn(len(testNames))]
				testDate := randomDate(s.rng, schoolStart, s.now)
				gradeNum := gradeToNum(st.GradeLevel)
				reading := 75 + s.rng.Intn(21)
				math := 75 + s.rng.Intn(21)
				language := 75 + s.rng.Intn(21)
				composite := float64(reading+math+language) / 3.0
				percentile := 50 + s.rng.Intn(46)

				scores := fmt.Sprintf(`{"reading":%d,"math":%d,"language":%d}`, reading, math, language)

				rows = append(rows, []any{
					uid(dStdTest, seq), f.ID, st.ID,
					testName, testDate.Format("2006-01-02"),
					gradeNum, scores,
					composite, percentile,
					"",
					f.CreatedAt, f.CreatedAt,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO comply_standardized_tests (id, family_id, student_id, test_name, test_date, grade_level, scores, composite_score, percentile, notes, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?::JSONB, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Learning Progress (quiz, video, project, sequence) ───────────────────────

func (s *seeder) seedLearningProgress() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		// ── Quiz Sessions ──
		var quizRows [][]any
		quizSeq := 0
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		if len(s.quizDefIDs) > 0 {
			for famIdx := 3; famIdx < len(s.families); famIdx++ {
				f := s.families[famIdx]
				if f.ActivityLevel < 2 || len(f.Students) == 0 {
					continue
				}
				sessionsPerStudent := 1 + s.rng.Intn(3)
				for _, st := range f.Students {
					for range sessionsPerStudent {
						quizSeq++
						qID := s.quizDefIDs[s.rng.Intn(len(s.quizDefIDs))]
						startedAt := randomDate(s.rng, maxTime(schoolStart, f.CreatedAt), s.now)
						status := "scored"
						if s.rng.Float64() < 0.30 {
							status = "in_progress"
						}

						var score, maxScore any = nil, nil
						var passed any = nil
						var submittedAt, scoredAt any = nil, nil

						if status == "scored" {
							sc := 60 + s.rng.Intn(41)
							score = sc
							maxScore = 100
							passed = sc >= 70
							submittedAt = startedAt.Add(30 * time.Minute)
							scoredAt = startedAt.Add(31 * time.Minute)
						}

						quizRows = append(quizRows, []any{
							uid(dQuizSess, quizSeq), f.ID, st.ID,
							qID, status, startedAt,
							submittedAt, scoredAt,
							score, maxScore, passed,
							"[]",
							nil, // scored_by
							startedAt, startedAt,
						})
					}
				}
			}
		}

		if err := execBatch(tx,
			"INSERT INTO learn_quiz_sessions (id, family_id, student_id, quiz_def_id, status, started_at, submitted_at, scored_at, score, max_score, passed, answers, scored_by, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::JSONB, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			quizRows,
		); err != nil {
			return fmt.Errorf("quiz sessions: %w", err)
		}

		// ── Video Progress ──
		var videoRows [][]any
		videoSeq := 0

		if len(s.videoDefIDs) > 0 {
			for famIdx := 3; famIdx < len(s.families); famIdx++ {
				f := s.families[famIdx]
				if f.ActivityLevel < 2 || len(f.Students) == 0 {
					continue
				}
				seen := map[string]bool{}
				videosPerStudent := 1 + s.rng.Intn(4)
				for _, st := range f.Students {
					attempts := 0
					added := 0
					for added < videosPerStudent && attempts < videosPerStudent*5 {
						attempts++
						vID := s.videoDefIDs[s.rng.Intn(len(s.videoDefIDs))]
						key := st.ID + "|" + vID
						if seen[key] {
							continue
						}
						seen[key] = true
						added++
						videoSeq++

						watched := 60 + s.rng.Intn(541)
						completed := s.rng.Float64() < 0.60
						var completedAt any = nil
						if completed {
							completedAt = s.now.Add(-time.Duration(s.rng.Intn(30*24)) * time.Hour)
						}

						videoRows = append(videoRows, []any{
							uid(dVideoProg, videoSeq), f.ID, st.ID,
							vID, watched, completed,
							watched, // last_position_seconds
							completedAt,
							s.now, s.now,
						})
					}
				}
			}
		}

		if err := execBatch(tx,
			"INSERT INTO learn_video_progress (id, family_id, student_id, video_def_id, watched_seconds, completed, last_position_seconds, completed_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (family_id, student_id, video_def_id) DO NOTHING",
			videoRows,
		); err != nil {
			return fmt.Errorf("video progress: %w", err)
		}

		// ── Project Progress ──
		var projRows [][]any
		projSeq := 0
		projStatuses := []string{"planning", "in_progress", "completed"}

		if len(s.projectDefIDs) > 0 {
			for famIdx := 3; famIdx < len(s.families); famIdx++ {
				f := s.families[famIdx]
				if f.ActivityLevel < 3 || len(f.Students) == 0 {
					continue
				}
				projPerStudent := 1 + s.rng.Intn(2)
				for _, st := range f.Students {
					usedProjs := pickN(s.rng, len(s.projectDefIDs), min(projPerStudent, len(s.projectDefIDs)))
					for _, pIdx := range usedProjs {
						projSeq++
						status := projStatuses[s.rng.Intn(len(projStatuses))]
						startedAt := randomDate(s.rng, maxTime(schoolStart, f.CreatedAt), s.now)
						var completedAt any = nil
						if status == "completed" {
							completedAt = randomDate(s.rng, startedAt, s.now)
						}

						projRows = append(projRows, []any{
							uid(dProjProg, projSeq), f.ID, st.ID,
							s.projectDefIDs[pIdx], status,
							"[]", // milestones
							startedAt, completedAt,
							"",   // notes
							"[]", // attachments
							startedAt, startedAt,
						})
					}
				}
			}
		}

		if err := execBatch(tx,
			"INSERT INTO learn_project_progress (id, family_id, student_id, project_def_id, status, milestones, started_at, completed_at, notes, attachments, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?::JSONB, ?, ?, ?, ?::JSONB, ?, ?)",
			"ON CONFLICT DO NOTHING",
			projRows,
		); err != nil {
			return fmt.Errorf("project progress: %w", err)
		}

		// ── Sequence Progress ──
		var seqRows [][]any
		seqSeq := 0

		if len(s.sequenceDefIDs) > 0 {
			for famIdx := 3; famIdx < len(s.families); famIdx++ {
				f := s.families[famIdx]
				if f.ActivityLevel < 4 || len(f.Students) == 0 {
					continue
				}
				seen := map[string]bool{}
				for _, st := range f.Students {
					sID := s.sequenceDefIDs[s.rng.Intn(len(s.sequenceDefIDs))]
					key := st.ID + "|" + sID
					if seen[key] {
						continue
					}
					seen[key] = true
					seqSeq++

					status := "in_progress"
					if s.rng.Float64() < 0.25 {
						status = "completed"
					}
					startedAt := randomDate(s.rng, maxTime(schoolStart, f.CreatedAt), s.now)
					var completedAt any = nil
					if status == "completed" {
						completedAt = randomDate(s.rng, startedAt, s.now)
					}

					seqRows = append(seqRows, []any{
						uid(dSeqProg, seqSeq), f.ID, st.ID,
						sID, 0, // current_item_index
						status,
						"[]", // item_completions
						startedAt, completedAt,
						startedAt, startedAt,
					})
				}
			}
		}

		return execBatch(tx,
			"INSERT INTO learn_sequence_progress (id, family_id, student_id, sequence_def_id, current_item_index, status, item_completions, started_at, completed_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?::JSONB, ?, ?, ?, ?)",
			"ON CONFLICT (family_id, student_id, sequence_def_id) DO NOTHING",
			seqRows,
		)
	})
}

// ─── Grading Scales ───────────────────────────────────────────────────────────

func (s *seeder) seedGradingScales() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		standardGrades := `[{"letter":"A","min_percent":90,"points":4.0},{"letter":"B","min_percent":80,"points":3.0},{"letter":"C","min_percent":70,"points":2.0},{"letter":"D","min_percent":60,"points":1.0},{"letter":"F","min_percent":0,"points":0.0}]`

		var rows [][]any
		for i, f := range s.families[3:] {
			rows = append(rows, []any{
				uid(dGradScale, i+1), f.ID,
				"Standard Letter Scale", "letter",
				standardGrades,
				true,
				f.CreatedAt, f.CreatedAt,
			})
		}

		return execBatch(tx,
			"INSERT INTO learn_grading_scales (id, family_id, name, scale_type, grades, is_default, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?::JSONB, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Schedule Templates ───────────────────────────────────────────────────────

func (s *seeder) seedScheduleTemplates() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		templateNames := map[string]string{
			"charlotte-mason": "Charlotte Mason Morning Routine",
			"classical":       "Classical Trivium Schedule",
			"traditional":     "Traditional School Day Template",
			"montessori":      "Montessori Work Period Template",
			"waldorf":         "Waldorf Rhythm Template",
			"unschooling":     "Child-Led Daily Flow",
		}

		var rows [][]any
		seq := 0
		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if f.ActivityLevel < 2 {
				continue
			}
			seq++
			name, ok := templateNames[f.PrimaryMethodology]
			if !ok {
				name = "Daily Schedule Template"
			}

			rows = append(rows, []any{
				uid(dSchedTmpl, seq), f.ID,
				name,
				fmt.Sprintf("Weekly schedule template for the %s family", f.DisplayName),
				"[]",
				true,
				f.CreatedAt, f.CreatedAt,
			})
		}

		return execBatch(tx,
			"INSERT INTO plan_schedule_templates (id, family_id, name, description, items, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?::JSONB, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Media Uploads ────────────────────────────────────────────────────────────

func (s *seeder) seedMediaUploads() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0
		for i, f := range s.families[3:] {
			seq++
			storageKey := fmt.Sprintf("seed/profile-photos/%s.jpg", f.ID)
			rows = append(rows, []any{
				uid(dMediaUp, seq), f.ID, f.PrimaryParentID,
				"profile_photo", "published",
				fmt.Sprintf("avatar_%d.jpg", i+1),
				"image/jpeg", 45000,
				storageKey,
				false, false, // has_thumb, has_medium
				nil,   // probe_metadata
				45000, // original_size_bytes
				false, // was_compressed
				nil, nil, // moderation_labels, last_csam_scanned_at
				nil,      // expires_at
				f.CreatedAt,
				f.CreatedAt, f.CreatedAt,
			})
		}

		return execBatch(tx,
			"INSERT INTO media_uploads (id, family_id, uploaded_by, context, status, original_filename, content_type, size_bytes, storage_key, has_thumb, has_medium, probe_metadata, original_size_bytes, was_compressed, moderation_labels, last_csam_scanned_at, expires_at, published_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (storage_key) DO NOTHING",
			rows,
		)
	})
}

// ─── Starter Recommendations ─────────────────────────────────────────────────

func (s *seeder) seedStarterRecommendations() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		type recTemplate struct {
			title       string
			description string
			priceType   string
			ageGroup    string
		}

		methodRecs := map[string][]recTemplate{
			"charlotte-mason": {
				{"A Charlotte Mason Companion", "The definitive guide to Charlotte Mason's philosophy and methods.", "paid", "all"},
				{"Ambleside Online Curriculum", "Free Charlotte Mason curriculum organised by year.", "free", "all"},
				{"Simply Charlotte Mason", "Practical CM resources for every subject.", "paid", "elementary"},
			},
			"classical": {
				{"The Well-Trained Mind", "Susan Wise Bauer's classical education guide.", "paid", "all"},
				{"Classical Conversations", "Community-based classical education program.", "paid", "elementary"},
				{"Memoria Press Curriculum", "Traditional classical curriculum with strong Latin emphasis.", "paid", "middle"},
			},
			"traditional": {
				{"Abeka Curriculum", "Traditional Christian school curriculum used by thousands of homeschoolers.", "paid", "all"},
				{"Horizons Math", "Spiral approach math curriculum with bright visuals.", "paid", "elementary"},
				{"Saxon Math", "Incremental, rigorous math program used widely by homeschoolers.", "paid", "middle"},
			},
			"montessori": {
				{"Montessori at Home", "Practical Montessori activities for home educators.", "free", "preschool"},
				{"Keys of the World", "Montessori-aligned geography and cultural studies.", "paid", "elementary"},
				{"Waseca Biomes", "Beautiful Montessori geography and biology materials.", "paid", "elementary"},
			},
			"waldorf": {
				{"Live Education Curriculum", "Waldorf-inspired homeschool curriculum.", "paid", "elementary"},
				{"Earthschooling", "Online Waldorf homeschool curriculum.", "paid", "all"},
				{"Christopherus Homeschool Resources", "Waldorf curriculum resources for home educators.", "paid", "elementary"},
			},
			"unschooling": {
				{"Free to Learn by Peter Gray", "Research-backed case for self-directed education.", "paid", "all"},
				{"The Unschooling Handbook", "Practical guide to interest-led learning.", "paid", "all"},
				{"Bored and Brilliant", "How doing nothing leads to our most creative ideas.", "free", "teen"},
			},
		}

		defaultRecs := []recTemplate{
			{"Homeschool Legal Defense Association", "Legal support and resources for homeschooling families.", "paid", "all"},
			{"Khan Academy", "Free world-class education for anyone, anywhere.", "free", "all"},
			{"Time4Learning", "Online curriculum for PreK-12 homeschoolers.", "paid", "all"},
		}

		priceTypes := []string{"free", "paid"}

		var rows [][]any
		seq := 0
		for i, f := range s.families[3:] {
			wizardID := uid(dWizard, i+1)
			recs, ok := methodRecs[f.PrimaryMethodology]
			if !ok {
				recs = defaultRecs
			}

			for order, rec := range recs {
				seq++
				pt := rec.priceType
				if pt == "" {
					pt = priceTypes[s.rng.Intn(len(priceTypes))]
				}
				rows = append(rows, []any{
					uid(dStarterRec, seq), f.ID, wizardID,
					f.PrimaryMethodology,
					rec.title, rec.description,
					nil, // link_url
					pt, rec.ageGroup,
					order,
					f.CreatedAt,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO onb_starter_recommendations (id, family_id, wizard_id, methodology_slug, title, description, link_url, price_type, age_group, sort_order, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}
