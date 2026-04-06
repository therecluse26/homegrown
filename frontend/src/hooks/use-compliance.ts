import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type AttendanceStatus = "present_full" | "present_partial" | "absent" | "not_applicable";

export type PaceStatus = "ahead" | "on_track" | "behind";

export interface StateRequirement {
  state_code: string;
  state_name: string;
  days_required: number;
  hours_required: number;
  subjects_required: string[];
  notification_threshold_days: number;
  description: string;
}

export interface ComplianceConfig {
  id: string;
  family_id: string;
  state_code: string;
  days_required: number;
  hours_required: number;
  configured: boolean;
}

export interface AttendanceEntry {
  id: string;
  student_id: string;
  attendance_date: string;
  status: AttendanceStatus;
  duration_minutes: number | null;
  notes: string | null;
  is_auto: boolean;
  manual_override: boolean;
  created_at: string;
}

export interface AttendanceSummaryRaw {
  total_days: number;
  present_full: number;
  present_partial: number;
  absent: number;
  not_applicable: number;
  total_hours: number;
  state_required_days: number | null;
  state_required_hours: number | null;
  pace_status: PaceStatus | null;
  projected_total_days: number | null;
}

export interface AttendanceSummary {
  days_present: number;
  days_partial: number;
  days_absent: number;
  total_days: number;
  days_required: number;
  pace: PaceStatus;
}

export interface StandardizedTest {
  id: string;
  student_id: string;
  test_name: string;
  test_date: string;
  grade_level: number | null;
  scores: Record<string, number>;
  composite_score: number | null;
  percentile: number | null;
  notes: string | null;
  created_at: string;
}

export interface TestSection {
  name: string;
  score: string;
}

export interface CreateTestRequest {
  student_id: string;
  test_name: string;
  test_date: string;
  scores: Record<string, number>;
}

export interface ComplianceAssessment {
  id: string;
  student_id: string;
  title: string;
  subject: string;
  assessment_type: string;
  score: number | null;
  max_score: number | null;
  grade_letter: string | null;
  grade_points: number | null;
  is_passing: boolean | null;
  assessment_date: string;
  notes: string | null;
  created_at: string;
}

// ─── Portfolio Types ─────────────────────────────────────────────────────────

export type PortfolioStatus = "configuring" | "generating" | "ready" | "failed" | "expired";
export type PortfolioOrganization = "chronological" | "by_subject" | "by_type";

