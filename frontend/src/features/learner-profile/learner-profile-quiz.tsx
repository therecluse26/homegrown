import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { Spinner } from "@/components/ui";
import { QuizQuestionScreen, type QuizQuestion } from "./quiz-question-screen";
import { ProfileSummary } from "./profile-summary";
import { useProfile, useSubmitQuiz } from "./use-learner-profile";
import { useStudents } from "@/hooks/use-family";
import type { QuizAnswer } from "./use-learner-profile";

// ─── Static question definitions ─────────────────────────────────────────────
// 12 scored questions (Q1-Q12) + 2 interest questions (Q13-Q14, multi-select)
// Values: Likert 4-point [0, 0.33, 0.67, 1.0] — computed per dimension server-side
// See specs/domains/18-learner-profile.md §5.1

const LIKERT = [0, 0.33, 0.67, 1.0] as const;

function likertQ(
  id: string,
  parentText: string,
  studentText: string,
  low: string,
  midLow: string,
  midHigh: string,
  high: string,
): { parent: QuizQuestion; student: QuizQuestion } {
  const opts = [
    { id: "a", label: low },
    { id: "b", label: midLow },
    { id: "c", label: midHigh },
    { id: "d", label: high },
  ];
  return {
    parent: { id, text: parentText, options: opts },
    student: { id, text: studentText, options: opts },
  };
}

type QuestionPair = { parent: QuizQuestion; student: QuizQuestion };

const SCORED_PAIRS: QuestionPair[] = [
  // activity_format (Q1, Q2)
  likertQ(
    "q1",
    "When learning something new, {name} prefers to:",
    "When you learn something new, you prefer to:",
    "Watch or listen carefully first",
    "Read about it, then try",
    "Jump in and experiment",
    "Make or build something",
  ),
  likertQ(
    "q2",
    "Which best describes how {name} absorbs information?",
    "Which best describes how you absorb information?",
    "Visual — charts, diagrams, videos",
    "Reading or hearing explanations",
    "Hands-on practice",
    "Creating and making",
  ),
  // session_length (Q3, Q4)
  likertQ(
    "q3",
    "How long can {name} focus on one subject before needing a break?",
    "How long can you focus on one subject before needing a break?",
    "10–15 minutes",
    "20–30 minutes",
    "45–60 minutes",
    "An hour or more",
  ),
  likertQ(
    "q4",
    "{name} prefers to study in:",
    "You prefer to study in:",
    "Many short bursts throughout the day",
    "Two or three medium blocks",
    "One long focused session",
    "Whatever feels right that day",
  ),
  // motivation (Q5, Q6)
  likertQ(
    "q5",
    "What motivates {name} most?",
    "What motivates you most?",
    "Mastering a skill perfectly",
    "Getting good marks or praise",
    "Finishing a project",
    "Exploring new ideas freely",
  ),
  likertQ(
    "q6",
    "When {name} struggles, they're more likely to:",
    "When you struggle, you're more likely to:",
    "Keep practicing until it clicks",
    "Ask for help right away",
    "Try a different approach",
    "Move on and come back later",
  ),
  // solo_collaborative (Q7, Q8)
  likertQ(
    "q7",
    "{name} learns best:",
    "You learn best:",
    "Entirely on their own",
    "Mostly solo with occasional check-ins",
    "In a small group",
    "With a partner or side by side",
  ),
  likertQ(
    "q8",
    "When working on a project, {name} prefers:",
    "When working on a project, you prefer:",
    "Having full control on their own",
    "Leading and delegating",
    "Equal collaboration",
    "Following someone else's lead",
  ),
  // structure (Q9, Q10)
  likertQ(
    "q9",
    "{name} works best with:",
    "You work best with:",
    "Very clear step-by-step instructions",
    "An outline with some flexibility",
    "Loose guidance and room to explore",
    "Complete freedom to self-direct",
  ),
  likertQ(
    "q10",
    "When given a new topic, {name} tends to:",
    "When given a new topic, you tend to:",
    "Follow a structured curriculum closely",
    "Mix curriculum with their own research",
    "Dive into their own questions first",
    "Pursue it in their own way entirely",
  ),
  // outdoor_kinesthetic (Q11, Q12)
  likertQ(
    "q11",
    "How important is physical movement in {name}'s learning?",
    "How important is physical movement in your learning?",
    "Not important — they learn fine sitting",
    "Nice to have occasionally",
    "Helps a lot",
    "Essential — they learn by moving",
  ),
  likertQ(
    "q12",
    "How often does {name} enjoy learning outdoors?",
    "How often do you enjoy learning outdoors?",
    "Rarely — prefers indoors",
    "Once in a while",
    "Often",
    "As much as possible",
  ),
];

