import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ───────────────────────────────────────────────────────────────────

// User Management
export interface AdminUserSummary {
  family_id: string;
  family_name: string;
  primary_parent_email: string;
  parent_count: number;
  student_count: number;
  subscription_tier: string;
  account_status: string;
  created_at: string;
  last_active_at?: string;
}

export interface AdminParentInfo {
  id: string;
  display_name: string;
  email: string;
  is_primary: boolean;
}

export interface AdminStudentInfo {
  id: string;
  display_name: string;
  grade_level?: string;
}

export interface AdminSubscriptionInfo {
  tier: string;
  status: string;
  expires_at?: string;
}

export interface ModerationActionSummary {
  action: string;
  reason: string;
  created_at: string;
}

export interface UserActivitySummary {
  last_login_at?: string;
  activity_count_7d: number;
}

export interface AdminUserDetail {
  family: {
    id: string;
    name: string;
    account_status: string;
    created_at: string;
    last_active_at?: string;
  };
  parents: AdminParentInfo[];
  students: AdminStudentInfo[];
  subscription?: AdminSubscriptionInfo;
  moderation_history: ModerationActionSummary[];
  recent_activity: UserActivitySummary;
}

// Feature Flags
export interface FeatureFlag {
  id: string;
  key: string;
  description: string;
  enabled: boolean;
  rollout_percentage?: number;
  allowed_family_ids: string[];
  created_by: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
}

// System Health
export interface ComponentHealth {
  name: string;
  status: string;
  latency_ms?: number;
  details?: string;
}

export interface SystemHealthResponse {
  status: string;
  components: ComponentHealth[];
  checked_at: string;
}

export interface QueueStatus {
  name: string;
  pending: number;
  processing: number;
  completed_24h: number;
  failed_24h: number;
}

export interface JobStatusResponse {
  queues: QueueStatus[];
  dead_letter_count: number;
}

export interface DeadLetterJob {
  id: string;
  queue: string;
  job_type: string;
  payload: unknown;
  error_message: string;
  failed_at: string;
  retry_count: number;
}

// Moderation Queue
export interface ModerationQueueItem {
  id: string;
  content_type: string;
  content_id: string;
  family_id: string;
  reason: string;
  status: string;
  details: unknown;
  created_at: string;
}

// Audit Log
export interface AuditLogEntry {
  id: string;
  admin_id: string;
  admin_email?: string;
  action: string;
  target_type: string;
  target_id?: string;
  details: unknown;
  created_at: string;
}

// Methodology Config
export interface MethodologyConfig {
  slug: string;
  display_name: string;
  enabled: boolean;
  settings: unknown;
  updated_at: string;
}

// Lifecycle
export interface DeletionSummary {
  family_id: string;
  family_name: string;
  requested_at: string;
  scheduled_at: string;
}

export interface RecoverySummary {
  id: string;
  family_id: string;
  family_name: string;
  requested_at: string;
  reason: string;
}

// Paginated wrapper
export interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
  has_more: boolean;
}

// Safety domain types (used by admin moderation + user appeals)
export interface ReportResponse {
  id: string;
  target_type: string;
  category: string;
  status: string;
  created_at: string;
}

export interface AdminReportResponse {
  id: string;
  reporter_family_id: string;
  target_type: string;
  target_id: string;
  target_family_id?: string;
  category: string;
  description?: string;
  priority: string;
  status: string;
  assigned_admin_id?: string;
  resolved_at?: string;
  created_at: string;
}

export interface ModActionResponse {
  id: string;
  admin_id: string;
  target_family_id: string;
  target_parent_id?: string;
  action_type: string;
  reason: string;
  report_id?: string;
  suspension_days?: number;
  suspension_expires_at?: string;
  created_at: string;
}

export interface AccountStatusResponse {
  status: string;
  suspended_at?: string;
  suspension_expires_at?: string;
  suspension_reason?: string;
}

export interface AppealResponse {
  id: string;
  action_id: string;
  status: string;
  appeal_text: string;
  resolution_text?: string;
  resolved_at?: string;
  created_at: string;
}

export interface AdminAppealResponse {
  id: string;
  family_id: string;
  action_id: string;
  original_action: ModActionResponse;
  appeal_text: string;
  status: string;
  assigned_admin_id?: string;
  resolution_text?: string;
  resolved_at?: string;
  created_at: string;
}

