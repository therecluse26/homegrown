package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ─── IAM: Families, Parents, Students ─────────────────────────────────────────

func (s *seeder) seedIAM() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		// ── Families (skip first 3 — they exist from basic seeder) ──
		var famRows [][]any
		for _, f := range s.families[3:] {
			// ~40% of families have 1–2 secondary methodologies.
			var secondarySlugs string
			if s.rng.Float64() < 0.40 {
				candidates := make([]string, 0, len(methodologySlugs)-1)
				for _, slug := range methodologySlugs {
					if slug != f.PrimaryMethodology {
						candidates = append(candidates, slug)
					}
				}
				n := 1 + s.rng.Intn(2) // 1–2 secondary
				picks := pickN(s.rng, len(candidates), min(n, len(candidates)))
				picked := make([]string, len(picks))
				for i, idx := range picks {
					picked[i] = candidates[idx]
				}
				secondarySlugs = "{" + strings.Join(picked, ",") + "}"
			} else {
				secondarySlugs = "{}"
			}

			famRows = append(famRows, []any{
				f.ID, f.DisplayName, f.StateCode, f.PrimaryMethodology,
				secondarySlugs,
				f.SubscriptionTier, "consented", f.CreatedAt, f.CreatedAt,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO iam_families (id, display_name, state_code, primary_methodology_slug, secondary_methodology_slugs, subscription_tier, coppa_consent_status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			famRows,
		); err != nil {
			return fmt.Errorf("families: %w", err)
		}

		// ── Parents ──
		var parentRows [][]any
		for _, f := range s.families[3:] {
			// Kratos identity ID = deterministic fallback (no real Kratos for generated parents).
			kratosID := uid(dParent, 9000000+len(parentRows))
			parentRows = append(parentRows, []any{
				f.PrimaryParentID, f.ID, kratosID, f.PrimaryParentName,
				f.PrimaryParentEmail, true, false, f.CreatedAt, f.CreatedAt,
			})
			if f.CoParentID != "" {
				cpKratos := uid(dParent, 9500000+len(parentRows))
				cpEmail := strings.Replace(f.PrimaryParentEmail, "@", ".co@", 1)
				parentRows = append(parentRows, []any{
					f.CoParentID, f.ID, cpKratos, f.CoParentName,
					cpEmail, false, false, f.CreatedAt, f.CreatedAt,
				})
			}
		}
		if err := execBatch(tx,
			"INSERT INTO iam_parents (id, family_id, kratos_identity_id, display_name, email, is_primary, is_platform_admin, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			parentRows,
		); err != nil {
			return fmt.Errorf("parents: %w", err)
		}

		// ── Students (~15% get a methodology override different from family primary) ──
		var studentRows [][]any
		for _, f := range s.families[3:] {
			for _, st := range f.Students {
				var override any = nil
				if s.rng.Float64() < 0.15 {
					// Pick a methodology different from the family's primary.
					for {
						slug := methodologySlugs[s.rng.Intn(len(methodologySlugs))]
						if slug != f.PrimaryMethodology {
							override = slug
							break
						}
					}
				}
				studentRows = append(studentRows, []any{
					st.ID, f.ID, st.Name, st.BirthYear, st.GradeLevel,
					override,
					f.CreatedAt, f.CreatedAt,
				})
			}
		}
		if err := execBatch(tx,
			"INSERT INTO iam_students (id, family_id, display_name, birth_year, grade_level, methodology_override_slug, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			studentRows,
		); err != nil {
			return fmt.Errorf("students: %w", err)
		}

		// ── Update primary_parent_id ──
		for _, f := range s.families[3:] {
			if err := tx.Exec(
				"UPDATE iam_families SET primary_parent_id = ? WHERE id = ? AND primary_parent_id IS NULL",
				f.PrimaryParentID, f.ID,
			).Error; err != nil {
				return fmt.Errorf("update primary parent: %w", err)
			}
		}

		// ── COPPA audit entries ──
		var coppaRows [][]any
		for i, f := range s.families[3:] {
			coppaRows = append(coppaRows, []any{
				uid(dCoppa, i+1), f.ID, "consent_granted", "registration_form",
				"registered", "consented", f.PrimaryParentID, "seeded", f.CreatedAt,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO iam_coppa_audit_log (id, family_id, action, method, previous_status, new_status, performed_by, ip_hash, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			coppaRows,
		); err != nil {
			return fmt.Errorf("coppa: %w", err)
		}

		return nil
	})
}

// ─── Onboarding ───────────────────────────────────────────────────────────────

func (s *seeder) seedOnboarding() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		for i, f := range s.families[3:] {
			rows = append(rows, []any{
				uid(dWizard, i+1), f.ID, "completed", "roadmap_review",
				"{family_profile,children,methodology,roadmap_review}",
				"quiz_informed", f.CreatedAt, f.CreatedAt, f.CreatedAt,
			})
		}
		return execBatch(tx,
			"INSERT INTO onb_wizard_progress (id, family_id, status, current_step, completed_steps, methodology_path, completed_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT (family_id) DO NOTHING",
			rows,
		)
	})
}

