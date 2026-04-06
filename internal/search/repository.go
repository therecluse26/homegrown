package search

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Cursor Helpers [12-search §4.1]
// Offset-encoded base64 cursors for relevance-ordered pagination.
// ═══════════════════════════════════════════════════════════════════════════════

func encodeSearchCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeSearchCursor(cursor string) (int, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}
	offset, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid cursor offset: %w", err)
	}
	return offset, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Social Search Repository [12-search §6.1, §10.1]
// ═══════════════════════════════════════════════════════════════════════════════

type PgSocialSearchRepository struct {
	db *gorm.DB
}

func NewPgSocialSearchRepository(db *gorm.DB) *PgSocialSearchRepository {
	return &PgSocialSearchRepository{db: db}
}

// SearchFamilies searches families by display name with friendship + discovery privacy. [12-search §10.1]
func (r *PgSocialSearchRepository) SearchFamilies(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeSearchCursor(*cursor)
		if err != nil {
			return nil, fmt.Errorf("search families: %w", err)
		}
	}

	sql := `
SELECT
    f.id AS family_id,
    f.display_name,
    md.display_name AS methodology_name,
    CASE WHEN sp.location_visible THEN f.location_region ELSE NULL END AS location_region,
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = ? AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = ? AND sf.requester_family_id = f.id)
        )
    ) AS is_friend,
    ts_rank(to_tsvector('english', f.display_name), plainto_tsquery('english', ?)) AS relevance
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id
LEFT JOIN method_definitions md ON md.slug = f.primary_methodology_slug
WHERE f.id != ?
AND (
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = ? AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = ? AND sf.requester_family_id = f.id)
        )
    )
    OR sp.location_visible = true
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = ?)
)
AND f.display_name ILIKE '%' || ? || '%'
ORDER BY relevance DESC, f.id
LIMIT ? OFFSET ?`

	type familyRow struct {
		FamilyID        uuid.UUID `gorm:"column:family_id"`
		DisplayName     string    `gorm:"column:display_name"`
		MethodologyName *string   `gorm:"column:methodology_name"`
		LocationRegion  *string   `gorm:"column:location_region"`
		IsFriend        bool      `gorm:"column:is_friend"`
		Relevance       float32   `gorm:"column:relevance"`
	}

	var rows []familyRow
	if err := r.db.WithContext(ctx).Raw(sql,
		searcherFamilyID, searcherFamilyID, // is_friend subquery
		query,                              // ts_rank
		searcherFamilyID,                   // WHERE f.id !=
		searcherFamilyID, searcherFamilyID, // friends exists subquery
		searcherFamilyID, searcherFamilyID, // blocks check
		query,                              // ILIKE
		limit, offset,                      // pagination
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search families: %w", err)
	}

	results := make([]SocialSearchResult, len(rows))
	for i, row := range rows {
		results[i] = SocialSearchResult{
			Relevance: row.Relevance,
			Result: SearchResult{
				Type: "family",
				Family: &FamilySearchResult{
					FamilyID:        row.FamilyID,
					DisplayName:     row.DisplayName,
					MethodologyName: row.MethodologyName,
					LocationRegion:  row.LocationRegion,
					IsFriend:        row.IsFriend,
					Relevance:       row.Relevance,
				},
			},
		}
	}
	return results, nil
}

