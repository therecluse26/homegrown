package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ─── Publishers & Creators ────────────────────────────────────────────────────

func (s *seeder) seedPublishers() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.publisherIDs = []string{existSeedPublisher} // include basic seeder's publisher
		s.creatorIDs = nil
		s.creatorParentIDs = nil

		// Publishers
		var pubRows [][]any
		for i, p := range publisherNames {
			pID := uid(dPublisher, i+1)
			s.publisherIDs = append(s.publisherIDs, pID)
			pubRows = append(pubRows, []any{
				pID, p.Name, p.Slug, p.Desc, false, true,
				s.now.AddDate(-1, 0, 0), s.now.AddDate(-1, 0, 0),
			})
		}
		if err := execBatch(tx,
			"INSERT INTO mkt_publishers (id, name, slug, description, is_platform, is_verified, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			pubRows,
		); err != nil {
			return fmt.Errorf("publishers: %w", err)
		}

		// Creators — pick parents from families to be marketplace creators.
		var creatorRows [][]any
		var memberRows [][]any
		for i := 0; i < numCreators; i++ {
			famIdx := 3 + i*28 // spread across families
			if famIdx >= len(s.families) {
				famIdx = 3 + (i % (len(s.families) - 3))
			}
			f := s.families[famIdx]
			cID := uid(dCreator, i+1)
			s.creatorIDs = append(s.creatorIDs, cID)
			s.creatorParentIDs = append(s.creatorParentIDs, f.PrimaryParentID)

			storeName := f.PrimaryParentName + "'s Homeschool Shop"
			bio := "Homeschool parent sharing resources from our " + methodologyDisplayNames[f.PrimaryMethodology] + " journey."

			creatorRows = append(creatorRows, []any{
				cID, f.PrimaryParentID, "active", storeName, bio,
				f.CreatedAt, f.CreatedAt, f.CreatedAt,
			})

			// Assign creator to a publisher.
			pubIdx := i % len(s.publisherIDs)
			memberRows = append(memberRows, []any{
				uid(dPubMember, i+1), s.publisherIDs[pubIdx], cID,
				"member", f.CreatedAt,
			})
		}

		if err := execBatch(tx,
			"INSERT INTO mkt_creators (id, parent_id, onboarding_status, store_name, store_bio, tos_accepted_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			creatorRows,
		); err != nil {
			return fmt.Errorf("creators: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO mkt_publisher_members (id, publisher_id, creator_id, role, created_at) VALUES ",
			"(?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			memberRows,
		)
	})
}

// ─── Published Learning Content ───────────────────────────────────────────────

