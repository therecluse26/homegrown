import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type CalendarSource =
  | "schedule"
  | "activities"
  | "attendance"
  | "events";

export type ScheduleCategory =
  | "lesson"
  | "reading"
  | "activity"
  | "assessment"
  | "field_trip"
  | "co_op"
  | "break"
  | "custom";

export interface CalendarItemDetails {
  type: string;
  description?: string;
  notes?: string;
  linked_activity_id?: string;
  subject?: string;
  tags?: string[];
  status?: string;
  group_name?: string;
  location?: string;
  rsvp_status?: string;
}

export interface CalendarItem {
  id: string;
  source: CalendarSource;
  title: string;
  start_time?: string;
  end_time?: string;
  duration_minutes?: number;
  category?: string;
  color?: string;
  student_id?: string;
  student_name?: string;
  is_completed?: boolean;
  date: string;
  details: CalendarItemDetails;
}

export interface CalendarDay {
  date: string;
  items: CalendarItem[];
}

export interface CalendarResponse {
  start: string;
  end: string;
  days: CalendarDay[];
}

export interface DayViewResponse {
  date: string;
  schedule_items: ScheduleItemResponse[];
  activities: ActivitySummary[];
  attendance?: AttendanceSummary;
  events: EventSummary[];
}

export interface WeekViewResponse {
  week_start: string;
  week_end: string;
  days: CalendarDay[];
}

export interface ScheduleItemResponse {
  id: string;
  title: string;
  description?: string;
  student_id?: string;
  student_name?: string;
  start_date: string;
  start_time?: string;
  end_time?: string;
  duration_minutes?: number;
  category: ScheduleCategory;
  subject_id?: string;
  color?: string;
  is_completed: boolean;
  completed_at?: string;
  linked_activity_id?: string;
  notes?: string;
  created_at: string;
}

export interface ScheduleItemListResponse {
  data: ScheduleItemResponse[];
  next_cursor?: string;
  has_more: boolean;
}

export interface ActivitySummary {
  id: string;
  title: string;
  date: string;
  student_id?: string;
  subject?: string;
  tags?: string[];
}

export interface AttendanceSummary {
  id: string;
  date: string;
  student_id?: string;
  status: string;
}

export interface EventSummary {
  id: string;
  title: string;
  date: string;
  start_time?: string;
  end_time?: string;
  group_name?: string;
  location?: string;
  rsvp_status?: string;
}

export interface CreateScheduleItemInput {
  title: string;
  description?: string;
  student_id?: string;
  start_date: string;
  start_time?: string;
  end_time?: string;
  duration_minutes?: number;
  category?: ScheduleCategory;
  subject_id?: string;
  color?: string;
  notes?: string;
}

export interface UpdateScheduleItemInput {
  title?: string;
  description?: string;
  student_id?: string;
  start_date?: string;
  start_time?: string;
  end_time?: string;
  duration_minutes?: number;
  category?: ScheduleCategory;
  subject_id?: string;
  color?: string;
  notes?: string;
}

export interface LogAsActivityInput {
  description?: string;
  tags?: string[];
}

// ─── Schedule Template Types ────────────────────────────────────────────────

export interface ScheduleTemplateItem {
  title: string;
  category: string;
  day_of_week: number;
  start_time: string;
  duration_minutes: number;
}

export interface ScheduleTemplate {
  id: string;
  name: string;
  description: string;
  methodology_slug?: string;
  items: ScheduleTemplateItem[];
  is_default: boolean;
}

export interface CreateScheduleTemplateInput {
  name: string;
  description?: string;
  methodology_slug?: string;
  items: ScheduleTemplateItem[];
}

export interface ApplyScheduleTemplateInput {
  week_start_date: string;
  student_id?: string;
}

// ─── Schedule Export Types ──────────────────────────────────────────────────

export interface ExportScheduleInput {
  format: "csv" | "ical";
  student_id?: string;
  start_date: string;
  end_date: string;
}

export interface ScheduleExportResponse {
  download_url: string;
  expires_at: string;
}

// ─── Calendar Queries ───────────────────────────────────────────────────────