// ─── Social Profiles ──────────────────────────────────────────────────────────

func (s *seeder) seedSocialProfiles() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		for _, f := range s.families[3:] {
			numStudents := len(f.Students)
			if numStudents == 0 {
				numStudents = 1
			}
			year := f.CreatedAt.Year()
			methName := methodologyDisplayNames[f.PrimaryMethodology]
			region := stateRegions[f.StateCode]
			if region == "" {
				region = f.StateCode
			}

			tmpl := bioTemplates[s.rng.Intn(len(bioTemplates))]
			bio := fmt.Sprintf(tmpl, numStudents, region, methName, year, f.StateCode)

			privacyJSON := `{"profile_visible":true,"show_methodology":true,"show_location":true}`
			rows = append(rows, []any{
				f.ID, bio, privacyJSON, true, f.CreatedAt, f.CreatedAt,
			})
		}
		return execBatch(tx,
			"INSERT INTO soc_profiles (family_id, bio, privacy_settings, location_visible, created_at, updated_at) VALUES ",
			"(?, ?, ?::JSONB, ?, ?, ?)",
			"ON CONFLICT (family_id) DO NOTHING",
			rows,
		)
	})
}

// ─── Friendships ──────────────────────────────────────────────────────────────

func (s *seeder) seedFriendships() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.friendships = make(map[int]map[int]bool)
		for i := range s.families {
			s.friendships[i] = make(map[int]bool)
		}

		// Seed family (idx 0) gets many friends for a rich social feed.
		seedFriendCount := 60
		friendTargets := pickN(s.rng, len(s.families)-3, seedFriendCount)
		for _, t := range friendTargets {
			idx := t + 3
			s.friendships[0][idx] = true
			s.friendships[idx][0] = true
		}

		// Friend family (idx 1) gets some friends too.
		for _, t := range pickN(s.rng, len(s.families)-3, 25) {
			idx := t + 3
			s.friendships[1][idx] = true
			s.friendships[idx][1] = true
		}

		// Each new family gets ~avgFriendsPerFamily friends (power law-ish).
		for i := 3; i < len(s.families); i++ {
			numFriends := s.families[i].ActivityLevel * 2
			if numFriends > len(s.families)-1 {
				numFriends = len(s.families) - 1
			}
			targets := pickN(s.rng, len(s.families), numFriends)
			for _, t := range targets {
				if t != i { // no self-friendship
					s.friendships[i][t] = true
					s.friendships[t][i] = true
				}
			}
		}

		// Deduplicate into ordered pairs and build rows.
		seen := make(map[[2]int]bool)
		var rows [][]any
		seq := 0
		for a, friends := range s.friendships {
			for b := range friends {
				pair := [2]int{a, b}
				if a > b {
					pair = [2]int{b, a}
				}
				if seen[pair] {
					continue
				}
				seen[pair] = true
				seq++

				status := "accepted"
				// ~5% pending
				if s.rng.Float64() < 0.05 {
					status = "pending"
				}
				created := randomDate(s.rng, s.families[a].CreatedAt, s.now)

				rows = append(rows, []any{
					uid(dFriend, seq),
					s.families[pair[0]].ID,
					s.families[pair[1]].ID,
					status, created, created,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO soc_friendships (id, requester_family_id, accepter_family_id, status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Groups ───────────────────────────────────────────────────────────────────

func (s *seeder) seedGroups() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.groupIDs = make([]string, 0, numGroups)

		// Create groups.
		var groupRows [][]any
		for i := 0; i < numGroups && i < len(groupTemplates); i++ {
			gt := groupTemplates[i]
			gID := uid(dGroup, i+1)
			s.groupIDs = append(s.groupIDs, gID)

			creatorIdx := s.rng.Intn(len(s.families)-3) + 3
			var methodology any = nil
			if gt.Methodology != "" {
				methodology = gt.Methodology
			}
			memberCount := 20 + s.rng.Intn(80) // 20–100 members

			created := randomDate(s.rng, s.families[creatorIdx].CreatedAt, s.now)
			groupRows = append(groupRows, []any{
				gID, "user_created", gt.Name, gt.Description,
				s.families[creatorIdx].ID, methodology,
				gt.JoinPolicy, memberCount, created, created,
			})
		}
		if err := execBatch(tx,
			"INSERT INTO soc_groups (id, group_type, name, description, creator_family_id, methodology_slug, join_policy, member_count, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			groupRows,
		); err != nil {
			return fmt.Errorf("groups: %w", err)
		}

		// Group memberships: each family joins 1–4 groups based on methodology and activity.
		// Track (group, family) pairs to avoid unique-constraint violations.
		type gfPair struct{ g, f string }
		seen := make(map[gfPair]bool)

		var memRows [][]any
		memSeq := 0
		for famIdx, f := range s.families {
			if famIdx < 3 {
				continue // existing families already have some memberships
			}
			numJoin := 1 + s.rng.Intn(min(4, len(s.groupIDs)))
			joined := pickN(s.rng, len(s.groupIDs), numJoin)
			for _, gIdx := range joined {
				pair := gfPair{s.groupIDs[gIdx], f.ID}
				if seen[pair] {
					continue
				}
				seen[pair] = true
				memSeq++
				role := "member"
				created := randomDate(s.rng, f.CreatedAt, s.now)
				memRows = append(memRows, []any{
					uid(dGroupMem, memSeq), s.groupIDs[gIdx], f.ID,
					role, "active", created, created, created,
				})
			}
		}

		// Seed family joins several groups for visibility.
		for _, gIdx := range pickN(s.rng, len(s.groupIDs), min(8, len(s.groupIDs))) {
			pair := gfPair{s.groupIDs[gIdx], existSeedFamilyID}
			if seen[pair] {
				continue
			}
			seen[pair] = true
			memSeq++
			memRows = append(memRows, []any{
				uid(dGroupMem, memSeq), s.groupIDs[gIdx], existSeedFamilyID,
				"member", "active", s.now.AddDate(0, -3, 0), s.now.AddDate(0, -3, 0), s.now.AddDate(0, -3, 0),
			})
		}

		return execBatch(tx,
			"INSERT INTO soc_group_members (id, group_id, family_id, role, status, joined_at, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			memRows,
		)
	})
}

// ─── Posts ─────────────────────────────────────────────────────────────────────

func (s *seeder) seedPosts() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.postIDs = nil
		s.postFamilyIdx = nil

		var rows [][]any
		seq := 0
		schoolStart := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)

		for famIdx := range s.families {
			if famIdx < 3 {
				continue // existing families have posts from basic seeder
			}
			f := s.families[famIdx]
			// Posts proportional to activity level: level * 2–4 posts.
			numPosts := f.ActivityLevel * (2 + s.rng.Intn(3))

			for j := 0; j < numPosts; j++ {
				seq++
				postID := uid(dPost, seq)
				s.postIDs = append(s.postIDs, postID)
				s.postFamilyIdx = append(s.postFamilyIdx, famIdx)

				content := postTemplates[s.rng.Intn(len(postTemplates))]
				postType := "text"
				if s.rng.Float64() < 0.15 {
					postType = "photo"
				} else if s.rng.Float64() < 0.10 {
					postType = "milestone"
				}

				visibility := "friends"
				var groupID any = nil
				// ~20% of posts go to a group.
				if len(s.groupIDs) > 0 && s.rng.Float64() < 0.20 {
					visibility = "group"
					groupID = s.groupIDs[s.rng.Intn(len(s.groupIDs))]
				}

				created := randomDate(s.rng, maxTime(schoolStart, f.CreatedAt), s.now)
				rows = append(rows, []any{
					postID, f.ID, f.PrimaryParentID, postType, content,
					groupID, visibility, 0, 0, false, created, created,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO soc_posts (id, family_id, author_parent_id, post_type, content, group_id, visibility, likes_count, comments_count, is_edited, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// ─── Comments ─────────────────────────────────────────────────────────────────

func (s *seeder) seedComments() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		for _, postID := range s.postIDs {
			// 0–5 comments per post.
			numComments := s.rng.Intn(6)
			for c := 0; c < numComments; c++ {
				seq++
				// Random commenter from families that are friends of the post author.
				commenterIdx := s.rng.Intn(len(s.families)-3) + 3
				cf := s.families[commenterIdx]
				content := commentTemplates[s.rng.Intn(len(commentTemplates))]
				created := randomDate(s.rng, s.now.AddDate(0, -6, 0), s.now)

				rows = append(rows, []any{
					uid(dComment, seq), postID, cf.ID, cf.PrimaryParentID,
					content, created, created,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO soc_comments (id, post_id, family_id, author_parent_id, content, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Post Likes ───────────────────────────────────────────────────────────────

func (s *seeder) seedPostLikes() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		var rows [][]any
		seq := 0

		for _, postID := range s.postIDs {
			// 0–10 likes per post.
			numLikes := s.rng.Intn(11)
			likers := pickN(s.rng, len(s.families), min(numLikes, len(s.families)))
			for _, likerIdx := range likers {
				seq++
				created := randomDate(s.rng, s.now.AddDate(0, -6, 0), s.now)
				rows = append(rows, []any{
					uid(dLike, seq), postID, s.families[likerIdx].ID, created,
				})
			}
		}

		return execBatch(tx,
			"INSERT INTO soc_post_likes (id, post_id, family_id, created_at) VALUES ",
			"(?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rows,
		)
	})
}

// ─── Conversations & Messages ─────────────────────────────────────────────────

func (s *seeder) seedConversations() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		// Create ~500 conversations between friends.
		numConvos := 500
		s.convoIDs = make([]string, 0, numConvos)

		var convoRows [][]any
		var partRows [][]any
		var msgRows [][]any
		convoSeq := 0
		msgSeq := 0

		for c := 0; c < numConvos; c++ {
			// Pick two random families that are friends.
			famA := s.rng.Intn(len(s.families))
			friends := s.friendships[famA]
			if len(friends) == 0 {
				continue
			}
			friendKeys := make([]int, 0, len(friends))
			for k := range friends {
				friendKeys = append(friendKeys, k)
			}
			famB := friendKeys[s.rng.Intn(len(friendKeys))]

			convoSeq++
			convoID := uid(dConvo, convoSeq)
			s.convoIDs = append(s.convoIDs, convoID)
			created := randomDate(s.rng, s.now.AddDate(0, -6, 0), s.now)

			convoRows = append(convoRows, []any{convoID, created, created})

			// Participants.
			partRows = append(partRows, []any{
				convoID, s.families[famA].PrimaryParentID, s.families[famA].ID,
			})
			partRows = append(partRows, []any{
				convoID, s.families[famB].PrimaryParentID, s.families[famB].ID,
			})

			// 2–8 messages per conversation.
			numMsgs := 2 + s.rng.Intn(7)
			for m := 0; m < numMsgs; m++ {
				msgSeq++
				sender := famA
				if m%2 == 1 {
					sender = famB
				}
				msgTime := created.Add(time.Duration(m) * time.Hour)

				content := messageContent(s.rng)
				msgRows = append(msgRows, []any{
					uid(dMessage, msgSeq), convoID,
					s.families[sender].PrimaryParentID, s.families[sender].ID,
					content, msgTime,
				})
			}
		}

		if err := execBatch(tx,
			"INSERT INTO soc_conversations (id, created_at, updated_at) VALUES ",
			"(?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			convoRows,
		); err != nil {
			return fmt.Errorf("conversations: %w", err)
		}

		if err := execBatch(tx,
			"INSERT INTO soc_conversation_participants (conversation_id, parent_id, family_id) VALUES ",
			"(?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			partRows,
		); err != nil {
			return fmt.Errorf("participants: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO soc_messages (id, conversation_id, sender_parent_id, sender_family_id, content, created_at) VALUES ",
			"(?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			msgRows,
		)
	})
}

var messageTemplates = []string{
	"Hi! How's your week going?",
	"We should get the kids together for a playdate soon!",
	"Thanks for the curriculum recommendation — we're loving it!",
	"Are you going to the park day on Thursday?",
	"Just wanted to check in. How are things?",
	"Did you see the new listing in the marketplace? Looks great!",
	"Happy to help! Let me know if you need anything else.",
	"That sounds wonderful! Count us in.",
	"We had the best field trip today — wish you could have joined!",
	"Would love to hear how you handle math with your older kids.",
	"Our family is praying for yours. Let me know how I can help.",
	"Great seeing you at co-op today! The kids had so much fun.",
	"I found a great nature study resource — want me to share the link?",
	"How did the science experiment go? My kids want to try it too!",
	"Let's plan a group hike sometime this month!",
}

func messageContent(rng *rand.Rand) string {
	return messageTemplates[rng.Intn(len(messageTemplates))]
}

// ─── Events ───────────────────────────────────────────────────────────────────

func (s *seeder) seedEvents() error {
	return bypassRLS(s.db, func(tx *gorm.DB) error {
		s.eventIDs = make([]string, 0, numEvents)
		s.eventDates = make([]time.Time, 0, numEvents)

		var eventRows [][]any
		var rsvpRows [][]any
		rsvpSeq := 0

		for i := 0; i < numEvents && i < len(eventTemplates)*4; i++ {
			et := eventTemplates[i%len(eventTemplates)]
			eID := uid(dEvent, i+1)
			s.eventIDs = append(s.eventIDs, eID)

			creatorIdx := s.rng.Intn(len(s.families)-3) + 3
			cf := s.families[creatorIdx]

			// Events span from 2 months ago to 3 months in the future.
			eventDate := randomDate(s.rng, s.now.AddDate(0, -2, 0), s.now.AddDate(0, 3, 0))
			s.eventDates = append(s.eventDates, eventDate)
			endDate := eventDate.Add(2 * time.Hour)

			region := stateRegions[cf.StateCode]
			if region == "" {
				region = cf.StateCode
			}
			locName := "Community Center"
			if et.IsVirtual {
				locName = ""
				region = ""
			}

			var groupID any = nil
			visibility := "discoverable"
			methodology := cf.PrimaryMethodology

			created := eventDate.AddDate(0, 0, -14-s.rng.Intn(30))

			eventRows = append(eventRows, []any{
				eID, cf.ID, cf.PrimaryParentID, groupID, et.Title, et.Description,
				eventDate, endDate, locName, region, et.IsVirtual, "",
				et.Capacity, visibility, "active", methodology, 0,
				created, created,
			})

			// RSVPs: 5–20 families per event.
			numRSVPs := 5 + s.rng.Intn(16)
			rsvpers := pickN(s.rng, len(s.families), min(numRSVPs, len(s.families)))
			for _, famIdx := range rsvpers {
				rsvpSeq++
				status := "going"
				r := s.rng.Float64()
				if r < 0.3 {
					status = "interested"
				} else if r < 0.4 {
					status = "not_going"
				}
				rsvpRows = append(rsvpRows, []any{
					uid(dRSVP, rsvpSeq), eID, s.families[famIdx].ID,
					status, created, created,
				})
			}
		}

		if err := execBatch(tx,
			"INSERT INTO soc_events (id, creator_family_id, creator_parent_id, group_id, title, description, event_date, end_date, location_name, location_region, is_virtual, virtual_url, capacity, visibility, status, methodology_slug, attendee_count, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			eventRows,
		); err != nil {
			return fmt.Errorf("events: %w", err)
		}

		return execBatch(tx,
			"INSERT INTO soc_event_rsvps (id, event_id, family_id, status, created_at, updated_at) VALUES ",
			"(?, ?, ?, ?, ?, ?)",
			"ON CONFLICT DO NOTHING",
			rsvpRows,
		)
	})
}