// SearchGroups searches groups by name and description. [12-search §10.1]
func (r *PgSocialSearchRepository) SearchGroups(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologySlug *string, limit int, cursor *string) ([]SocialSearchResult, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeSearchCursor(*cursor)
		if err != nil {
			return nil, fmt.Errorf("search groups: %w", err)
		}
	}

	sql := `
SELECT
    g.id AS group_id,
    g.name,
    g.description,
    (SELECT COUNT(*) FROM soc_group_members gm WHERE gm.group_id = g.id AND gm.status = 'active') AS member_count,
    md.display_name AS methodology_name,
    ts_rank(
        to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, '')),
        websearch_to_tsquery('english', ?)
    ) AS relevance
FROM soc_groups g
LEFT JOIN method_definitions md ON md.slug = g.methodology_slug
WHERE to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, ''))
    @@ websearch_to_tsquery('english', ?)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = ?)
)
AND (? = '' OR g.methodology_slug = ?)
ORDER BY relevance DESC, g.id
LIMIT ? OFFSET ?`

	type groupRow struct {
		GroupID         uuid.UUID `gorm:"column:group_id"`
		Name            string    `gorm:"column:name"`
		Description     *string   `gorm:"column:description"`
		MemberCount     int32     `gorm:"column:member_count"`
		MethodologyName *string   `gorm:"column:methodology_name"`
		Relevance       float32   `gorm:"column:relevance"`
	}

	// Use empty string as "no filter" sentinel — GORM Raw() drops all args
	// when any variadic param is a nil interface{}.
	methSlug := ""
	if methodologySlug != nil {
		methSlug = *methodologySlug
	}

	var rows []groupRow
	if err := r.db.WithContext(ctx).Raw(sql,
		query,                // websearch_to_tsquery in SELECT
		query,                // websearch_to_tsquery in WHERE
		searcherFamilyID,     // blocker check #1
		searcherFamilyID,     // blocker check #2
		methSlug, methSlug,   // methodology filter (= '' OR = ?)
		limit, offset,        // pagination
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search groups: %w", err)
	}

	results := make([]SocialSearchResult, len(rows))
	for i, row := range rows {
		results[i] = SocialSearchResult{
			Relevance: row.Relevance,
			Result: SearchResult{
				Type: "group",
				Group: &GroupSearchResult{
					GroupID:         row.GroupID,
					Name:            row.Name,
					Description:     row.Description,
					MemberCount:     row.MemberCount,
					MethodologyName: row.MethodologyName,
					Relevance:       row.Relevance,
				},
			},
		}
	}
	return results, nil
}

// SearchEvents searches events by title and description with visibility enforcement. [12-search §10.1]
func (r *PgSocialSearchRepository) SearchEvents(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologySlug *string, limit int, cursor *string) ([]SocialSearchResult, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeSearchCursor(*cursor)
		if err != nil {
			return nil, fmt.Errorf("search events: %w", err)
		}
	}

	sql := `
SELECT
    e.id AS event_id,
    e.title,
    e.description,
    e.event_date,
    e.location_name,
    e.is_virtual,
    e.visibility,
    e.attendee_count,
    ts_rank(
        to_tsvector('english', coalesce(e.title, '') || ' ' || coalesce(e.description, '')),
        websearch_to_tsquery('english', ?)
    ) AS relevance
FROM soc_events e
WHERE e.status = 'active'
AND to_tsvector('english', coalesce(e.title, '') || ' ' || coalesce(e.description, ''))
    @@ websearch_to_tsquery('english', ?)
AND (
    e.visibility = 'discoverable'
    OR (
        e.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = ? AND sf.accepter_family_id = e.creator_family_id)
                OR (sf.accepter_family_id = ? AND sf.requester_family_id = e.creator_family_id)
            )
        )
    )
    OR (
        e.visibility = 'group'
        AND e.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = e.group_id
            AND gm.family_id = ?
            AND gm.status = 'active'
        )
    )
    OR e.creator_family_id = ?
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = e.creator_family_id)
       OR (sb.blocker_family_id = e.creator_family_id AND sb.blocked_family_id = ?)
)
AND (? = '' OR e.methodology_slug = ?)
ORDER BY relevance DESC, e.id
LIMIT ? OFFSET ?`

	type eventRow struct {
		EventID       uuid.UUID `gorm:"column:event_id"`
		Title         string    `gorm:"column:title"`
		Description   *string   `gorm:"column:description"`
		EventDate     time.Time `gorm:"column:event_date"`
		LocationName  *string   `gorm:"column:location_name"`
		IsVirtual     bool      `gorm:"column:is_virtual"`
		Visibility    string    `gorm:"column:visibility"`
		AttendeeCount int32     `gorm:"column:attendee_count"`
		Relevance     float32   `gorm:"column:relevance"`
	}

	// Use empty string as "no filter" sentinel — GORM Raw() drops all args
	// when any variadic param is a nil interface{}.
	methSlug := ""
	if methodologySlug != nil {
		methSlug = *methodologySlug
	}

	var rows []eventRow
	if err := r.db.WithContext(ctx).Raw(sql,
		query,                                              // websearch_to_tsquery in SELECT
		query,                                              // websearch_to_tsquery in WHERE
		searcherFamilyID, searcherFamilyID,                 // friends check
		searcherFamilyID,                                   // group member check
		searcherFamilyID,                                   // creator check
		searcherFamilyID, searcherFamilyID,                 // blocks check
		methSlug, methSlug,                                 // methodology filter
		limit, offset,                                      // pagination
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}

	results := make([]SocialSearchResult, len(rows))
	for i, row := range rows {
		results[i] = SocialSearchResult{
			Relevance: row.Relevance,
			Result: SearchResult{
				Type: "event",
				Event: &EventSearchResult{
					EventID:       row.EventID,
					Title:         row.Title,
					Description:   row.Description,
					EventDate:     row.EventDate,
					LocationName:  row.LocationName,
					IsVirtual:     row.IsVirtual,
					Visibility:    row.Visibility,
					AttendeeCount: row.AttendeeCount,
					Relevance:     row.Relevance,
				},
			},
		}
	}
	return results, nil
}

