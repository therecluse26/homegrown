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