export interface PortfolioSummary {
  id: string;
  title: string;
  status: PortfolioStatus;
  item_count: number;
  date_range_start: string;
  date_range_end: string;
  generated_at: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface PortfolioDetail {
  id: string;
  student_id: string;
  student_name: string;
  title: string;
  status: PortfolioStatus;
  date_range_start: string;
  date_range_end: string;
  organization: PortfolioOrganization;
  cover_student_name?: string;
  cover_date_range?: string;
  items: PortfolioItem[];
  download_url?: string;
  created_at: string;
  updated_at: string;
}

export type PortfolioItemType = "work_sample" | "assessment" | "attendance" | "journal" | "activity";

export interface PortfolioItem {
  id: string;
  source_type: PortfolioItemType;
  source_id: string;
  display_order: number;
  cached_title: string;
  cached_subject?: string | null;
  cached_date: string;
  cached_description?: string | null;
}

export interface PortfolioItemCandidate {
  id: string;
  item_type: PortfolioItemType;
  title: string;
  subject?: string;
  date: string;
}

export interface CreatePortfolioInput {
  student_id: string;
  title: string;
  date_range_start: string;
  date_range_end: string;
  organization: PortfolioOrganization;
  cover_student_name?: string;
  cover_date_range?: string;
}

export interface UpdatePortfolioInput {
  title?: string;
  organization?: PortfolioOrganization;
  cover_student_name?: string;
  cover_date_range?: string;
  items?: { source_type: PortfolioItemType; source_id: string }[];
}

// ─── Transcript Types ────────────────────────────────────────────────────────

export type TranscriptStatus = "configuring" | "generating" | "ready" | "failed" | "expired";
export type CourseLevel = "regular" | "honors" | "ap" | "dual_enrollment";
export type GpaDisplay = "four_point" | "percentage" | "pass_fail";

export interface TranscriptSummary {
  id: string;
  title: string;
  status: TranscriptStatus;
  grade_levels: string[];
  generated_at: string | null;
  created_at: string;
}

export interface TranscriptDetail {
  id: string;
  student_id: string;
  student_name: string;
  title: string;
  grade_levels: string[];
  status: string;
  gpa_unweighted: number | null;
  gpa_weighted: number | null;
  courses: TranscriptCourse[];
  generated_at: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface TranscriptCourse {
  id: string;
  student_id: string;
  title: string;
  subject: string;
  grade_level: number;
  credits: number;
  grade_letter: string | null;
  grade_points: number | null;
  level: string;
  school_year: string;
  semester: string | null;
  created_at: string;
}

export interface CreateTranscriptInput {
  student_id: string;
  title: string;
  grade_levels?: string[];
  gpa_display?: GpaDisplay;
}

export interface UpdateTranscriptInput {
  title?: string;
}

export interface AddCourseInput {
  title: string;
  subject: string;
  grade_level: number;
  credits: number;
  grade_letter?: string;
  grade_points?: number;
  level: string;
  school_year: string;
  semester?: string;
}

export interface UpdateCourseInput {
  title?: string;
  subject?: string;
  level?: string;
  credits?: number;
  grade_letter?: string;
  grade_points?: number;
  semester?: string;
}

// ─── Queries ────────────────────────────────────────────────────────────────

export function useStateRequirements(stateCode: string | undefined) {
  return useQuery({
    queryKey: ["compliance", "requirements", stateCode],
    queryFn: () =>
      apiClient<StateRequirement>(
        `/v1/compliance/state-requirements/${stateCode ?? ""}`,
      ),
    enabled: !!stateCode,
    staleTime: 1000 * 60 * 60, // 1 hour — state requirements rarely change
  });
}

export function useComplianceConfig() {
  return useQuery({
    queryKey: ["compliance", "config"],
    queryFn: () =>
      apiClient<ComplianceConfig>("/v1/compliance/config"),
    staleTime: 1000 * 60 * 5,
  });
}

export function useAttendance(studentId: string, month?: string) {
  return useQuery({
    queryKey: ["compliance", "attendance", studentId, month],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (month) {
        // Backend expects start_date/end_date in RFC3339 format
        const parts = month.split("-").map(Number);
        const y = parts[0]!;
        const m = parts[1]!;
        const start = new Date(Date.UTC(y, m - 1, 1));
        const end = new Date(Date.UTC(y, m, 0, 23, 59, 59));
        params.set("start_date", start.toISOString());
        params.set("end_date", end.toISOString());
      }
      const qs = params.toString();
      const resp = await apiClient<{ records: AttendanceEntry[]; next_cursor: string | null }>(
        `/v1/compliance/students/${studentId}/attendance${qs ? `?${qs}` : ""}`,
      );
      return resp.records;
    },
    enabled: !!studentId,
    staleTime: 1000 * 30,
  });
}

export function useAttendanceSummary(studentId: string) {
  // Use school-year range (Aug 1 of prior year through Jul 31)
  const now = new Date();
  const yearStart = now.getMonth() >= 7 ? now.getFullYear() : now.getFullYear() - 1;
  const startDate = `${yearStart}-08-01T00:00:00Z`;
  const endDate = `${yearStart + 1}-07-31T23:59:59Z`;

  return useQuery({
    queryKey: ["compliance", "attendance", "summary", studentId],
    queryFn: async (): Promise<AttendanceSummary> => {
      const raw = await apiClient<AttendanceSummaryRaw>(
        `/v1/compliance/students/${studentId}/attendance/summary?start_date=${startDate}&end_date=${endDate}`,
      );
      return {
        days_present: raw.present_full,
        days_partial: raw.present_partial,
        days_absent: raw.absent,
        total_days: raw.total_days,
        days_required: raw.state_required_days ?? 0,
        pace: raw.pace_status ?? "on_track",
      };
    },
    enabled: !!studentId,
    staleTime: 1000 * 60,
  });
}

export function useStandardizedTests(studentId: string) {
  return useQuery({
    queryKey: ["compliance", "tests", studentId],
    queryFn: async () => {
      const resp = await apiClient<{ tests: StandardizedTest[]; next_cursor: string | null }>(
        `/v1/compliance/students/${studentId}/tests`,
      );
      return resp.tests;
    },
    enabled: !!studentId,
    staleTime: 1000 * 60,
  });
}

export function useComplianceAssessments(studentId: string) {
  return useQuery({
    queryKey: ["compliance", "assessments", studentId],
    queryFn: async () => {
      const resp = await apiClient<{ records: ComplianceAssessment[]; next_cursor: string | null }>(
        `/v1/compliance/students/${studentId}/assessments`,
      );
      return resp.records;
    },
    enabled: !!studentId,
    staleTime: 1000 * 60,
  });
}

// ─── Mutations ──────────────────────────────────────────────────────────────

export function useSaveComplianceConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      state_code: string;
      days_required: number;
      hours_required: number;
    }) =>
      apiClient<ComplianceConfig>("/v1/compliance/config", {
        method: "PUT",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["compliance", "config"] });
    },
  });
}

export function useRecordAttendance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: {
      student_id: string;
      attendance_date: string;
      status: AttendanceStatus;
      duration_minutes?: number;
      notes?: string;
    }) => {
      const { student_id, ...body } = params;
      return apiClient<AttendanceEntry>(
        `/v1/compliance/students/${student_id}/attendance`,
        { method: "POST", body },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "attendance"],
      });
    },
  });
}

export function useCreateStandardizedTest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CreateTestRequest) => {
      const { student_id, ...body } = params;
      return apiClient<StandardizedTest>(
        `/v1/compliance/students/${student_id}/tests`,
        { method: "POST", body },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "tests"],
      });
    },
  });
}