// SearchPosts searches posts by content with privacy enforcement. [12-search §10.1]
func (r *PgSocialSearchRepository) SearchPosts(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeSearchCursor(*cursor)
		if err != nil {
			return nil, fmt.Errorf("search posts: %w", err)
		}
	}

	sql := `
SELECT
    p.id AS post_id,
    LEFT(p.content, 200) AS content_snippet,
    p.family_id AS author_family_id,
    f.display_name AS author_display_name,
    g.name AS group_name,
    p.created_at,
    ts_rank(p.search_vector, websearch_to_tsquery('english', ?)) AS relevance
FROM soc_posts p
JOIN iam_families f ON f.id = p.family_id
LEFT JOIN soc_groups g ON g.id = p.group_id
WHERE p.search_vector @@ websearch_to_tsquery('english', ?)
AND (
    p.family_id = ?
    OR (
        p.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = ? AND sf.accepter_family_id = p.family_id)
                OR (sf.accepter_family_id = ? AND sf.requester_family_id = p.family_id)
            )
        )
    )
    OR (
        p.visibility = 'group'
        AND p.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = p.group_id
            AND gm.family_id = ?
            AND gm.status = 'active'
        )
    )
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = p.family_id)
       OR (sb.blocker_family_id = p.family_id AND sb.blocked_family_id = ?)
)
ORDER BY relevance DESC, p.id
LIMIT ? OFFSET ?`

	type postRow struct {
		PostID            uuid.UUID `gorm:"column:post_id"`
		ContentSnippet    string    `gorm:"column:content_snippet"`
		AuthorFamilyID    uuid.UUID `gorm:"column:author_family_id"`
		AuthorDisplayName string    `gorm:"column:author_display_name"`
		GroupName         *string   `gorm:"column:group_name"`
		CreatedAt         time.Time `gorm:"column:created_at"`
		Relevance         float32   `gorm:"column:relevance"`
	}

	var rows []postRow
	if err := r.db.WithContext(ctx).Raw(sql,
		query,                              // ts_rank
		query,                              // search_vector match
		searcherFamilyID,                   // own posts
		searcherFamilyID, searcherFamilyID, // friends check
		searcherFamilyID,                   // group member check
		searcherFamilyID, searcherFamilyID, // blocks check
		limit, offset,                      // pagination
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}

	results := make([]SocialSearchResult, len(rows))
	for i, row := range rows {
		results[i] = SocialSearchResult{
			Relevance: row.Relevance,
			Result: SearchResult{
				Type: "post",
				Post: &PostSearchResult{
					PostID:            row.PostID,
					ContentSnippet:    row.ContentSnippet,
					AuthorFamilyID:    row.AuthorFamilyID,
					AuthorDisplayName: row.AuthorDisplayName,
					GroupName:         row.GroupName,
					CreatedAt:         row.CreatedAt,
					Relevance:         row.Relevance,
				},
			},
		}
	}
	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Marketplace Search Repository [12-search §6.2, §10.2]
// ═══════════════════════════════════════════════════════════════════════════════

type PgMarketplaceSearchRepository struct {
	db *gorm.DB
}

func NewPgMarketplaceSearchRepository(db *gorm.DB) *PgMarketplaceSearchRepository {
	return &PgMarketplaceSearchRepository{db: db}
}

