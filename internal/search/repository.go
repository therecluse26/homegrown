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
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    ) AS is_friend,
    ts_rank(to_tsvector('english', f.display_name), plainto_tsquery('english', $2)) AS relevance
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id
LEFT JOIN method_definitions md ON md.id = f.primary_methodology_id
WHERE f.id != $1
AND (
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    )
    OR sp.location_visible = true
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = $1)
)
AND f.display_name ILIKE '%' || $2 || '%'
ORDER BY relevance DESC, f.id
LIMIT $3 OFFSET $4`

	type familyRow struct {
		FamilyID        uuid.UUID `gorm:"column:family_id"`
		DisplayName     string    `gorm:"column:display_name"`
		MethodologyName *string   `gorm:"column:methodology_name"`
		LocationRegion  *string   `gorm:"column:location_region"`
		IsFriend        bool      `gorm:"column:is_friend"`
		Relevance       float32   `gorm:"column:relevance"`
	}

	var rows []familyRow
	if err := r.db.WithContext(ctx).Raw(sql, searcherFamilyID, query, limit, offset).Scan(&rows).Error; err != nil {
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
func (r *PgSocialSearchRepository) SearchGroups(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error) {
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
        websearch_to_tsquery('english', $2)
    ) AS relevance
FROM soc_groups g
LEFT JOIN method_definitions md ON md.id = g.methodology_id
WHERE to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, ''))
    @@ websearch_to_tsquery('english', $2)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = $1)
)
AND ($3::uuid IS NULL OR g.methodology_id = $3)
ORDER BY relevance DESC, g.id
LIMIT $4 OFFSET $5`

	type groupRow struct {
		GroupID         uuid.UUID `gorm:"column:group_id"`
		Name            string    `gorm:"column:name"`
		Description     *string   `gorm:"column:description"`
		MemberCount     int32     `gorm:"column:member_count"`
		MethodologyName *string   `gorm:"column:methodology_name"`
		Relevance       float32   `gorm:"column:relevance"`
	}

	var rows []groupRow
	if err := r.db.WithContext(ctx).Raw(sql, searcherFamilyID, query, methodologyID, limit, offset).Scan(&rows).Error; err != nil {
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
func (r *PgSocialSearchRepository) SearchEvents(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error) {
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
        websearch_to_tsquery('english', $2)
    ) AS relevance
FROM soc_events e
WHERE e.status = 'active'
AND to_tsvector('english', coalesce(e.title, '') || ' ' || coalesce(e.description, ''))
    @@ websearch_to_tsquery('english', $2)
AND (
    e.visibility = 'discoverable'
    OR (
        e.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = $1 AND sf.accepter_family_id = e.creator_family_id)
                OR (sf.accepter_family_id = $1 AND sf.requester_family_id = e.creator_family_id)
            )
        )
    )
    OR (
        e.visibility = 'group'
        AND e.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = e.group_id
            AND gm.family_id = $1
            AND gm.status = 'active'
        )
    )
    OR e.creator_family_id = $1
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = e.creator_family_id)
       OR (sb.blocker_family_id = e.creator_family_id AND sb.blocked_family_id = $1)
)
AND ($3::uuid IS NULL OR e.methodology_id = $3)
ORDER BY relevance DESC, e.id
LIMIT $4 OFFSET $5`

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

	var rows []eventRow
	if err := r.db.WithContext(ctx).Raw(sql, searcherFamilyID, query, methodologyID, limit, offset).Scan(&rows).Error; err != nil {
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
    ts_rank(p.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM soc_posts p
JOIN iam_families f ON f.id = p.family_id
LEFT JOIN soc_groups g ON g.id = p.group_id
WHERE p.search_vector @@ websearch_to_tsquery('english', $2)
AND (
    p.family_id = $1
    OR (
        p.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = $1 AND sf.accepter_family_id = p.family_id)
                OR (sf.accepter_family_id = $1 AND sf.requester_family_id = p.family_id)
            )
        )
    )
    OR (
        p.visibility = 'group'
        AND p.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = p.group_id
            AND gm.family_id = $1
            AND gm.status = 'active'
        )
    )
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = p.family_id)
       OR (sb.blocker_family_id = p.family_id AND sb.blocked_family_id = $1)
)
ORDER BY relevance DESC, p.id
LIMIT $3 OFFSET $4`

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
	if err := r.db.WithContext(ctx).Raw(sql, searcherFamilyID, query, limit, offset).Scan(&rows).Error; err != nil {
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
    ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) AS relevance
FROM mkt_listings l
JOIN mkt_publishers p ON p.id = l.publisher_id
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  AND ($2::uuid[] IS NULL OR l.methodology_tags && $2)
  AND ($3::text[] IS NULL OR l.subject_tags && $3)
  AND ($4::smallint IS NULL OR l.grade_min <= $4)
  AND ($5::smallint IS NULL OR l.grade_max >= $5)
  AND ($6::int IS NULL OR l.price_cents <= $6)
  AND ($7::int IS NULL OR l.price_cents >= $7)
  AND ($8::text IS NULL OR l.content_type = $8)
  AND ($9::text[] IS NULL OR l.worldview_tags && $9)
  AND ($10::bool IS NULL OR NOT $10 OR l.price_cents = 0)
ORDER BY
    CASE WHEN $11 = 'relevance' THEN ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) END DESC,
    CASE WHEN $11 = 'price_asc' THEN l.price_cents END ASC,
    CASE WHEN $11 = 'price_desc' THEN l.price_cents END DESC,
    CASE WHEN $11 = 'rating' THEN l.rating_avg END DESC,
    CASE WHEN $11 = 'recency' THEN l.published_at END DESC,
    l.id
