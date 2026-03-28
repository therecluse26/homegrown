import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface FamilySearchResult {
  family_id: string;
  display_name: string;
  methodology_name?: string;
  location_region?: string;
  is_friend: boolean;
  relevance: number;
}

export interface GroupSearchResult {
  group_id: string;
  name: string;
  description?: string;
  member_count: number;
  methodology_name?: string;
  relevance: number;
}

export interface EventSearchResult {
  event_id: string;
  title: string;
  description?: string;
  event_date: string;
  location_name?: string;
  is_virtual: boolean;
  visibility: string;
  attendee_count: number;
  relevance: number;
}

export interface PostSearchResult {
  post_id: string;
  content_snippet: string;
  author_family_id: string;
  author_display_name: string;
  group_name?: string;
  created_at: string;
  relevance: number;
}

export interface ListingSearchResult {
  listing_id: string;
  title: string;
  description_snippet: string;
  price_cents: number;
  content_type: string;
  rating_avg?: number;
  rating_count: number;
  publisher_name: string;
  methodology_tags: string[];
  subject_tags: string[];
  published_at: string;
  relevance: number;
}

export interface ActivitySearchResult {
  activity_id: string;
  title: string;
  description?: string;
  student_id: string;
  student_name: string;
  activity_date: string;
  subject_tags: string[];
  relevance: number;
}

export interface JournalSearchResult {
  journal_id: string;
  title: string;
  content_snippet: string;
  student_id: string;
  student_name: string;
  entry_date: string;
  entry_type: string;
  relevance: number;
}

export interface ReadingItemSearchResult {
  reading_item_id: string;
  title: string;
  author?: string;
  description?: string;
  student_id: string;
  student_name: string;
  status: string;
  relevance: number;
}

export interface SearchResult {
  type: string;
  family?: FamilySearchResult;
  group?: GroupSearchResult;
  event?: EventSearchResult;
  post?: PostSearchResult;
  listing?: ListingSearchResult;
  activity?: ActivitySearchResult;
  journal?: JournalSearchResult;
  reading_item?: ReadingItemSearchResult;
}

export interface FacetBucket {
  value: string;
  display_name: string;
  count: number;
}

export interface FacetCounts {
  methodology_tags: FacetBucket[];
  subject_tags: FacetBucket[];
  content_type: FacetBucket[];
  worldview_tags: FacetBucket[];
  price_ranges: FacetBucket[];
  rating_ranges: FacetBucket[];
}

export interface SearchResponse {
  results: SearchResult[];
  total_count: number;
  facets?: FacetCounts;
  next_cursor?: string;
}

export interface AutocompleteSuggestion {
  text: string;
  entity_type: string;
  entity_id: string;
  score: number;
}

export interface AutocompleteResponse {
  suggestions: AutocompleteSuggestion[];
}

export type SearchScope = "social" | "marketplace" | "learning";

export interface SearchParams {
  q: string;
  scope: SearchScope;
  cursor?: string;
  limit?: number;
  sort?: string;
  // Social filters
  sub_scope?: string;
  methodology_slug?: string;
  // Marketplace filters
  methodology_tags?: string[];
  subject_tags?: string[];
  grade_min?: number;
  grade_max?: number;
  price_min?: number;
  price_max?: number;
  content_type?: string;
  worldview_tags?: string[];
  free_only?: boolean;
  // Learning filters
  student_id?: string;
  source_type?: string;
  date_from?: string;
  date_to?: string;
}

// ─── Hooks ───────────────────────────────────────────────────────────────────

function buildSearchQuery(params: SearchParams): string {
  const qs = new URLSearchParams();
  qs.set("q", params.q);
  qs.set("scope", params.scope);
  if (params.cursor) qs.set("cursor", params.cursor);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.sort) qs.set("sort", params.sort);
  if (params.sub_scope) qs.set("sub_scope", params.sub_scope);
  if (params.methodology_slug) qs.set("methodology_slug", params.methodology_slug);
  if (params.content_type) qs.set("content_type", params.content_type);
  if (params.grade_min != null) qs.set("grade_min", String(params.grade_min));
  if (params.grade_max != null) qs.set("grade_max", String(params.grade_max));
  if (params.price_min != null) qs.set("price_min", String(params.price_min));
  if (params.price_max != null) qs.set("price_max", String(params.price_max));
  if (params.free_only) qs.set("free_only", "true");
  if (params.student_id) qs.set("student_id", params.student_id);
  if (params.source_type) qs.set("source_type", params.source_type);
  if (params.date_from) qs.set("date_from", params.date_from);
  if (params.date_to) qs.set("date_to", params.date_to);
  params.methodology_tags?.forEach((t) => qs.append("methodology_tags", t));
  params.subject_tags?.forEach((t) => qs.append("subject_tags", t));
  params.worldview_tags?.forEach((t) => qs.append("worldview_tags", t));
  return qs.toString();
}

export function useSearch(params: SearchParams | null) {
  return useQuery({
    queryKey: ["search", params],
    queryFn: () =>
      apiClient<SearchResponse>(
        `/v1/search/search?${buildSearchQuery(params!)}`,
      ),
    enabled: !!params && params.q.length >= 2,
  });
}

export function useAutocomplete(q: string, scope?: SearchScope) {
  const qs = new URLSearchParams();
  qs.set("q", q);
  if (scope) qs.set("scope", scope);
  return useQuery({
    queryKey: ["search", "autocomplete", q, scope],
    queryFn: () =>
      apiClient<AutocompleteResponse>(
        `/v1/search/autocomplete?${qs.toString()}`,
      ),
    enabled: q.length >= 1,
  });
}

export function useSearchSuggestions() {
  return useQuery({
    queryKey: ["search", "suggestions"],
    queryFn: () =>
      apiClient<AutocompleteResponse>("/v1/search/suggestions"),
  });
}
