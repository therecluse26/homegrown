/**
 * UX prototype page for HOM-91: Learner Profile quiz flow, profile summary,
 * and fit-badge components. Not part of the production routing tree — used
 * for visual verification and design screenshots only.
 *
 * Route: /ux/learner-profile (AppShell, authenticated)
 */
import { useState } from "react";
import { QuizQuestionScreen, type QuizQuestion } from "./quiz-question-screen";
import { ProfileSummary } from "./profile-summary";
import { ProfileNudge } from "./profile-nudge";
import { FitBadge } from "@/components/ui/fit-badge";
import { Card } from "@/components/ui";

// ── Static sample data ──────────────────────────────────────────────────────

const PARENT_QUESTION: QuizQuestion = {
  id: "pacing",
  text: "How does Maya prefer to pace through learning material?",
  options: [
    { id: "short", label: "Short, focused sessions" },
    { id: "long", label: "Long, deep dives" },
    { id: "flexible", label: "Flexible — as the mood strikes" },
    { id: "structured", label: "Structured daily schedule" },
  ],
};

const STUDENT_QUESTION: QuizQuestion = {
  id: "style",
  text: "How do YOU like to learn best?",
  options: [
    { id: "making", label: "Making & building", emoji: "🎨" },
    { id: "reading", label: "Reading & listening", emoji: "📖" },
    { id: "exploring", label: "Exploring & discovering", emoji: "🔬" },
    { id: "performing", label: "Acting & performing", emoji: "🎭" },
  ],
};

const INTERESTS_QUESTION: QuizQuestion = {
  id: "interests",
  text: "Which activities does Maya enjoy most? (pick all that apply)",
  multiSelect: true,
  options: [
    { id: "art", label: "Art & drawing" },
    { id: "science", label: "Science experiments" },
    { id: "reading", label: "Reading for fun" },
    { id: "building", label: "Building & making" },
    { id: "music", label: "Music" },
    { id: "outdoors", label: "Outdoor exploration" },
  ],
};

const SAMPLE_INTERESTS = ["🎨 Art", "🔬 Science", "📚 Books", "🎵 Music"];

const SAMPLE_SUMMARY =
  "Maya learns best with hands-on activities, short focused sessions, and prefers working on her own. She tends to explore topics through making and experimentation rather than passive reading.";

// ── Section wrapper ─────────────────────────────────────────────────────────
function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mb-12">
      <h2 className="type-label-lg text-on-surface-variant font-semibold uppercase tracking-wider mb-4 pb-2 border-b border-outline-variant">
        {title}
      </h2>
      {children}
    </section>
  );
}

// ── Prototype page ──────────────────────────────────────────────────────────
export function LearnerProfilePrototype() {
  const [parentSelected, setParentSelected] = useState<string[]>([]);
  const [studentSelected, setStudentSelected] = useState<string[]>([]);
  const [multiSelected, setMultiSelected] = useState<string[]>([]);
  const [nudgeDismissed, setNudgeDismissed] = useState(false);

  function toggleParent(id: string) {
    setParentSelected([id]);
  }
  function toggleStudent(id: string) {
    setStudentSelected([id]);
  }
  function toggleMulti(id: string) {
    setMultiSelected((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }

  return (
    <div className="max-w-2xl mx-auto py-8 px-4">
      <div className="mb-8">
        <h1 className="type-headline-md text-on-surface font-bold mb-1">
          HOM-91 — UX Prototype
        </h1>
        <p className="type-body-md text-on-surface-variant">
          Learner Profile: quiz flow, profile summary, fit-badge
        </p>
      </div>

      {/* 1. Parent-proxy quiz question */}
      <Section title="1. Quiz — Parent variant (Q3 of 12)">
        <Card>
          <QuizQuestionScreen
            variant="parent"
            question={PARENT_QUESTION}
            questionIndex={2}
            totalQuestions={12}
            selectedIds={parentSelected}
            studentName="Maya"
            onSelect={toggleParent}
            onSkip={() => {}}
            onBack={() => {}}
            onNext={() => {}}
          />
        </Card>
      </Section>

      {/* 2. Student self-report quiz question */}
      <Section title="2. Quiz — Student variant (Q3 of 8)">
        <Card>
          <QuizQuestionScreen
            variant="student"
            question={STUDENT_QUESTION}
            questionIndex={2}
            totalQuestions={8}
            selectedIds={studentSelected}
            studentName="Maya"
            onSelect={toggleStudent}
            onSkip={() => {}}
            onBack={() => {}}
            onNext={() => {}}
          />
        </Card>
      </Section>

      {/* 3. Multi-select question (interests) */}
      <Section title="3. Quiz — Multi-select question (parent variant, Q9 of 12)">
        <Card>
          <QuizQuestionScreen
            variant="parent"
            question={INTERESTS_QUESTION}
            questionIndex={8}
            totalQuestions={12}
            selectedIds={multiSelected}
            studentName="Maya"
            onSelect={toggleMulti}
            onSkip={() => {}}
            onBack={() => {}}
            onNext={() => {}}
          />
        </Card>
      </Section>

      {/* 4. Profile summary */}
      <Section title="4. Profile summary (post-quiz)">
        <Card>
          <ProfileSummary
            studentName="Maya"
            summaryText={SAMPLE_SUMMARY}
            interests={SAMPLE_INTERESTS}
            onRetake={() => {}}
            onEditInterests={() => {}}
          />
        </Card>
      </Section>

      {/* 5. Fit-badge */}
      <Section title="5. Fit-badge on a content card">
        <div className="max-w-sm">
          <Card className="overflow-hidden">
            {/* Mock image placeholder */}
            <div className="mb-3 -mx-card-padding -mt-card-padding h-36 bg-gradient-to-br from-primary-container to-secondary-container flex items-center justify-center">
              <span className="text-4xl">🌿</span>
            </div>
            <div className="px-1">
              <div className="mb-2">
                <FitBadge
                  studentName="Maya"
                  whyText="This activity is hands-on and short — matches how Maya likes to learn."
                />
              </div>
              <p className="type-title-sm text-on-surface font-semibold mb-1">
                Nature Journaling Starter Pack
              </p>
              <p className="type-body-sm text-on-surface-variant">
                Ages 6–12 · 45 min · Charlotte Mason
              </p>
            </div>
          </Card>
        </div>

        {/* Family-level variant */}
        <div className="mt-4 flex items-center gap-3">
          <FitBadge
            whyText="This resource matches multiple students in your family."
          />
          <span className="type-body-sm text-on-surface-variant">
            ← Family-level "Great match" variant
          </span>
        </div>
      </Section>

      {/* 6. Dashboard nudge */}
      <Section title="6. Dashboard nudge (skipped-quiz families)">
        {nudgeDismissed ? (
          <p className="type-body-sm text-on-surface-variant italic">
            Nudge dismissed. In production this would not re-appear in the same
            session.
          </p>
        ) : (
          <ProfileNudge
            studentName="Maya"
            onStart={() => {}}
            onDismiss={() => setNudgeDismissed(true)}
          />
        )}
      </Section>
    </div>
  );
}
