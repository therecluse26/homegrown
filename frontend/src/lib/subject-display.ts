/**
 * Maps subject slugs to human-readable display names.
 *
 * Used across activity-log, feed, marketplace listing detail, and other
 * places where subjects are shown to the user.
 */

const SUBJECT_DISPLAY_MAP: Record<string, string> = {
  math: "Math",
  mathematics: "Mathematics",
  science: "Science",
  english: "English",
  language_arts: "Language Arts",
  reading: "Reading",
  writing: "Writing",
  history: "History",
  social_studies: "Social Studies",
  geography: "Geography",
  art: "Art",
  music: "Music",
  physical_education: "Physical Education",
  pe: "PE",
  health: "Health",
  foreign_language: "Foreign Language",
  spanish: "Spanish",
  french: "French",
  latin: "Latin",
  computer_science: "Computer Science",
  technology: "Technology",
  bible: "Bible",
  theology: "Theology",
  nature_study: "Nature Study",
  handicrafts: "Handicrafts",
  living_books: "Living Books",
  narration: "Narration",
  copywork: "Copywork",
  dictation: "Dictation",
  picture_study: "Picture Study",
  composer_study: "Composer Study",
  general: "General",
  elective: "Elective",
  logic: "Logic",
  rhetoric: "Rhetoric",
  grammar: "Grammar",
  classical_languages: "Classical Languages",
  philosophy: "Philosophy",
  practical_life: "Practical Life",
  handwork: "Handwork",
};

/**
 * Convert a subject slug (e.g. "language_arts") to a display name
 * (e.g. "Language Arts"). Falls back to title-casing the slug if unknown.
 */
export function subjectDisplayName(slug: string): string {
  const mapped = SUBJECT_DISPLAY_MAP[slug];
  if (mapped) return mapped;

  // Fallback: replace underscores with spaces and title-case each word
  return slug
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}