export interface DashboardStats {
  pending_reports: number;
  critical_reports: number;
  unreviewed_flags: number;
  pending_appeals: number;
  active_suspensions: number;
  active_bans: number;
  reports_last_24h: number;
  actions_last_24h: number;
}

// ─── Admin: User Management ─────────────────────────────────────────────────

export function useAdminSearchUsers(params: {
  q?: string;
  status?: string;
  subscription?: string;
}) {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.status) qs.set("status", params.status);
  if (params.subscription) qs.set("subscription", params.subscription);
  const query = qs.toString();
  return useQuery({
    queryKey: ["admin", "users", params],
    queryFn: () =>
      apiClient<PaginatedResponse<AdminUserSummary>>(
        `/v1/admin/users${query ? `?${query}` : ""}`,
      ),
  });
}

export function useAdminUserDetail(userId: string | undefined) {
  return useQuery({
    queryKey: ["admin", "user", userId],
    queryFn: () => apiClient<AdminUserDetail>(`/v1/admin/users/${userId}`),
    enabled: !!userId,
  });
}

export function useAdminSuspendUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
      apiClient<void>(`/v1/admin/users/${userId}/suspend`, {
        method: "POST",
        body: { reason },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useAdminBanUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
      apiClient<void>(`/v1/admin/users/${userId}/ban`, {
        method: "POST",
        body: { reason },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useAdminUnsuspendUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      apiClient<void>(`/v1/admin/users/${userId}/unsuspend`, {
        method: "POST",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

// ─── Admin: System Health ────────────────────────────────────────────────────

export function useSystemHealth() {
  return useQuery({
    queryKey: ["admin", "health"],
    queryFn: () => apiClient<SystemHealthResponse>("/v1/admin/system/health"),
    refetchInterval: 30_000,
  });
}

export function useJobStatus() {
  return useQuery({
    queryKey: ["admin", "jobs"],
    queryFn: () => apiClient<JobStatusResponse>("/v1/admin/system/jobs"),
  });
}

// ─── Admin: Moderation Queue ─────────────────────────────────────────────────

export function useAdminModerationQueue(status?: string) {
  const qs = status ? `?status=${status}` : "";
  return useQuery({
    queryKey: ["admin", "moderation", "queue", status],
    queryFn: () =>
      apiClient<PaginatedResponse<ModerationQueueItem>>(
        `/v1/admin/moderation/queue${qs}`,
      ),
  });
}

export function useAdminTakeModerationAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      itemId,
      action,
      reason,
    }: {
      itemId: string;
      action: string;
      reason?: string;
    }) =>
      apiClient<void>(`/v1/admin/moderation/queue/${itemId}/action`, {
        method: "POST",
        body: { action, reason },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "moderation"] });
    },
  });
}

// ─── Admin: Audit Log ────────────────────────────────────────────────────────

export function useAdminAuditLog(params: {
  admin_id?: string;
  action?: string;
  target_type?: string;
  from_date?: string;
  to_date?: string;
}) {
  const qs = new URLSearchParams();
  if (params.admin_id) qs.set("admin_id", params.admin_id);
  if (params.action) qs.set("action", params.action);
  if (params.target_type) qs.set("target_type", params.target_type);
  if (params.from_date) qs.set("from_date", params.from_date);
  if (params.to_date) qs.set("to_date", params.to_date);
  const query = qs.toString();
  return useQuery({
    queryKey: ["admin", "audit", params],
    queryFn: () =>
      apiClient<PaginatedResponse<AuditLogEntry>>(
        `/v1/admin/audit${query ? `?${query}` : ""}`,
      ),
  });
}

// ─── Admin: Feature Flags ────────────────────────────────────────────────────

export function useFeatureFlags() {
  return useQuery({
    queryKey: ["admin", "flags"],
    queryFn: () => apiClient<FeatureFlag[]>("/v1/admin/flags"),
  });
}

export function useUpdateFeatureFlag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      key,
      ...body
    }: {
      key: string;
      enabled?: boolean;
      rollout_percentage?: number;
      whitelisted_families?: string[];
    }) =>
      apiClient<FeatureFlag>(`/v1/admin/flags/${key}`, {
        method: "PATCH",
        body: body,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "flags"] });
    },
  });
}

