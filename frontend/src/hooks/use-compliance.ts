import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type AttendanceStatus = "present" | "absent" | "partial" | "excused";

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
  date: string;
  status: AttendanceStatus;
  auto_generated: boolean;
  notes: string;
  created_at: string;
}

export interface AttendanceSummary {
  student_id: string;
  student_name: string;
  days_present: number;
  days_partial: number;
  days_absent: number;
  days_excused: number;
  total_days: number;
  days_required: number;
  pace: PaceStatus;
}

export interface StandardizedTest {
  id: string;
  student_id: string;
  student_name: string;
  test_name: string;
  test_date: string;
  sections: TestSection[];
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
  sections: TestSection[];
}

export interface ComplianceAssessment {
  id: string;
  assessment_id: string;
  assessment_title: string;
  student_name: string;
  requirement_id: string;
  requirement_name: string;
  score: string;
  date: string;
}

// ─── Portfolio Types ─────────────────────────────────────────────────────────

export type PortfolioStatus = "draft" | "generating" | "ready";
export type PortfolioOrganization = "chronological" | "by_subject" | "by_type";

export interface PortfolioSummary {
  id: string;
  student_id: string;
  student_name: string;
  title: string;
  status: PortfolioStatus;
  date_range_start: string;
  date_range_end: string;
  item_count: number;
  download_url?: string;
  created_at: string;
  updated_at: string;
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

export interface PortfolioItem {
  id: string;
  item_type: "work_sample" | "assessment" | "attendance" | "journal" | "activity";
  title: string;
  subject?: string;
  date: string;
  source_id: string;
  sort_order: number;
}

export interface PortfolioItemCandidate {
  id: string;
  item_type: PortfolioItem["item_type"];
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
  items?: { source_id: string; item_type: PortfolioItem["item_type"]; sort_order: number }[];
}

// ─── Transcript Types ────────────────────────────────────────────────────────

export type TranscriptStatus = "draft" | "generating" | "ready";
export type CourseLevel = "regular" | "honors" | "ap" | "dual_enrollment";
export type GpaDisplay = "four_point" | "percentage" | "pass_fail";

export interface TranscriptSummary {
  id: string;
  student_id: string;
  student_name: string;
  title: string;
  status: TranscriptStatus;
  semester_count: number;
  total_credits: number;
  cumulative_gpa?: number;
  download_url?: string;
  created_at: string;
  updated_at: string;
}

export interface TranscriptDetail {
  id: string;
  student_id: string;
  student_name: string;
  title: string;
  status: TranscriptStatus;
  gpa_display: GpaDisplay;
  semesters: TranscriptSemester[];
  cumulative_gpa?: number;
  total_credits: number;
  download_url?: string;
  created_at: string;
  updated_at: string;
}

export interface TranscriptSemester {
  id: string;
  name: string;
  sort_order: number;
  courses: TranscriptCourse[];
  semester_gpa?: number;
  semester_credits: number;
}

export interface TranscriptCourse {
  id: string;
  title: string;
  subject?: string;
  level: CourseLevel;
  credits: number;
  grade: string;
  grade_points?: number;
  sort_order: number;
}

export interface CreateTranscriptInput {
  student_id: string;
  title: string;
  gpa_display?: GpaDisplay;
}

export interface UpdateTranscriptInput {
  title?: string;
  gpa_display?: GpaDisplay;
}

export interface AddSemesterInput {
  name: string;
  sort_order: number;
}

export interface AddCourseInput {
  semester_id: string;
  title: string;
  subject?: string;
  level: CourseLevel;
  credits: number;
  grade: string;
  sort_order: number;
}

export interface UpdateCourseInput {
  title?: string;
  subject?: string;
  level?: CourseLevel;
  credits?: number;
  grade?: string;
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
    queryFn: () => {
      const params = new URLSearchParams();
      if (month) params.set("month", month);
      const qs = params.toString();
      return apiClient<AttendanceEntry[]>(
        `/v1/compliance/attendance/${studentId}${qs ? `?${qs}` : ""}`,
      );
    },
    enabled: !!studentId,
    staleTime: 1000 * 30,
  });
}

export function useAttendanceSummary() {
  return useQuery({
    queryKey: ["compliance", "attendance", "summary"],
    queryFn: () =>
      apiClient<AttendanceSummary[]>("/v1/compliance/attendance/summary"),
    staleTime: 1000 * 60,
  });
}

