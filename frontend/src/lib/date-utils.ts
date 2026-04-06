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
