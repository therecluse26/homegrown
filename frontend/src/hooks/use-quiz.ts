import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/api/client";

// ─── Types ──────────────────────────────────────────────────────────────────

export type QuizSessionStatus =
  | "not_started"
  | "in_progress"
  | "submitted"
  | "scored";

export interface QuizQuestionResponse {
  question_id: string;
  sort_order: number;
  points: number;
  question_type: string;
  content: string;
  answer_data?: Record<string, unknown>;
  auto_scorable: boolean;
}

export interface QuizDefResponse {
  id: string;
  publisher_id: string;
  title: string;
  description?: string;
  subject_tags: string[];
  methodology_id?: string;
  time_limit_minutes?: number;
  passing_score_percent: number;
  shuffle_questions: boolean;
  show_correct_after: boolean;
  question_count: number;
  created_at: string;
}

export interface QuizDefDetailResponse extends QuizDefResponse {
  questions: QuizQuestionResponse[];
}

export interface QuizSessionResponse {
  id: string;
  student_id: string;
  quiz_def_id: string;
  status: QuizSessionStatus;
  started_at?: string;
  submitted_at?: string;
  scored_at?: string;
  score?: number;
  max_score?: number;
  passed?: boolean;
  answers?: Record<string, unknown>;
  created_at: string;
}

export interface QuestionScore {
  question_id: string;
  points_awarded: number;
  feedback?: string;
}

// ─── Quiz Definition Queries ────────────────────────────────────────────────

export function useQuizDef(id: string) {
  return useQuery({
    queryKey: ["learning", "quiz-defs", id],
    queryFn: () =>
      apiClient<QuizDefDetailResponse>(`/v1/learning/quiz-defs/${id}`),
    enabled: !!id,
  });
}

// ─── Quiz Session Queries ───────────────────────────────────────────────────

export function useQuizSession(studentId: string, sessionId: string) {
  return useQuery({
    queryKey: ["learning", "quiz-sessions", studentId, sessionId],
    queryFn: () =>
      apiClient<QuizSessionResponse>(
        `/v1/learning/students/${studentId}/quiz-sessions/${sessionId}`,
      ),
    enabled: !!studentId && !!sessionId,
  });
}

// ─── Quiz Session Mutations ─────────────────────────────────────────────────

export function useStartQuizSession(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cmd: { quiz_def_id: string }) =>
      apiClient<QuizSessionResponse>(
        `/v1/learning/students/${studentId}/quiz-sessions`,
        { method: "POST", body: cmd },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: ["learning", "quiz-sessions", studentId],
      });
    },
  });
}

export function useUpdateQuizSession(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      sessionId,
      ...cmd
    }: {
      sessionId: string;
      answers?: Record<string, unknown>;
      submit?: boolean;
    }) =>
      apiClient<QuizSessionResponse>(
        `/v1/learning/students/${studentId}/quiz-sessions/${sessionId}`,
        { method: "PATCH", body: cmd },
      ),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({
        queryKey: [
          "learning",
          "quiz-sessions",
          studentId,
          vars.sessionId,
        ],
      });
    },
  });
}

export function useScoreQuiz(studentId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      sessionId,
      scores,
    }: {
      sessionId: string;
      scores: QuestionScore[];
    }) =>
      apiClient<QuizSessionResponse>(
        `/v1/learning/students/${studentId}/quiz-sessions/${sessionId}/score`,
        { method: "POST", body: { scores } },
      ),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({
        queryKey: [
          "learning",
          "quiz-sessions",
          studentId,
          vars.sessionId,
        ],
      });
      void qc.invalidateQueries({
        queryKey: ["learning", "progress", studentId],
      });
    },
  });
}