const INTEREST_OPTIONS = [
  { id: "art", label: "Art & drawing" },
  { id: "science", label: "Science & experiments" },
  { id: "reading", label: "Reading for fun" },
  { id: "building", label: "Building & making" },
  { id: "music", label: "Music" },
  { id: "outdoors", label: "Outdoor exploration" },
  { id: "math", label: "Math & puzzles" },
  { id: "writing", label: "Writing & storytelling" },
  { id: "history", label: "History & geography" },
  { id: "cooking", label: "Cooking & baking" },
];

const INTEREST_Q13: QuizQuestion = {
  id: "q13",
  text: "Which topics does {name} enjoy most? (pick all that apply)",
  options: INTEREST_OPTIONS,
  multiSelect: true,
};

const INTEREST_Q14_STUDENT: QuizQuestion = {
  id: "q14",
  text: "What do you enjoy most? (pick as many as you like)",
  options: INTEREST_OPTIONS,
  multiSelect: true,
};

const LIKERT_VALUE_MAP: Record<string, number> = {
  a: LIKERT[0],
  b: LIKERT[1],
  c: LIKERT[2],
  d: LIKERT[3],
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

function interpolateName(text: string, name: string): string {
  return text.replace(/\{name\}/g, name);
}

function currentYear(): number {
  return new Date().getFullYear();
}

function isChildRespondent(birthYear: number | undefined): boolean {
  if (!birthYear) return false;
  return currentYear() - birthYear >= 8;
}

// ─── Total questions: 12 scored + 1 interest = 13 ─────────────────────────

const TOTAL_QUESTIONS = SCORED_PAIRS.length + 1; // 13

// ─── Component ───────────────────────────────────────────────────────────────

type Props = {
  /** Pass when embedding as a child. Omit when used as a route page (reads from URL params). */
  studentId?: string;
  /** Called when quiz is complete and summary should be shown at a higher level */
  onComplete?: () => void;
  /** When true, show the privacy note and page chrome (standalone quiz page mode) */
  standalone?: boolean;
};

export function LearnerProfileQuiz({ studentId: propStudentId, onComplete, standalone = false }: Props) {
  const params = useParams<{ studentId: string }>();
  const studentId = propStudentId ?? params.studentId;
  const navigate = useNavigate();
  const students = useStudents();
  const student = students.data?.find((s) => s.id === studentId);
  const profileQuery = useProfile(studentId);
  const submitQuiz = useSubmitQuiz(studentId);

  const [questionIndex, setQuestionIndex] = useState(0);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  // scoredAnswers[i] = selected option id (or null if skipped) for scored Q i (0-11)
  const [scoredAnswers, setScoredAnswers] = useState<(string | null)[]>(
    Array(SCORED_PAIRS.length).fill(null),
  );
  const [interests, setInterests] = useState<string[]>([]);
  const [phase, setPhase] = useState<"quiz" | "summary">("quiz");

  const studentName = student?.display_name ?? "your child";

  if (students.isLoading || profileQuery.isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    );
  }

  // If profile already exists, jump straight to summary
  if (phase === "summary" || (profileQuery.data && phase !== "quiz")) {
    const profile = submitQuiz.data ?? profileQuery.data;
    return (
      <ProfileSummary
        studentName={studentName}
        summaryText={profile?.summary_text ?? ""}
        interests={profile?.interests ?? []}
        onRetake={() => {
          setPhase("quiz");
          setQuestionIndex(0);
          setSelectedIds([]);
          setScoredAnswers(Array(SCORED_PAIRS.length).fill(null));
          setInterests([]);
        }}
        onEditInterests={() => {
          setPhase("quiz");
          setQuestionIndex(SCORED_PAIRS.length); // jump to Q13
          setSelectedIds([...(profile?.interests ?? [])]);
        }}
      />
    );
  }

  const isChildMode = isChildRespondent(student?.birth_year);
  const variant = isChildMode ? "student" : "parent";

  // Build the current question
  function buildQuestion(): QuizQuestion {
    if (questionIndex < SCORED_PAIRS.length) {
      const pair = SCORED_PAIRS[questionIndex]!;
      const raw = isChildMode ? pair.student : pair.parent;
      return {
        ...raw,
        text: interpolateName(raw.text, studentName),
      };
    }
    // Final question (index 12) = a single interest multi-select. INTEREST_Q13
    // and INTEREST_Q14_STUDENT are two phrasings of the SAME question; exactly
    // one is shown, chosen by respondent (parent vs. child). [board: HOM-88]
    const raw = isChildMode ? INTEREST_Q14_STUDENT : INTEREST_Q13;
    return { ...raw, text: interpolateName(raw.text, studentName) };
  }

  const currentQuestion = buildQuestion();
  const isScoredQuestion = questionIndex < SCORED_PAIRS.length;

  function handleSelect(optionId: string) {
    if (isScoredQuestion) {
      // Single-select for scored questions
      setSelectedIds([optionId]);
    } else {
      // Multi-select for interests
      setSelectedIds((prev) =>
        prev.includes(optionId)
          ? prev.filter((id) => id !== optionId)
          : [...prev, optionId],
      );
    }
  }

  function commitCurrent() {
    if (isScoredQuestion) {
      const next = [...scoredAnswers];
      next[questionIndex] = selectedIds[0] ?? null;
      setScoredAnswers(next);
    } else {
      setInterests(selectedIds);
    }
  }

  function goNext() {
    commitCurrent();
    if (questionIndex < TOTAL_QUESTIONS - 1) {
      const nextIdx = questionIndex + 1;
      // Pre-populate selectedIds from existing answers when going back/forth
      if (nextIdx < SCORED_PAIRS.length) {
        const saved = scoredAnswers[nextIdx];
        setSelectedIds(saved ? [saved] : []);
      } else {
        setSelectedIds([...interests]);
      }
      setQuestionIndex(nextIdx);
    } else {
      // Pass selectedIds directly: setInterests() is async and hasn't propagated
      // yet when handleSubmit() runs in the same tick.
      handleSubmit(isScoredQuestion ? undefined : selectedIds);
    }
  }

  function goBack() {
    if (questionIndex === 0) return;
    const prevIdx = questionIndex - 1;
    if (prevIdx < SCORED_PAIRS.length) {
      const saved = scoredAnswers[prevIdx];
      setSelectedIds(saved ? [saved] : []);
    } else {
      setSelectedIds([...interests]);
    }
    setQuestionIndex(prevIdx);
  }

  function skipCurrent() {
    if (isScoredQuestion) {
      const next = [...scoredAnswers];
      next[questionIndex] = null;
      setScoredAnswers(next);
    }
    if (questionIndex < TOTAL_QUESTIONS - 1) {
      setSelectedIds([]);
      setQuestionIndex((i) => i + 1);
    } else {
      handleSubmit();
    }
  }

  function handleSubmit(pendingInterests?: string[]) {
    const answers: QuizAnswer[] = scoredAnswers
      .map((optId, i) => ({
        question_id: i + 1,
        value: optId !== null ? LIKERT_VALUE_MAP[optId] : undefined,
      }))
      .filter((a) => a.value !== undefined) as QuizAnswer[];

    const respondent = isChildMode ? "child" : "parent";
    // pendingInterests overrides the interests state when setInterests() hasn't
    // propagated yet (e.g. called in the same tick as the state update).
    const finalInterests = pendingInterests ?? interests;

    submitQuiz.mutate(
      { answers, respondent, interests: finalInterests.length ? finalInterests : undefined },
      {
        onSuccess: () => {
          setPhase("summary");
          onComplete?.();
          if (standalone && studentId) {
            void navigate(`/students/${studentId}/learner-profile`);
          }
        },
      },
    );
  }

  const isLastQuestion = questionIndex === TOTAL_QUESTIONS - 1;
  // For last scored question or interest question: enable Next when something is selected
  // Allow next even without selection (can skip)
  const hasSelection = selectedIds.length > 0;

  const quizBody = (
    <div>
      {submitQuiz.isError && (
        <div
          role="alert"
          className="mb-4 rounded-lg bg-error-container px-4 py-3 type-body-sm text-on-error-container"
        >
          Something went wrong. Please try again.
        </div>
      )}
      <QuizQuestionScreen
        variant={variant}
        question={currentQuestion}
        questionIndex={questionIndex}
        totalQuestions={TOTAL_QUESTIONS}
        selectedIds={selectedIds}
        studentName={studentName}
        onSelect={handleSelect}
        onSkip={skipCurrent}
        onBack={goBack}
        onNext={isLastQuestion || hasSelection ? goNext : () => {}}
      />
      {submitQuiz.isPending && (
        <div className="mt-4 flex justify-center">
          <Spinner />
        </div>
      )}
    </div>
  );

  if (standalone) {
    return (
      <div className="max-w-2xl mx-auto py-6 px-4">
        <div className="mb-4 rounded-lg bg-surface-container-low px-4 py-3 type-body-sm text-on-surface-variant shadow-ghost-border">
          Only your family can see this. Learner Profiles are never shared or used for ads.{" "}
          <Link to="/legal/privacy" className="underline hover:text-on-surface transition-colors">
            Learn more
          </Link>
        </div>
        {quizBody}
      </div>
    );
  }

  return quizBody;
}
