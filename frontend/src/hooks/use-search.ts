import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

// ─── Type aliases (from generated schema) ──────────────────────────────────

export type FamilySearchResult =
  components["schemas"]["search.FamilySearchResult"];
export type GroupSearchResult =
  components["schemas"]["search.GroupSearchResult"];
export type EventSearchResult =
  components["schemas"]["search.EventSearchResult"];
export type PostSearchResult =
  components["schemas"]["search.PostSearchResult"];
export type ListingSearchResult =
  components["schemas"]["search.ListingSearchResult"];
export type ActivitySearchResult =
  components["schemas"]["search.ActivitySearchResult"];
export type JournalSearchResult =
  components["schemas"]["search.JournalSearchResult"];
export type ReadingItemSearchResult =
  components["schemas"]["search.ReadingItemSearchResult"];
export type SearchResult = components["schemas"]["search.SearchResult"];
export type FacetBucket = components["schemas"]["search.FacetBucket"];
export type FacetCounts = components["schemas"]["search.FacetCounts"];
export type SearchResponse = components["schemas"]["search.SearchResponse"];
export type AutocompleteSuggestion =
  components["schemas"]["search.AutocompleteSuggestion"];
export type AutocompleteResponse =
  components["schemas"]["search.AutocompleteResponse"];

// ─── Local types (not returned by API) ──────────────────────────────────────

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
        `/v1/search?${buildSearchQuery(params!)}`,
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
