import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface SubjectTaxonomyResponse {
  id: string;
  parent_id?: string;
  name: string;
  slug: string;
  level: number;
  children: SubjectTaxonomyResponse[];
  is_custom: boolean;
}

export interface CustomSubjectResponse {
  id: string;
  name: string;
  slug: string;
  parent_taxonomy_id?: string;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useSubjectTaxonomy(params?: {
  level?: number;
  parent_id?: string;
}) {
  return useQuery({
    queryKey: ["learning", "taxonomy", params],
    queryFn: () => {
      const sp = new URLSearchParams();
      if (params?.level !== undefined)
        sp.set("level", String(params.level));
      if (params?.parent_id) sp.set("parent_id", params.parent_id);
      const qs = sp.toString();
      return apiClient<SubjectTaxonomyResponse[]>(
        `/v1/learning/taxonomy${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60 * 10, // 10 min — taxonomy rarely changes
  });
}

// ─── Slug → Name Utilities ───────────────────────────────────────────────────

/** Flatten a taxonomy tree into a slug → display-name map. */
function flattenTaxonomy(
  nodes: SubjectTaxonomyResponse[],
  map: Map<string, string> = new Map(),
): Map<string, string> {
  for (const node of nodes) {
    map.set(node.slug, node.name);
    if (node.children.length > 0) flattenTaxonomy(node.children, map);
  }
  return map;
}

/**
 * Returns a function that converts a subject slug to its display name.
 * Falls back to a humanised slug if taxonomy hasn't loaded yet.
 */
export function useSubjectNameResolver() {
  const { data: taxonomy } = useSubjectTaxonomy();

  const nameMap = taxonomy ? flattenTaxonomy(taxonomy) : new Map<string, string>();

  return (slug: string): string => {
    if (nameMap.has(slug)) return nameMap.get(slug)!;
    // Graceful fallback: convert slug separators to spaces and title-case.
    return slug
      .replace(/[_.\-]/g, " ")
      .replace(/\b\w/g, (c) => c.toUpperCase());
  };
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useCreateCustomSubject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      name: string;
      parent_taxonomy_id?: string;
    }) =>
      apiClient<CustomSubjectResponse>(
        "/v1/learning/taxonomy/custom",
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "taxonomy"],
      });
    },
  });
}
