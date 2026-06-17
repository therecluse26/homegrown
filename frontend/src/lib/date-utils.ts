/**
 * Parse a date-only string ("YYYY-MM-DD") as local noon to avoid
 * timezone-shift issues where UTC midnight displays as the previous day
 * in negative-offset timezones.
 *
 * For full datetime strings (containing "T"), passes through to `new Date()`.
 */
export function parseLocalDate(dateStr: string): Date {
  // If it's a date-only string (no time component), append noon local time
  if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
    return new Date(`${dateStr}T12:00:00`);
  }
  return new Date(dateStr);
}

/**
 * Format a datetime string as a human-readable relative time ("2h ago", "3d ago").
 * Falls back to locale date string for dates older than 7 days.
 */
export function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}