// SearchListings performs full-text search with faceted filtering and sorting. [12-search §10.2]
func (r *PgMarketplaceSearchRepository) SearchListings(ctx context.Context, query string, filters *MarketplaceSearchFilters, sortOrder SearchSortOrder, limit int, cursor *string) (*MarketplaceSearchResults, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeSearchCursor(*cursor)
		if err != nil {
			return nil, fmt.Errorf("search listings: %w", err)
		}
	}

	// Build WHERE clause dynamically to avoid untyped NULL issues with GORM.
	// PostgreSQL cannot infer the type of NULL when used with array operators (&&).
	var whereClauses []string
	var whereArgs []any

	whereClauses = append(whereClauses, "l.status = 'published'")
	whereClauses = append(whereClauses, "l.search_vector @@ websearch_to_tsquery('english', ?)")
	whereArgs = append(whereArgs, query)

	if len(filters.MethodologyTags) > 0 {
		whereClauses = append(whereClauses, "l.methodology_tags && ?")
		whereArgs = append(whereArgs, uuidSliceToStringSlice(filters.MethodologyTags))
	}
	if len(filters.SubjectTags) > 0 {
		whereClauses = append(whereClauses, "l.subject_tags && ?")
		whereArgs = append(whereArgs, filters.SubjectTags)
	}
	if filters.GradeMax != nil {
		whereClauses = append(whereClauses, "l.grade_min <= ?")
		whereArgs = append(whereArgs, *filters.GradeMax)
	}
	if filters.GradeMin != nil {
		whereClauses = append(whereClauses, "l.grade_max >= ?")
		whereArgs = append(whereArgs, *filters.GradeMin)
	}
	if filters.PriceMax != nil {
		whereClauses = append(whereClauses, "l.price_cents <= ?")
		whereArgs = append(whereArgs, *filters.PriceMax)
	}
	if filters.PriceMin != nil {
		whereClauses = append(whereClauses, "l.price_cents >= ?")
		whereArgs = append(whereArgs, *filters.PriceMin)
	}
	if filters.ContentType != nil {
		whereClauses = append(whereClauses, "l.content_type = ?")
		whereArgs = append(whereArgs, *filters.ContentType)
	}
	if len(filters.WorldviewTags) > 0 {
		whereClauses = append(whereClauses, "l.worldview_tags && ?")
		whereArgs = append(whereArgs, filters.WorldviewTags)
	}
	if filters.FreeOnly != nil && *filters.FreeOnly {
		whereClauses = append(whereClauses, "l.price_cents = 0")
	}

	whereSQL := strings.Join(whereClauses, "\n  AND ")

	sortStr := string(sortOrder)

	sql := `
SELECT
    l.id AS listing_id,
    l.title,
    LEFT(l.description, 200) AS description_snippet,
    l.price_cents,
    l.content_type,
    l.rating_avg,
    l.rating_count,
    p.name AS publisher_name,
    array_to_string(l.methodology_tags, ',') AS methodology_tags_csv,
    array_to_string(l.subject_tags, ',') AS subject_tags_csv,
    l.published_at,
    ts_rank(l.search_vector, websearch_to_tsquery('english', ?)) AS relevance
FROM mkt_listings l
JOIN mkt_publishers p ON p.id = l.publisher_id
WHERE ` + whereSQL + `
ORDER BY
    CASE WHEN ? = 'relevance' THEN ts_rank(l.search_vector, websearch_to_tsquery('english', ?)) END DESC,
    CASE WHEN ? = 'price_asc' THEN l.price_cents END ASC,
    CASE WHEN ? = 'price_desc' THEN l.price_cents END DESC,
    CASE WHEN ? = 'rating' THEN l.rating_avg END DESC,
    CASE WHEN ? = 'recency' THEN l.published_at END DESC,
    l.id
LIMIT ? OFFSET ?`

	// Build args: ts_rank in SELECT, then WHERE args, then ORDER BY + pagination
	var args []any
	args = append(args, query) // ts_rank in SELECT
	args = append(args, whereArgs...)
	args = append(args, sortStr, query) // ORDER BY relevance
	args = append(args, sortStr)        // ORDER BY price_asc
	args = append(args, sortStr)        // ORDER BY price_desc
	args = append(args, sortStr)        // ORDER BY rating
	args = append(args, sortStr)        // ORDER BY recency
	args = append(args, limit, offset)  // pagination

	type listingRow struct {
		ListingID          uuid.UUID `gorm:"column:listing_id"`
		Title              string    `gorm:"column:title"`
		DescriptionSnippet string    `gorm:"column:description_snippet"`
		PriceCents         int32     `gorm:"column:price_cents"`
		ContentType        string    `gorm:"column:content_type"`
		RatingAvg          *float64  `gorm:"column:rating_avg"`
		RatingCount        int32     `gorm:"column:rating_count"`
		PublisherName      string    `gorm:"column:publisher_name"`
		MethodologyTagsCsv string    `gorm:"column:methodology_tags_csv"`
		SubjectTagsCsv     string    `gorm:"column:subject_tags_csv"`
		PublishedAt        time.Time `gorm:"column:published_at"`
		Relevance          float32   `gorm:"column:relevance"`
	}

	var rows []listingRow
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("search listings: %w", err)
	}

	listings := make([]ListingSearchResult, len(rows))
	for i, row := range rows {
		listings[i] = ListingSearchResult{
			ListingID:          row.ListingID,
			Title:              row.Title,
			DescriptionSnippet: row.DescriptionSnippet,
			PriceCents:         row.PriceCents,
			ContentType:        row.ContentType,
			RatingAvg:          row.RatingAvg,
			RatingCount:        row.RatingCount,
			PublisherName:      row.PublisherName,
			MethodologyTags:    parseUUIDTags(row.MethodologyTagsCsv),
			SubjectTags:        parseCsvTags(row.SubjectTagsCsv),
			PublishedAt:        row.PublishedAt,
			Relevance:          row.Relevance,
		}
	}

	// Get total count with the same filters
	countSQL := `
SELECT COUNT(*)
FROM mkt_listings l
WHERE ` + whereSQL

	var totalCount int64
	if err := r.db.WithContext(ctx).Raw(countSQL, whereArgs...).Scan(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("count listings: %w", err)
	}

	// Build next cursor if there are more results
	var nextCursor *string
	if len(listings) == limit {
		c := encodeSearchCursor(offset + limit)
		nextCursor = &c
	}

	return &MarketplaceSearchResults{
		Listings:   listings,
		TotalCount: totalCount,
		NextCursor: nextCursor,
	}, nil
}

