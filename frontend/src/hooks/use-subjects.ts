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
