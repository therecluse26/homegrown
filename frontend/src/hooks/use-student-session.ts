import { createContext, useContext } from "react";

// ─── Types ──────────────────────────────────────────────────────────────────

export interface StudentSession {
  /** The active student's ID */
  studentId: string;
  /** The active student's display name */
  studentName: string;
  /** When the session was started (ISO string) */
  startedAt: string;
  /** When the session is scheduled to end (ISO string, if duration preset used) */
  expiresAt?: string;
}

export interface StudentSessionContextValue {
  /** Current active student session, if any */
  session: StudentSession | null;
  /** Start a student session */
  startSession: (session: StudentSession) => void;
  /** End the current student session */
  endSession: () => void;
  /** Whether a student session is active */
  isActive: boolean;
  /** Minutes remaining in session, if expiry is set */
  minutesRemaining: number | null;
}

// ─── Context ─────────────────────────────────────────────────────────────────

export const StudentSessionContext =
  createContext<StudentSessionContextValue | null>(null);

// ─── Hook ────────────────────────────────────────────────────────────────────

export function useStudentSession(): StudentSessionContextValue {
  const ctx = useContext(StudentSessionContext);
  if (!ctx) {
    // Provide a no-op default when used outside the student shell
    return {
      session: null,
      startSession: () => {},
      endSession: () => {},
      isActive: false,
      minutesRemaining: null,
    };
  }
  return ctx;
}
