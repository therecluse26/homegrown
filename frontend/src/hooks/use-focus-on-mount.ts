import { useEffect, type RefObject } from "react";

export function useFocusOnMount(ref: RefObject<HTMLElement | null>) {
  useEffect(() => {
    ref.current?.focus({ preventScroll: false });
  }, [ref]);
}