LIMIT $12 OFFSET $13`

	var methodologyTags any
	if len(filters.MethodologyTags) > 0 {
		methodologyTags = uuidSliceToStringSlice(filters.MethodologyTags)
	}

	var subjectTags any
	if len(filters.SubjectTags) > 0 {
		subjectTags = filters.SubjectTags
	}

	var worldviewTags any
	if len(filters.WorldviewTags) > 0 {
		worldviewTags = filters.WorldviewTags
	}

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
	if err := r.db.WithContext(ctx).Raw(sql,
		query, methodologyTags, subjectTags,
		filters.GradeMax, filters.GradeMin,
		filters.PriceMax, filters.PriceMin,
		filters.ContentType, worldviewTags, filters.FreeOnly,
		string(sortOrder), limit, offset,
	).Scan(&rows).Error; err != nil {
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
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  AND ($2::uuid[] IS NULL OR l.methodology_tags && $2)
  AND ($3::text[] IS NULL OR l.subject_tags && $3)
  AND ($4::smallint IS NULL OR l.grade_min <= $4)
  AND ($5::smallint IS NULL OR l.grade_max >= $5)
  AND ($6::int IS NULL OR l.price_cents <= $6)
  AND ($7::int IS NULL OR l.price_cents >= $7)
  AND ($8::text IS NULL OR l.content_type = $8)
  AND ($9::text[] IS NULL OR l.worldview_tags && $9)
  AND ($10::bool IS NULL OR NOT $10 OR l.price_cents = 0)`

	var totalCount int64
	if err := r.db.WithContext(ctx).Raw(countSQL,
		query, methodologyTags, subjectTags,
		filters.GradeMax, filters.GradeMin,
		filters.PriceMax, filters.PriceMin,
		filters.ContentType, worldviewTags, filters.FreeOnly,
	).Scan(&totalCount).Error; err != nil {
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
LEFT JOIN method_definitions md ON md.id = mt.tag_value
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
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

	var allRows []learningRow

	if includeActivity {
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
    ts_rank(al.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_activity_logs al
JOIN iam_students s ON s.id = al.student_id
WHERE al.family_id = $1
  AND al.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR al.student_id = $3)
  AND ($4::date IS NULL OR al.activity_date >= $4)
  AND ($5::date IS NULL OR al.activity_date <= $5)
  AND ($6::text[] IS NULL OR al.subject_tags && $6)
ORDER BY relevance DESC, al.id
LIMIT $7`

		var subjectTags any
		if len(filters.SubjectTags) > 0 {
			subjectTags = filters.SubjectTags
		}

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(actSQL, familyID, query, filters.StudentID, filters.DateFrom, filters.DateTo, subjectTags, limit).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search activities: %w", err)
		}
		allRows = append(allRows, rows...)
	}

	if includeJournal {
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
    ts_rank(je.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_journal_entries je
JOIN iam_students s ON s.id = je.student_id
WHERE je.family_id = $1
  AND je.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR je.student_id = $3)
  AND ($4::date IS NULL OR je.entry_date >= $4)
  AND ($5::date IS NULL OR je.entry_date <= $5)
  AND ($6::text[] IS NULL OR je.subject_tags && $6)
ORDER BY relevance DESC, je.id
LIMIT $7`

		var subjectTags any
		if len(filters.SubjectTags) > 0 {
			subjectTags = filters.SubjectTags
		}

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(jrnlSQL, familyID, query, filters.StudentID, filters.DateFrom, filters.DateTo, subjectTags, limit).Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("search journals: %w", err)
		}
		allRows = append(allRows, rows...)
	}

	if includeReading {
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
    ts_rank(ri.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_reading_items ri
JOIN learn_reading_progress rp ON rp.reading_item_id = ri.id AND rp.family_id = $1
JOIN iam_students s ON s.id = rp.student_id
WHERE ri.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR rp.student_id = $3)
  AND ($4::date IS NULL OR rp.created_at::date >= $4)
  AND ($5::date IS NULL OR rp.created_at::date <= $5)
  AND ($6::text[] IS NULL OR ri.subject_tags && $6)
ORDER BY relevance DESC, ri.id
LIMIT $7`

		var subjectTags any
		if len(filters.SubjectTags) > 0 {
			subjectTags = filters.SubjectTags
		}

		var rows []learningRow
		if err := r.db.WithContext(ctx).Raw(readSQL, familyID, query, filters.StudentID, filters.DateFrom, filters.DateTo, subjectTags, limit).Scan(&rows).Error; err != nil {
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
    similarity(l.title, $1) AS score
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.title % $1
ORDER BY score DESC
LIMIT $2`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql, query, limit).Scan(&suggestions).Error; err != nil {
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
WHERE f.id != $1
AND f.display_name ILIKE $2 || '%'
AND (
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    )
    OR sp.location_visible = true
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = $1)
)
LIMIT $3
)
UNION ALL
(
SELECT
    g.name AS text,
    'group' AS entity_type,
    g.id AS entity_id,
    1.0::real AS score
FROM soc_groups g
WHERE g.name ILIKE $2 || '%'
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = $1)
)
LIMIT $3
)
ORDER BY text
LIMIT $3`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql, searcherFamilyID, query, limit).Scan(&suggestions).Error; err != nil {
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
WHERE family_id = $1
AND title ILIKE $2 || '%'
LIMIT $3
)
UNION ALL
(
SELECT title AS text,
       'journal' AS entity_type,
       id AS entity_id,
       1.0::real AS score
FROM learn_journal_entries
WHERE family_id = $1
AND title ILIKE $2 || '%'
LIMIT $3
)
ORDER BY text
LIMIT $3`

	var suggestions []AutocompleteSuggestion
	if err := r.db.WithContext(ctx).Raw(sql, familyID, query, limit).Scan(&suggestions).Error; err != nil {
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