export function useCalendar(params: {
  start: string;
  end: string;
  student_id?: string;
}) {
  return useQuery({
    queryKey: ["planning", "calendar", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      searchParams.set("start", params.start);
      searchParams.set("end", params.end);
      if (params.student_id)
        searchParams.set("student_id", params.student_id);
      return apiClient<CalendarResponse>(
        `/v1/planning/calendar?${searchParams.toString()}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

export function useDayView(date: string, studentId?: string) {
  return useQuery({
    queryKey: ["planning", "calendar", "day", date, studentId],
    queryFn: () => {
      const qs = studentId ? `?student_id=${studentId}` : "";
      return apiClient<DayViewResponse>(
        `/v1/planning/calendar/day/${date}${qs}`,
      );
    },
    enabled: !!date,
    staleTime: 1000 * 60,
  });
}

export function useWeekView(date: string, studentId?: string) {
  return useQuery({
    queryKey: ["planning", "calendar", "week", date, studentId],
    queryFn: () => {
      const qs = studentId ? `?student_id=${studentId}` : "";
      return apiClient<WeekViewResponse>(
        `/v1/planning/calendar/week/${date}${qs}`,
      );
    },
    enabled: !!date,
    staleTime: 1000 * 60,
  });
}

// ─── Schedule Item Queries ──────────────────────────────────────────────────

export function useScheduleItem(id: string | undefined) {
  return useQuery({
    queryKey: ["planning", "schedule-items", id],
    queryFn: () =>
      apiClient<ScheduleItemResponse>(
        `/v1/planning/schedule-items/${id ?? ""}`,
      ),
    enabled: !!id,
  });
}

export function useScheduleItems(params?: {
  start_date?: string;
  end_date?: string;
  student_id?: string;
  category?: ScheduleCategory;
  is_completed?: boolean;
}) {
  return useQuery({
    queryKey: ["planning", "schedule-items", params],
    queryFn: () => {
      const searchParams = new URLSearchParams();
      if (params?.start_date)
        searchParams.set("start_date", params.start_date);
      if (params?.end_date) searchParams.set("end_date", params.end_date);
      if (params?.student_id)
        searchParams.set("student_id", params.student_id);
      if (params?.category) searchParams.set("category", params.category);
      if (params?.is_completed !== undefined)
        searchParams.set("is_completed", String(params.is_completed));
      const qs = searchParams.toString();
      return apiClient<ScheduleItemListResponse>(
        `/v1/planning/schedule-items${qs ? `?${qs}` : ""}`,
      );
    },
    staleTime: 1000 * 60,
  });
}

// ─── Schedule Item Mutations ────────────────────────────────────────────────

export function useCreateScheduleItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateScheduleItemInput) =>
      apiClient<{ id: string }>("/v1/planning/schedule-items", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
    },
  });
}

export function useUpdateScheduleItem(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateScheduleItemInput) =>
      apiClient<void>(`/v1/planning/schedule-items/${id}`, {
        method: "PATCH",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
    },
  });
}

export function useDeleteScheduleItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/planning/schedule-items/${id}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
    },
  });
}

export function useCompleteScheduleItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiClient<void>(`/v1/planning/schedule-items/${id}/complete`, {
        method: "PATCH",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
    },
  });
}

export function useLogAsActivity(itemId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: LogAsActivityInput) =>
      apiClient<{ activity_id: string }>(
        `/v1/planning/schedule-items/${itemId}/log`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
      void queryClient.invalidateQueries({ queryKey: ["learn"] });
    },
  });
}

// ─── Schedule Template Queries & Mutations ──────────────────────────────────

export function useScheduleTemplates() {
  return useQuery({
    queryKey: ["planning", "templates"],
    queryFn: () =>
      apiClient<ScheduleTemplate[]>("/v1/planning/templates"),
    staleTime: 1000 * 60 * 5,
  });
}

export function useCreateScheduleTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateScheduleTemplateInput) =>
      apiClient<ScheduleTemplate>("/v1/planning/templates", {
        method: "POST",
        body,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["planning", "templates"],
      });
    },
  });
}

export function useApplyScheduleTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      templateId,
      ...body
    }: ApplyScheduleTemplateInput & { templateId: string }) =>
      apiClient<{ items_created: number }>(
        `/v1/planning/templates/${templateId}/apply`,
        { method: "POST", body },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["planning"] });
    },
  });
}

export function useDeleteScheduleTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (templateId: string) =>
      apiClient<void>(`/v1/planning/templates/${templateId}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["planning", "templates"],
      });
    },
  });
}

// ─── Schedule Export ─────────────────────────────────────────────────────────

export function useExportSchedule() {
  return useMutation({
    mutationFn: (body: ExportScheduleInput) =>
      apiClient<ScheduleExportResponse>("/v1/planning/schedule/export", {
        method: "POST",
        body,
      }),
  });
}