export function useCreateAssessment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      student_id: string;
      assessment_id: string;
      requirement_id: string;
    }) =>
      apiClient<ComplianceAssessment>(
        `/v1/compliance/students/${body.student_id}/assessments`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "assessments"],
      });
    },
  });
}

// ─── Portfolio Queries & Mutations ──────────────────────────────────────────

export function usePortfolios(studentId: string) {
  return useQuery({
    queryKey: ["compliance", "portfolios", studentId],
    queryFn: () =>
      apiClient<PortfolioSummary[]>(
        `/v1/compliance/students/${studentId}/portfolios`,
      ),
    enabled: !!studentId,
    staleTime: 1000 * 60,
  });
}

export function usePortfolioDetail(studentId: string, id: string | undefined) {
  return useQuery({
    queryKey: ["compliance", "portfolios", studentId, id],
    queryFn: () =>
      apiClient<PortfolioDetail>(
        `/v1/compliance/students/${studentId}/portfolios/${id ?? ""}`,
      ),
    enabled: !!studentId && !!id,
    staleTime: 1000 * 30,
  });
}

export function usePortfolioItemCandidates(params: {
  student_id: string;
  date_range_start: string;
  date_range_end: string;
  item_type?: PortfolioItemType;
}) {
  return useQuery({
    queryKey: ["compliance", "portfolios", "candidates", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      searchParams.set("start", params.date_range_start);
      searchParams.set("end", params.date_range_end);
      if (params.item_type) searchParams.set("type", params.item_type);
      return apiClient<PortfolioItemCandidate[]>(
        `/v1/compliance/students/${params.student_id}/portfolios/candidates?${searchParams.toString()}`,
      );
    },
    enabled: !!params.student_id && !!params.date_range_start && !!params.date_range_end,
    staleTime: 1000 * 60,
  });
}

export function useCreatePortfolio() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CreatePortfolioInput) => {
      const { student_id, date_range_start, date_range_end, ...rest } = params;
      return apiClient<PortfolioSummary>(
        `/v1/compliance/students/${student_id}/portfolios`,
        {
          method: "POST",
          body: {
            ...rest,
            date_range_start: `${date_range_start}T00:00:00Z`,
            date_range_end: `${date_range_end}T23:59:59Z`,
          },
        },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

export function useUpdatePortfolio(studentId: string, id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdatePortfolioInput) =>
      apiClient<PortfolioDetail>(
        `/v1/compliance/students/${studentId}/portfolios/${id}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

export function useGeneratePortfolio(studentId: string, id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ status: PortfolioStatus }>(
        `/v1/compliance/students/${studentId}/portfolios/${id}/generate`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios", studentId, id],
      });
    },
  });
}

export function useDeletePortfolio(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/compliance/students/${studentId}/portfolios/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

// ─── Transcript Queries & Mutations ─────────────────────────────────────────

export function useTranscripts(studentId: string) {
  return useQuery({
    queryKey: ["compliance", "transcripts", studentId],
    queryFn: () =>
      apiClient<TranscriptSummary[]>(
        `/v1/compliance/students/${studentId}/transcripts`,
      ),
    enabled: !!studentId,
    staleTime: 1000 * 60,
  });
}

export function useTranscriptDetail(studentId: string, id: string | undefined) {
  return useQuery({
    queryKey: ["compliance", "transcripts", studentId, id],
    queryFn: () =>
      apiClient<TranscriptDetail>(
        `/v1/compliance/students/${studentId}/transcripts/${id ?? ""}`,
      ),
    enabled: !!studentId && !!id,
    staleTime: 1000 * 30,
  });
}

export function useCreateTranscript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: CreateTranscriptInput) => {
      const { student_id, gpa_display: _, ...rest } = params;
      return apiClient<TranscriptSummary>(
        `/v1/compliance/students/${student_id}/transcripts`,
        {
          method: "POST",
          body: { ...rest, grade_levels: rest.grade_levels ?? [] },
        },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useUpdateTranscript(studentId: string, id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateTranscriptInput) =>
      apiClient<TranscriptDetail>(
        `/v1/compliance/students/${studentId}/transcripts/${id}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useAddCourse(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AddCourseInput) =>
      apiClient<TranscriptCourse>(
        `/v1/compliance/students/${studentId}/courses`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useUpdateCourse(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: { courseId: string } & UpdateCourseInput) => {
      const { courseId, ...body } = params;
      return apiClient<TranscriptCourse>(
        `/v1/compliance/students/${studentId}/courses/${courseId}`,
        { method: "PATCH", body },
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useDeleteCourse(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (courseId: string) =>
      apiClient<void>(
        `/v1/compliance/students/${studentId}/courses/${courseId}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useGenerateTranscript(studentId: string, id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ status: TranscriptStatus }>(
        `/v1/compliance/students/${studentId}/transcripts/${id}/generate`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", studentId, id],
      });
    },
  });
}

export function useDeleteTranscript(studentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(
        `/v1/compliance/students/${studentId}/transcripts/${id}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}
