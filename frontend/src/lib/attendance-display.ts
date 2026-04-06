/**
 * Attendance status display formatting.
 *
 * Maps raw enum values (present_full, present_partial, absent, not_applicable)
 * to user-friendly display strings.
 */

const ATTENDANCE_DISPLAY: Record<string, string> = {
  present_full: "Present (Full Day)",
  present_partial: "Present (Partial)",
  absent: "Absent",
  not_applicable: "Excused / N/A",
};

/**
 * Convert an attendance status enum value to a human-readable label.
 * Falls back to replacing underscores with spaces if the value is unknown.
 */
export function attendanceDisplayName(status: string): string {
  const mapped = ATTENDANCE_DISPLAY[status];
  if (mapped) return mapped;
  return status.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}