export function useCreateFeatureFlag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      key: string;
      description: string;
      enabled: boolean;
      rollout_percentage?: number;
    }) =>
      apiClient<FeatureFlag>("/v1/admin/flags", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "flags"] });
    },
  });
}

// ─── Admin: Methodology Config ───────────────────────────────────────────────

export interface MethodologyTool {
  key: string;
  label: string;
  description: string;
  enabled: boolean;
}

export interface MethodologyConfigFull {
  slug: string;
  display_name: string;
  philosophy: string;
  tools: MethodologyTool[];
}

export function useMethodologyConfigs() {
  return useQuery({
    queryKey: ["admin", "methodologies"],
    queryFn: () =>
      apiClient<MethodologyConfigFull[]>("/v1/admin/methodologies"),
  });
}

export function useUpdateMethodologyConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      slug,
      ...body
    }: {
      slug: string;
      display_name?: string;
      philosophy?: string;
      tools?: MethodologyTool[];
    }) =>
      apiClient<MethodologyConfigFull>(
        `/v1/admin/methodologies/${slug}`,
        {
          method: "PATCH",
          body: body,
        },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "methodologies"] });
    },
  });
}

// ─── Admin: Safety Reports ───────────────────────────────────────────────────

export function useAdminReports(params: {
  status?: string;
  priority?: string;
  category?: string;
}) {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.priority) qs.set("priority", params.priority);
  if (params.category) qs.set("category", params.category);
  const query = qs.toString();
  return useQuery({
    queryKey: ["admin", "safety", "reports", params],
    queryFn: () =>
      apiClient<PaginatedResponse<AdminReportResponse>>(
        `/v1/admin/safety/reports${query ? `?${query}` : ""}`,
      ),
  });
}

export function useAdminSafetyActions() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      target_family_id: string;
      action_type: string;
      reason: string;
      report_id?: string;
      suspension_days?: number;
    }) =>
      apiClient<ModActionResponse>("/v1/admin/safety/actions", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "safety"] });
    },
  });
}

// ─── Admin: Appeals ──────────────────────────────────────────────────────────

export function useAdminAppeals(status?: string) {
  const qs = status ? `?status=${status}` : "";
  return useQuery({
    queryKey: ["admin", "safety", "appeals", status],
    queryFn: () =>
      apiClient<PaginatedResponse<AdminAppealResponse>>(
        `/v1/admin/safety/appeals${qs}`,
      ),
  });
}

export function useAdminResolveAppeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      appealId,
      status,
      resolution_text,
    }: {
      appealId: string;
      status: string;
      resolution_text: string;
    }) =>
      apiClient<void>(`/v1/admin/safety/appeals/${appealId}`, {
        method: "PATCH",
        body: { status, resolution_text },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "safety", "appeals"] });
    },
  });
}

// ─── Admin: Safety Dashboard ─────────────────────────────────────────────────

export function useAdminSafetyDashboard() {
  return useQuery({
    queryKey: ["admin", "safety", "dashboard"],
    queryFn: () =>
      apiClient<DashboardStats>("/v1/admin/safety/dashboard"),
  });
}

// ─── User-facing Safety hooks ────────────────────────────────────────────────

export function useMyReports() {
  return useQuery({
    queryKey: ["safety", "my-reports"],
    queryFn: () => apiClient<ReportResponse[]>("/v1/safety/reports"),
  });
}

export function useSubmitReport() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: {
      target_type: string;
      target_id: string;
      category: string;
      description?: string;
    }) =>
      apiClient<ReportResponse>("/v1/safety/reports", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["safety", "my-reports"] });
    },
  });
}

export function useAccountStatus() {
  return useQuery({
    queryKey: ["safety", "account-status"],
    queryFn: () =>
      apiClient<AccountStatusResponse>("/v1/safety/account-status"),
  });
}

export function useMyAppeals() {
  return useQuery({
    queryKey: ["safety", "my-appeals"],
    queryFn: () => apiClient<AppealResponse[]>("/v1/safety/appeals"),
  });
}

export function useSubmitAppeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: { action_id: string; appeal_text: string }) =>
      apiClient<AppealResponse>("/v1/safety/appeals", {
        method: "POST",
        body: cmd,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["safety", "my-appeals"] });
    },
  });
}