export function useStandardizedTests(studentId?: string) {
  return useQuery({
    queryKey: ["compliance", "tests", studentId],
    queryFn: () => {
      const params = new URLSearchParams();
      if (studentId) params.set("student_id", studentId);
      const qs = params.toString();
      return apiClient<StandardizedTest[]>(
        `/v1/compliance/tests${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

export function useComplianceAssessments() {
  return useQuery({
    queryKey: ["compliance", "assessments"],
    queryFn: () =>
      apiClient<ComplianceAssessment[]>("/v1/compliance/assessments"),
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
    mutationFn: (body: {
      student_id: string;
      date: string;
      status: AttendanceStatus;
      notes?: string;
    }) =>
      apiClient<AttendanceEntry>("/v1/compliance/attendance", {
        method: "POST",
        body,
      }),
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
    mutationFn: (body: CreateTestRequest) =>
      apiClient<StandardizedTest>("/v1/compliance/tests", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "tests"],
      });
    },
  });
}

export function useLinkAssessment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      assessment_id: string;
      requirement_id: string;
    }) =>
      apiClient<void>("/v1/compliance/assessments/link", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "assessments"],
      });
    },
  });
}

// ─── Portfolio Queries & Mutations ──────────────────────────────────────────

export function usePortfolios(studentId?: string) {
  return useQuery({
    queryKey: ["compliance", "portfolios", studentId],
    queryFn: () => {
      const params = new URLSearchParams();
      if (studentId) params.set("student_id", studentId);
      const qs = params.toString();
      return apiClient<PortfolioSummary[]>(
        `/v1/compliance/portfolios${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

export function usePortfolioDetail(id: string | undefined) {
  return useQuery({
    queryKey: ["compliance", "portfolios", id],
    queryFn: () =>
      apiClient<PortfolioDetail>(
        `/v1/compliance/portfolios/${id ?? ""}`,
      ),
    enabled: !!id,
    staleTime: 1000 * 30,
  });
}

export function usePortfolioItemCandidates(params: {
  student_id: string;
  date_range_start: string;
  date_range_end: string;
  item_type?: PortfolioItem["item_type"];
}) {
  return useQuery({
    queryKey: ["compliance", "portfolios", "candidates", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      searchParams.set("student_id", params.student_id);
      searchParams.set("start", params.date_range_start);
      searchParams.set("end", params.date_range_end);
      if (params.item_type) searchParams.set("type", params.item_type);
      return apiClient<PortfolioItemCandidate[]>(
        `/v1/compliance/portfolios/candidates?${searchParams.toString()}`,
      );
    },
    enabled: !!params.student_id && !!params.date_range_start && !!params.date_range_end,
    staleTime: 1000 * 60,
  });
}

export function useCreatePortfolio() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreatePortfolioInput) =>
      apiClient<PortfolioSummary>("/v1/compliance/portfolios", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

export function useUpdatePortfolio(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdatePortfolioInput) =>
      apiClient<PortfolioDetail>(
        `/v1/compliance/portfolios/${id}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

export function useGeneratePortfolio(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ status: PortfolioStatus }>(
        `/v1/compliance/portfolios/${id}/generate`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios", id],
      });
    },
  });
}

export function useDeletePortfolio() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/compliance/portfolios/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "portfolios"],
      });
    },
  });
}

// ─── Transcript Queries & Mutations ─────────────────────────────────────────

export function useTranscripts(studentId?: string) {
  return useQuery({
    queryKey: ["compliance", "transcripts", studentId],
    queryFn: () => {
      const params = new URLSearchParams();
      if (studentId) params.set("student_id", studentId);
      const qs = params.toString();
      return apiClient<TranscriptSummary[]>(
        `/v1/compliance/transcripts${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

export function useTranscriptDetail(id: string | undefined) {
  return useQuery({
    queryKey: ["compliance", "transcripts", id],
    queryFn: () =>
      apiClient<TranscriptDetail>(
        `/v1/compliance/transcripts/${id ?? ""}`,
      ),
    enabled: !!id,
    staleTime: 1000 * 30,
  });
}

export function useCreateTranscript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateTranscriptInput) =>
      apiClient<TranscriptSummary>("/v1/compliance/transcripts", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useUpdateTranscript(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateTranscriptInput) =>
      apiClient<TranscriptDetail>(
        `/v1/compliance/transcripts/${id}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}

export function useAddSemester(transcriptId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AddSemesterInput) =>
      apiClient<TranscriptSemester>(
        `/v1/compliance/transcripts/${transcriptId}/semesters`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", transcriptId],
      });
    },
  });
}

export function useAddCourse(transcriptId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AddCourseInput) =>
      apiClient<TranscriptCourse>(
        `/v1/compliance/transcripts/${transcriptId}/courses`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", transcriptId],
      });
    },
  });
}

export function useUpdateCourse(transcriptId: string, courseId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateCourseInput) =>
      apiClient<TranscriptCourse>(
        `/v1/compliance/transcripts/${transcriptId}/courses/${courseId}`,
        { method: "PATCH", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", transcriptId],
      });
    },
  });
}

export function useDeleteCourse(transcriptId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (courseId: string) =>
      apiClient<void>(
        `/v1/compliance/transcripts/${transcriptId}/courses/${courseId}`,
        { method: "DELETE" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", transcriptId],
      });
    },
  });
}

export function useGenerateTranscript(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiClient<{ status: TranscriptStatus }>(
        `/v1/compliance/transcripts/${id}/generate`,
        { method: "POST" },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts", id],
      });
    },
  });
}

export function useDeleteTranscript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/compliance/transcripts/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["compliance", "transcripts"],
      });
    },
  });
}