// CountFacets counts listings per facet value across all 6 dimensions. [12-search §10.2]
func (r *PgMarketplaceSearchRepository) CountFacets(ctx context.Context, query string, filters *MarketplaceSearchFilters) (*FacetCounts, error) {
	facets := &FacetCounts{}

	// ── Methodology tags facet (with display name from method_definitions) ──
	methodologySQL := `
SELECT mt.tag_value::text AS value,
       COALESCE(md.display_name, mt.tag_value::text) AS display_name,
       COUNT(*) AS count
FROM mkt_listings l, unnest(l.methodology_tags) AS mt(tag_value)
LEFT JOIN method_definitions md ON md.slug = mt.tag_value::text
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
GROUP BY mt.tag_value, md.display_name
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(methodologySQL, query).Scan(&facets.MethodologyTags).Error; err != nil {
		return nil, fmt.Errorf("count methodology facets: %w", err)
	}

	// ── Subject tags facet ──
	subjectSQL := `
SELECT st.tag_value AS value,
       st.tag_value AS display_name,
       COUNT(*) AS count
FROM mkt_listings l, unnest(l.subject_tags) AS st(tag_value)
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
GROUP BY st.tag_value
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(subjectSQL, query).Scan(&facets.SubjectTags).Error; err != nil {
		return nil, fmt.Errorf("count subject facets: %w", err)
	}

	// ── Content type facet ──
	contentTypeSQL := `
SELECT l.content_type AS value,
       l.content_type AS display_name,
       COUNT(*) AS count
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
GROUP BY l.content_type
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(contentTypeSQL, query).Scan(&facets.ContentType).Error; err != nil {
		return nil, fmt.Errorf("count content type facets: %w", err)
	}

	// ── Worldview tags facet ──
	worldviewSQL := `
SELECT wt.tag_value AS value,
       wt.tag_value AS display_name,
       COUNT(*) AS count
FROM mkt_listings l, unnest(l.worldview_tags) AS wt(tag_value)
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
GROUP BY wt.tag_value
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(worldviewSQL, query).Scan(&facets.WorldviewTags).Error; err != nil {
		return nil, fmt.Errorf("count worldview facets: %w", err)
	}

	// ── Price range facet ──
	priceSQL := `
SELECT
    CASE
        WHEN l.price_cents = 0 THEN 'free'
        WHEN l.price_cents BETWEEN 1 AND 1000 THEN '1_to_10'
        WHEN l.price_cents BETWEEN 1001 AND 2500 THEN '10_to_25'
        WHEN l.price_cents BETWEEN 2501 AND 5000 THEN '25_to_50'
        ELSE '50_plus'
    END AS value,
    CASE
        WHEN l.price_cents = 0 THEN 'Free'
        WHEN l.price_cents BETWEEN 1 AND 1000 THEN '$1–$10'
        WHEN l.price_cents BETWEEN 1001 AND 2500 THEN '$10–$25'
        WHEN l.price_cents BETWEEN 2501 AND 5000 THEN '$25–$50'
        ELSE '$50+'
    END AS display_name,
    COUNT(*) AS count
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
GROUP BY value, display_name
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(priceSQL, query).Scan(&facets.PriceRanges).Error; err != nil {
		return nil, fmt.Errorf("count price facets: %w", err)
	}

	// ── Rating range facet ──
	ratingSQL := `
SELECT
    CASE
        WHEN l.rating_avg >= 4.0 THEN '4_to_5'
        WHEN l.rating_avg >= 3.0 THEN '3_to_4'
        WHEN l.rating_avg >= 2.0 THEN '2_to_3'
        ELSE '1_to_2'
    END AS value,
    CASE
        WHEN l.rating_avg >= 4.0 THEN '4–5 stars'
        WHEN l.rating_avg >= 3.0 THEN '3–4 stars'
        WHEN l.rating_avg >= 2.0 THEN '2–3 stars'
        ELSE '1–2 stars'
    END AS display_name,
    COUNT(*) AS count
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', ?)
  AND l.rating_avg IS NOT NULL
GROUP BY value, display_name
ORDER BY count DESC`

	if err := r.db.WithContext(ctx).Raw(ratingSQL, query).Scan(&facets.RatingRanges).Error; err != nil {
		return nil, fmt.Errorf("count rating facets: %w", err)
	}

	return facets, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Learning Search Repository [12-search §6.3, §10.3]
// ═══════════════════════════════════════════════════════════════════════════════

type PgLearningSearchRepository struct {
	db *gorm.DB
}

func NewPgLearningSearchRepository(db *gorm.DB) *PgLearningSearchRepository {
	return &PgLearningSearchRepository{db: db}
}

// SearchLearning searches family's own learning data with UNION ALL. [12-search §10.3]
func (r *PgLearningSearchRepository) SearchLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, filters *LearningSearchFilters, limit int, cursor *string) ([]LearningSearchResult, error) {
	familyID := familyScope.FamilyID()

	// Note: cursor pagination for learning is handled at the service level
	// since we run separate queries per source type and merge results.
	_ = cursor

	// Build source type filter
	includeActivity := filters.SourceType == nil || *filters.SourceType == LearningSourceTypeActivity
	includeJournal := filters.SourceType == nil || *filters.SourceType == LearningSourceTypeJournal
	includeReading := filters.SourceType == nil || *filters.SourceType == LearningSourceTypeReading

	type learningRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		SourceType  string    `gorm:"column:source_type"`
		Title       string    `gorm:"column:title"`
		DescSnippet *string   `gorm:"column:description_snippet"`
		StudentID   uuid.UUID `gorm:"column:student_id"`
		StudentName string    `gorm:"column:student_name"`
		SubjectTags string    `gorm:"column:subject_tags"`
		ItemDate    time.Time `gorm:"column:item_date"`
		EntryType   string    `gorm:"column:entry_type"`
		Author      *string   `gorm:"column:author"`
		Status      string    `gorm:"column:status"`
		Relevance   float32   `gorm:"column:relevance"`
	}

	// Build dynamic filter clauses shared across all learning sub-queries.
	// We avoid "? IS NULL OR col op ?" patterns because GORM + pgx can't
	// infer the type of an untyped NULL for array operators (&&).
	buildLearningFilters := func(studentCol, dateCol, tagsCol string) (string, []any) {
		var clauses []string
		var args []any
		if filters.StudentID != nil {
			clauses = append(clauses, studentCol+" = ?")
			args = append(args, *filters.StudentID)
		}
		if filters.DateFrom != nil {
			clauses = append(clauses, dateCol+" >= ?")
			args = append(args, *filters.DateFrom)
		}
		if filters.DateTo != nil {
			clauses = append(clauses, dateCol+" <= ?")
			args = append(args, *filters.DateTo)
		}
		if len(filters.SubjectTags) > 0 {
			clauses = append(clauses, tagsCol+" && ?")
			args = append(args, filters.SubjectTags)
		}
		extra := ""
		if len(clauses) > 0 {
			extra = "\n  AND " + strings.Join(clauses, "\n  AND ")
		}
		return extra, args
	}

	var allRows []learningRow

	if includeActivity {
		filterSQL, filterArgs := buildLearningFilters("al.student_id", "al.activity_date", "al.subject_tags")

		actSQL := `
SELECT
    al.id,
    'activity' AS source_type,
    al.title,
    LEFT(al.description, 200) AS description_snippet,
    al.student_id,
    s.display_name AS student_name,
    array_to_string(al.subject_tags, ',') AS subject_tags,
    al.activity_date AS item_date,
    '' AS entry_type,
    NULL AS author,
    '' AS status,
    ts_rank(al.search_vector, websearch_to_tsquery('english', ?)) AS relevance
FROM learn_activity_logs al
JOIN iam_students s ON s.id = al.student_id
WHERE al.family_id = ?
  AND al.search_vector @@ websearch_to_tsquery('english', ?)` + filterSQL + `
ORDER BY relevance DESC, al.id
LIMIT ?`

		var args []any
		args = append(args, query, familyID, query)
		args = append(args, filterArgs...)
		args = append(args, limit)

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(actSQL, args...).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search activities: %w", err)
		}
		allRows = append(allRows, rows...)
	}

	if includeJournal {
		filterSQL, filterArgs := buildLearningFilters("je.student_id", "je.entry_date", "je.subject_tags")

		jrnlSQL := `
SELECT
    je.id,
    'journal' AS source_type,
    je.title,
    LEFT(je.content, 200) AS description_snippet,
    je.student_id,
    s.display_name AS student_name,
    array_to_string(je.subject_tags, ',') AS subject_tags,
    je.entry_date AS item_date,
    je.entry_type,
    NULL AS author,
    '' AS status,
    ts_rank(je.search_vector, websearch_to_tsquery('english', ?)) AS relevance
FROM learn_journal_entries je
JOIN iam_students s ON s.id = je.student_id
WHERE je.family_id = ?
  AND je.search_vector @@ websearch_to_tsquery('english', ?)` + filterSQL + `
ORDER BY relevance DESC, je.id
LIMIT ?`

		var args []any
		args = append(args, query, familyID, query)
		args = append(args, filterArgs...)
		args = append(args, limit)

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(jrnlSQL, args...).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search journals: %w", err)
		}
		allRows = append(allRows, rows...)
	}

	if includeReading {
		filterSQL, filterArgs := buildLearningFilters("rp.student_id", "rp.created_at::date", "ri.subject_tags")

		readSQL := `
SELECT
    ri.id,
    'reading' AS source_type,
    ri.title,
    LEFT(ri.description, 200) AS description_snippet,
    rp.student_id,
    s.display_name AS student_name,
    array_to_string(ri.subject_tags, ',') AS subject_tags,
    rp.created_at::date AS item_date,
    '' AS entry_type,
    ri.author,
    rp.status,
    ts_rank(ri.search_vector, websearch_to_tsquery('english', ?)) AS relevance
FROM learn_reading_items ri
JOIN learn_reading_progress rp ON rp.reading_item_id = ri.id AND rp.family_id = ?
JOIN iam_students s ON s.id = rp.student_id
WHERE ri.search_vector @@ websearch_to_tsquery('english', ?)` + filterSQL + `
ORDER BY relevance DESC, ri.id
LIMIT ?`

		var args []any
		args = append(args, query, familyID, query)
		args = append(args, filterArgs...)
		args = append(args, limit)

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(readSQL, args...).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search reading items: %w", err)
		}
		allRows = append(allRows, rows...)
	}

	// Convert to LearningSearchResult with all fields populated
	results := make([]LearningSearchResult, len(allRows))
	for i, row := range allRows {
		result := SearchResult{Type: row.SourceType}
		switch row.SourceType {
		case "activity":
			result.Activity = &ActivitySearchResult{
				ActivityID:   row.ID,
				Title:        row.Title,
				Description:  row.DescSnippet,
				StudentID:    row.StudentID,
				StudentName:  row.StudentName,
				ActivityDate: row.ItemDate,
				SubjectTags:  parseCsvTags(row.SubjectTags),
				Relevance:    row.Relevance,
			}
		case "journal":
			snippet := ""
			if row.DescSnippet != nil {
				snippet = *row.DescSnippet
			}
			result.Journal = &JournalSearchResult{
				JournalID:      row.ID,
				Title:          row.Title,
				ContentSnippet: snippet,
				StudentID:      row.StudentID,
				StudentName:    row.StudentName,
				EntryDate:      row.ItemDate,
				EntryType:      row.EntryType,
				Relevance:      row.Relevance,
			}
		case "reading":
			result.ReadingItem = &ReadingItemSearchResult{
				ReadingItemID: row.ID,
				Title:         row.Title,
				Author:        row.Author,
				Description:   row.DescSnippet,
				StudentID:     row.StudentID,
				StudentName:   row.StudentName,
				Status:        row.Status,
				Relevance:     row.Relevance,
			}
		}
		results[i] = LearningSearchResult{
			Result:    result,
			Relevance: row.Relevance,
		}
	}

	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Autocomplete Repository [12-search §6.4, §11]
// ═══════════════════════════════════════════════════════════════════════════════

type PgAutocompleteRepository struct {
	db *gorm.DB
}

func NewPgAutocompleteRepository(db *gorm.DB) *PgAutocompleteRepository {
	return &PgAutocompleteRepository{db: db}
}

// AutocompleteMarketplace uses pg_trgm similarity on listing titles. [12-search §11.1]
func (r *PgAutocompleteRepository) AutocompleteMarketplace(ctx context.Context, query string, limit int) ([]AutocompleteSuggestion, error) {
	sql := `
SELECT DISTINCT
    l.title AS text,
    'listing' AS entity_type,
    l.id AS entity_id,
    similarity(l.title, ?) AS score
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.title % ?
ORDER BY score DESC
LIMIT ?`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql,
		query,  // similarity
		query,  // trigram match
		limit,  // pagination
	).Scan(&suggestions).Error; err != nil {
		return nil, fmt.Errorf("autocomplete marketplace: %w", err)
	}
	return suggestions, nil
}

// AutocompleteSocial uses ILIKE prefix match with privacy enforcement. [12-search §11.2]
func (r *PgAutocompleteRepository) AutocompleteSocial(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int) ([]AutocompleteSuggestion, error) {
	sql := `
(
SELECT
    f.display_name AS text,
    'family' AS entity_type,
    f.id AS entity_id,
    1.0::real AS score
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id
WHERE f.id != ?
AND f.display_name ILIKE ? || '%'
AND (
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = ? AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = ? AND sf.requester_family_id = f.id)
        )
    )
    OR sp.location_visible = true
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = ?)
)
LIMIT ?
)
UNION ALL
(
SELECT
    g.name AS text,
    'group' AS entity_type,
    g.id AS entity_id,
    1.0::real AS score
FROM soc_groups g
WHERE g.name ILIKE ? || '%'
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = ? AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = ?)
)
LIMIT ?
)
ORDER BY text
LIMIT ?`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql,
		searcherFamilyID,                           // f.id !=
		query,                                      // ILIKE family
		searcherFamilyID, searcherFamilyID,         // friends check
		searcherFamilyID, searcherFamilyID,         // blocks check (family)
		limit,                                      // LIMIT (family subquery)
		query,                                      // ILIKE group
		searcherFamilyID, searcherFamilyID,         // blocks check (group)
		limit,                                      // LIMIT (group subquery)
		limit,                                      // LIMIT (outer)
	).Scan(&suggestions).Error; err != nil {
		return nil, fmt.Errorf("autocomplete social: %w", err)
	}
	return suggestions, nil
}

// AutocompleteLearning uses ILIKE prefix match on family-scoped learning titles. [12-search §11.3]
func (r *PgAutocompleteRepository) AutocompleteLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, limit int) ([]AutocompleteSuggestion, error) {
	familyID := familyScope.FamilyID()

	sql := `
