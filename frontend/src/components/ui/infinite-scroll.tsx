import { useEffect, useRef, type ReactNode } from "react";
import { Spinner } from "./spinner";

type InfiniteScrollProps = {
  /** Called when the sentinel enters the viewport */
  onLoadMore: () => void;
  /** Whether more data is currently being fetched */
  loading?: boolean;
  /** Whether there are more pages to load */
  hasMore: boolean;
  children: ReactNode;
  className?: string;
};

export function InfiniteScroll({
  onLoadMore,
  loading = false,
  hasMore,
  children,
  className = "",
}: InfiniteScrollProps) {
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasMore) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry?.isIntersecting && !loading) {
          onLoadMore();
        }
      },
      { rootMargin: "200px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [onLoadMore, loading, hasMore]);

  return (
    <div className={className}>
      {children}

      <div ref={sentinelRef} className="flex justify-center py-4">
        {loading && <Spinner size="md" className="text-primary" />}
      </div>

      {hasMore && (
        <div className="sr-only" aria-live="polite">
          {loading ? "Loading more items" : ""}
        </div>
      )}
    </div>
  );
}