func (s *seeder) seedPublishedContent() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		now := s.now

		// ── Activity definitions ──
		s.activityDefIDs = make([]string, 0, numActivityDefs)
		var actRows [][]any
		for i := 0; i < numActivityDefs; i++ {
			at := activityTemplates[i%len(activityTemplates)]
			aID := uid(dActDef, i+1)
			s.activityDefIDs = append(s.activityDefIDs, aID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]

			title := at.Title
			if i >= len(activityTemplates) {
				title = fmt.Sprintf("%s — Variation %d", at.Title, i/len(activityTemplates)+1)
			}

			// Assign methodology: round-robin across slugs so every methodology has content.
			methSlug := methodologySlugs[i%len(methodologySlugs)]
			methID := methodologyUUIDs[methSlug]

			actRows = append(actRows, []any{
				aID, pubID, title, at.Description,
				"{" + strings.Join(at.Subjects, ",") + "}",
				methID, at.Duration, true, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO learn_activity_defs (id, publisher_id, title, description, subject_tags, methodology_id, est_duration_minutes, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			actRows,
		); err != nil {
			return fmt.Errorf("activity defs: %w", err)
		}

		// ── Reading items (books) ──
		s.readingItemIDs = make([]string, 0, numReadingItems)
		var readRows [][]any
		for i := 0; i < numReadingItems; i++ {
			bt := bookCatalog[i%len(bookCatalog)]
			rID := uid(dReadItem, i+1)
			s.readingItemIDs = append(s.readingItemIDs, rID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]

			readRows = append(readRows, []any{
				rID, pubID, bt.Title, bt.Author,
				"{" + strings.Join(bt.Subjects, ",") + "}",
				bt.Title + " — a beloved book for homeschool families.",
				bt.PageCount, true, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO learn_reading_items (id, publisher_id, title, author, subject_tags, description, page_count, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			readRows,
		); err != nil {
			return fmt.Errorf("reading items: %w", err)
		}

		// ── Assessment definitions ──
		s.assessmentDefIDs = make([]string, 0, numAssessmentDefs)
		var assRows [][]any
		assessTitles := []string{
			"Reading Comprehension", "Math Facts Mastery", "Spelling Test",
			"History Narration", "Science Vocabulary", "Latin Declension Quiz",
			"Geography Map Quiz", "Grammar Assessment", "Writing Evaluation",
			"Nature Identification", "Music Theory Check", "Art Appreciation",
			"Phonics Assessment", "Multiplication Tables", "Division Facts",
			"Fraction Concepts", "Poetry Recitation", "Timeline Accuracy",
			"Handwriting Evaluation", "Oral Narration", "Lab Report Quality",
			"Research Skills", "Presentation Skills", "Critical Thinking",
			"Problem Solving", "Reading Fluency", "Vocabulary Building",
			"Essay Writing", "Book Report", "Creative Writing Portfolio",
		}
		for i := 0; i < numAssessmentDefs; i++ {
			aID := uid(dAssDef, i+1)
			s.assessmentDefIDs = append(s.assessmentDefIDs, aID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]
			title := assessTitles[i%len(assessTitles)]
			subjects := []string{subjectTags[s.rng.Intn(len(subjectTags))]}
			scoringType := "percentage"
			if s.rng.Float64() < 0.3 {
				scoringType = "letter"
			}
			assRows = append(assRows, []any{
				aID, pubID, title, "Assessment: " + title,
				"{" + strings.Join(subjects, ",") + "}",
				scoringType, 100, true, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO learn_assessment_defs (id, publisher_id, title, description, subject_tags, scoring_type, max_score, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			assRows,
		); err != nil {
			return fmt.Errorf("assessment defs: %w", err)
		}

		// ── Quiz definitions with questions ──
		s.quizDefIDs = make([]string, 0, numQuizDefs)
		var quizRows [][]any
		var questionRows [][]any
		var qqRows [][]any // quiz_questions join
		qSeq := 0

		quizTitles := []string{
			"Reading Comprehension Quiz", "Math Facts Speed Round", "History Timeline Quiz",
			"Science Vocabulary Check", "Latin Word Roots", "Geography Capitals",
			"Grammar Fundamentals", "Nature Identification", "Music Theory Basics",
			"Spelling Challenge", "Poetry Recognition", "Map Skills",
			"Fraction Fundamentals", "Astronomy Basics", "World Cultures",
		}

		for i := 0; i < numQuizDefs; i++ {
			qID := uid(dQuizDef, i+1)
			s.quizDefIDs = append(s.quizDefIDs, qID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]
			title := quizTitles[i%len(quizTitles)]
			subjects := []string{subjectTags[s.rng.Intn(len(subjectTags))]}
			methSlug := methodologySlugs[i%len(methodologySlugs)]
			methID := methodologyUUIDs[methSlug]

			quizRows = append(quizRows, []any{
				qID, pubID, title, "Quiz: " + title,
				"{" + strings.Join(subjects, ",") + "}",
				methID, 15, 70, false, true, numQuestionsPerQuiz, true, now, now,
			})

			// Questions for this quiz.
			for j := 0; j < numQuestionsPerQuiz; j++ {
				qSeq++
				questID := uid(dQuestion, qSeq)
				var qType, answerData string
				switch j % 3 {
				case 1:
					qType = "true_false"
					answerData = `{"correct":true}`
				case 2:
					qType = "short_answer"
					answerData = `{"sample_answer":"Sample answer text"}`
				default:
					qType = "multiple_choice"
					answerData = `{"correct":"a","options":["a","b","c","d"]}`
				}

				questionRows = append(questionRows, []any{
					questID, pubID, qType,
					fmt.Sprintf("Question %d for %s", j+1, title),
					answerData,
					"{" + strings.Join(subjects, ",") + "}",
					methID, // inherit methodology from parent quiz
					1 + s.rng.Intn(5), // difficulty 1–5
					qType != "short_answer",
					10, true, now, now,
				})

				qqRows = append(qqRows, []any{
					qID, questID, j + 1, nil,
				})
			}
		}

		if err := execBatch(tx,
			"INSERT INTO learn_quiz_defs (id, publisher_id, title, description, subject_tags, methodology_id, time_limit_minutes, passing_score_percent, shuffle_questions, show_correct_after, question_count, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			quizRows,
		); err != nil {
			return fmt.Errorf("quiz defs: %w", err)
		}

		if err := execBatch(tx,
			"INSERT INTO learn_questions (id, publisher_id, question_type, content, answer_data, subject_tags, methodology_id, difficulty_level, auto_scorable, points, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?::JSONB, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			questionRows,
		); err != nil {
			return fmt.Errorf("questions: %w", err)
		}

		if err := execBatch(tx,
			"INSERT INTO learn_quiz_questions (quiz_def_id, question_id, sort_order, points_override) VALUES ",
			"(?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			qqRows,
		); err != nil {
			return fmt.Errorf("quiz_questions: %w", err)
		}

		// ── Video definitions ──
		s.videoDefIDs = make([]string, 0, numVideoDefs)
		var vidRows [][]any
		videoTitles := []string{
			"Introduction to Nature Journaling", "Watercolor Basics for Kids",
			"History: Ancient Egypt", "Math: Understanding Fractions",
			"Science: The Water Cycle", "Latin Pronunciation Guide",
			"Geography: The Continents", "Art: Drawing Animals",
			"Music: Reading Sheet Music", "Poetry: How to Memorize",
			"Handwriting: Cursive Basics", "Science: Simple Machines",
			"History: Medieval Life", "Math: Multiplication Strategies",
			"Nature: Bird Identification", "Art: Perspective Drawing",
			"Science: Plant Life Cycle", "Geography: Map Reading Skills",
			"History: American Revolution", "Math: Geometry Basics",
			"Science: Human Body Systems", "Music: Rhythm & Tempo",
			"Art: Color Theory", "History: Ancient Greece",
			"Science: Electricity Basics",
		}
		for i := 0; i < numVideoDefs; i++ {
			vID := uid(dVideoDef, i+1)
			s.videoDefIDs = append(s.videoDefIDs, vID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]
			title := videoTitles[i%len(videoTitles)]
			subjects := []string{subjectTags[s.rng.Intn(len(subjectTags))]}
			duration := 300 + s.rng.Intn(1500) // 5–30 minutes
			methSlug := methodologySlugs[i%len(methodologySlugs)]
			methID := methodologyUUIDs[methSlug]

			vidRows = append(vidRows, []any{
				vID, pubID, title, "Video lesson: " + title,
				"{" + strings.Join(subjects, ",") + "}",
				methID, duration, "", "youtube", "", true, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO learn_video_defs (id, publisher_id, title, description, subject_tags, methodology_id, duration_seconds, video_url, video_source, external_video_id, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			vidRows,
		); err != nil {
			return fmt.Errorf("video defs: %w", err)
		}

		// ── Project definitions ──
		s.projectDefIDs = make([]string, 0, numProjectDefs)
		var projRows [][]any
		projectTitles := []string{
			"Build a Bird Feeder", "Create a Family Timeline",
			"Design a Medieval Castle", "Plant a Container Garden",
			"Write & Illustrate a Story", "Build a Simple Machine",
			"Create a Weather Station", "Make a Solar System Model",
			"Sew a Stuffed Animal", "Build a Catapult",
			"Create a Culture Box", "Design an Invention",
		}
		for i := 0; i < numProjectDefs; i++ {
			pID := uid(dProjDef, i+1)
			s.projectDefIDs = append(s.projectDefIDs, pID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]
			title := projectTitles[i%len(projectTitles)]
			subjects := []string{subjectTags[s.rng.Intn(len(subjectTags))]}

			milestones, _ := json.Marshal([]map[string]string{
				{"title": "Planning", "description": "Research and plan the project"},
				{"title": "Materials", "description": "Gather all needed materials"},
				{"title": "Building", "description": "Build or create the project"},
				{"title": "Presentation", "description": "Present or display the finished work"},
			})

			projRows = append(projRows, []any{
				pID, pubID, title, "Hands-on project: " + title,
				"{" + strings.Join(subjects, ",") + "}",
				string(milestones), true, now, now,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO learn_project_defs (id, publisher_id, title, description, subject_tags, milestone_templates, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?::JSONB, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			projRows,
		); err != nil {
			return fmt.Errorf("project defs: %w", err)
		}

		// ── Sequence definitions ──
		s.sequenceDefIDs = make([]string, 0, numSequenceDefs)
		var seqDefRows [][]any
		var seqItemRows [][]any
		siSeq := 0

		seqTitles := []string{
			"Charlotte Mason Morning Routine", "Classical Math Sequence",
			"Nature Study: Four Seasons", "History: Ancient Civilizations",
			"Reading: Fantasy Classics", "Science: Life Sciences",
			"Art Appreciation: Masters", "Latin: Year One",
		}
		for i := 0; i < numSequenceDefs; i++ {
			sID := uid(dSeqDef, i+1)
			s.sequenceDefIDs = append(s.sequenceDefIDs, sID)
			pubID := s.publisherIDs[s.rng.Intn(len(s.publisherIDs))]
			title := seqTitles[i%len(seqTitles)]
			subjects := []string{subjectTags[s.rng.Intn(len(subjectTags))]}
			methSlug := methodologySlugs[i%len(methodologySlugs)]
			methID := methodologyUUIDs[methSlug]

			seqDefRows = append(seqDefRows, []any{
				sID, pubID, title, "Learning sequence: " + title,
				"{" + strings.Join(subjects, ",") + "}",
				methID, true, true, now, now,
			})

			// 3–5 items per sequence mixing content types.
			numItems := 3 + s.rng.Intn(3)
			for j := 0; j < numItems; j++ {
				siSeq++
				contentType := "activity_def"
				contentID := s.activityDefIDs[s.rng.Intn(len(s.activityDefIDs))]
				switch j % 3 {
				case 1:
					contentType = "reading_item"
					contentID = s.readingItemIDs[s.rng.Intn(len(s.readingItemIDs))]
				case 2:
					contentType = "quiz_def"
					contentID = s.quizDefIDs[s.rng.Intn(len(s.quizDefIDs))]
				}
				seqItemRows = append(seqItemRows, []any{
					uid(dSeqItem, siSeq), sID, j + 1, contentType, contentID,
					true, j > 0, now,
				})
			}
		}

		if err := execBatch(tx,
			"INSERT INTO learn_sequence_defs (id, publisher_id, title, description, subject_tags, methodology_id, is_linear, is_active, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			seqDefRows,
		); err != nil {
			return fmt.Errorf("sequence defs: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO learn_sequence_items (id, sequence_def_id, sort_order, content_type, content_id, is_required, unlock_after_previous, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			seqItemRows,
		)
	})
}

// ─── Marketplace Listings ─────────────────────────────────────────────────────

func (s *seeder) seedListings() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.listingIDs = make([]string, 0, numListings)
		s.listingCreators = make([]string, 0, numListings)
		s.listingPrices = make([]int, 0, numListings)

		var rows [][]any
		for i := 0; i < numListings; i++ {
			lt := listingTemplates[i%len(listingTemplates)]
			lID := uid(dListing, i+1)
			s.listingIDs = append(s.listingIDs, lID)

			creatorIdx := i % len(s.creatorIDs)
			creatorID := s.creatorIDs[creatorIdx]
			s.listingCreators = append(s.listingCreators, creatorID)
			s.listingPrices = append(s.listingPrices, lt.PriceCents)

			pubIdx := creatorIdx % len(s.publisherIDs)
			pubID := s.publisherIDs[pubIdx]

			title := lt.Title
			if i >= len(listingTemplates) {
				title = fmt.Sprintf("%s — Edition %d", lt.Title, i/len(listingTemplates)+1)
			}

			// Rating varies by listing.
			ratingAvg := 3.5 + s.rng.Float64()*1.5 // 3.5–5.0
			ratingCount := s.rng.Intn(50)

			created := randomDate(s.rng, s.now.AddDate(-1, 0, 0), s.now.AddDate(0, -1, 0))

			// Tag each listing with 1–2 methodology UUIDs.
			numMethTags := 1 + s.rng.Intn(2)
			methPicks := pickN(s.rng, len(methodologySlugs), numMethTags)
			methUUIDs := make([]string, len(methPicks))
			for mi, idx := range methPicks {
				methUUIDs[mi] = methodologyUUIDs[methodologySlugs[idx]]
			}
			methTagArr := "{" + strings.Join(methUUIDs, ",") + "}"

			rows = append(rows, []any{
				lID, creatorID, pubID, title, lt.Description,
				lt.PriceCents,
				methTagArr,
				"{" + strings.Join(lt.Subjects, ",") + "}",
				lt.GradeMin, lt.GradeMax, lt.ContentType,
				"published", ratingAvg, ratingCount, 1,
				created, created, created,
			})
		}

		return execBatch(tx,
			"INSERT INTO mkt_listings (id, creator_id, publisher_id, title, description, price_cents, methodology_tags, subject_tags, grade_min, grade_max, content_type, status, rating_avg, rating_count, version, published_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?::UUID[], ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Purchases ────────────────────────────────────────────────────────────────

func (s *seeder) seedPurchases() error {
	s.purchaseListingMap = make(map[string]string)
	s.purchaseFamilyMap = make(map[string]string)

	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		// Track (family, listing) pairs to avoid unique constraint violations.
		type flPair struct{ f, l string }
		seen := make(map[flPair]bool)

		// ~30% of families make 1–5 purchases.
		for famIdx := 3; famIdx < len(s.families); famIdx++ {
			f := s.families[famIdx]
			if s.rng.Float64() > 0.30 {
				continue
			}

			numPurchases := 1 + s.rng.Intn(5)
			purchasedListings := pickN(s.rng, len(s.listingIDs), min(numPurchases, len(s.listingIDs)))

			for _, lIdx := range purchasedListings {
				// Skip if the creator is this family's parent.
				creatorID := s.listingCreators[lIdx]
				isOwnListing := false
				for _, cpID := range s.creatorParentIDs {
					if cpID == f.PrimaryParentID {
						isOwnListing = true
						break
					}
				}
				if isOwnListing {
					continue
				}

				pair := flPair{f.ID, s.listingIDs[lIdx]}
				if seen[pair] {
					continue
				}
				seen[pair] = true

				seq++
				purchaseID := uid(dPurchase, seq)
				s.purchaseListingMap[purchaseID] = s.listingIDs[lIdx]
				s.purchaseFamilyMap[purchaseID] = f.ID

				price := s.listingPrices[lIdx]
				platformFee := price * 15 / 100
				creatorPayout := price - platformFee
				created := randomDate(s.rng, f.CreatedAt, s.now)

				rows = append(rows, []any{
					purchaseID, f.ID, s.listingIDs[lIdx], creatorID,
					price, platformFee, creatorPayout, created,
				})
			}
		}

		// Seed family buys 10 listings for a rich library.
		for _, lIdx := range pickN(s.rng, len(s.listingIDs), min(10, len(s.listingIDs))) {
			pair := flPair{existSeedFamilyID, s.listingIDs[lIdx]}
			if seen[pair] {
				continue
			}
			seen[pair] = true

			seq++
			purchaseID := uid(dPurchase, seq)
			s.purchaseListingMap[purchaseID] = s.listingIDs[lIdx]
			s.purchaseFamilyMap[purchaseID] = existSeedFamilyID

			price := s.listingPrices[lIdx]
			rows = append(rows, []any{
				purchaseID, existSeedFamilyID, s.listingIDs[lIdx],
				s.listingCreators[lIdx], price, price * 15 / 100, price * 85 / 100,
				s.now.AddDate(0, -2, -s.rng.Intn(60)),
			})
		}

		return execBatch(tx,
			"INSERT INTO mkt_purchases (id, family_id, listing_id, creator_id, amount_cents, platform_fee_cents, creator_payout_cents, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Reviews ──────────────────────────────────────────────────────────────────

func (s *seeder) seedReviews() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		// Reviews require a real purchase. Iterate purchases and review ~40%.
		for purchaseID, listingID := range s.purchaseListingMap {
			if s.rng.Float64() > 0.40 {
				continue
			}
			seq++
			familyID := s.purchaseFamilyMap[purchaseID]
			rt := reviewTemplates[s.rng.Intn(len(reviewTemplates))]
			created := randomDate(s.rng, s.now.AddDate(0, -6, 0), s.now)

			rows = append(rows, []any{
				uid(dReview, seq), listingID, purchaseID,
				familyID, rt.Rating, rt.Text,
				false, "approved", created, created,
			})
		}

		return execBatch(tx,
			"INSERT INTO mkt_reviews (id, listing_id, purchase_id, family_id, rating, review_text, is_anonymous, moderation_status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}