(
SELECT title AS text,
       'activity' AS entity_type,
       id AS entity_id,
       1.0::real AS score
FROM learn_activity_logs
WHERE family_id = ?
AND title ILIKE ? || '%'
LIMIT ?
)
UNION ALL
(
SELECT title AS text,
       'journal' AS entity_type,
       id AS entity_id,
       1.0::real AS score
FROM learn_journal_entries
WHERE family_id = ?
AND title ILIKE ? || '%'
LIMIT ?
)
ORDER BY text
LIMIT ?`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql,
		familyID, query, limit,   // activity subquery
		familyID, query, limit,   // journal subquery
		limit,                    // outer LIMIT
	).Scan(&suggestions).Error; err != nil {
		return nil, fmt.Errorf("autocomplete learning: %w", err)
	}
	return suggestions, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func uuidSliceToStringSlice(uuids []uuid.UUID) []string {
	result := make([]string, len(uuids))
	for i, u := range uuids {
		result[i] = u.String()
	}
	return result
}

// parseCsvTags splits a comma-separated string into a string slice. Empty input returns nil.
func parseCsvTags(csv string) []string {
	if csv == "" {
		return nil
	}
	return strings.Split(csv, ",")
}

// parseUUIDTags splits a comma-separated UUID string into a UUID slice.
func parseUUIDTags(csv string) []uuid.UUID {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	result := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if u, err := uuid.Parse(p); err == nil {
			result = append(result, u)
		}
	}
	return result
}
