import { useState, useCallback, useEffect, type KeyboardEvent } from "react";
import { createPortal } from "react-dom";
import { X, ChevronLeft, ChevronRight } from "lucide-react";
import { Icon } from "./icon";

type GalleryImage = {
  src: string;
  alt: string;
};

type ImageGalleryProps = {
  images: GalleryImage[];
  /** Grid columns (default 3) */
  columns?: 2 | 3 | 4;
  className?: string;
};

const gridClasses = {
  2: "grid-cols-2",
  3: "grid-cols-2 sm:grid-cols-3",
  4: "grid-cols-2 sm:grid-cols-3 lg:grid-cols-4",
} as const;

export function ImageGallery({
  images,
  columns = 3,
  className = "",
}: ImageGalleryProps) {
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

  const closeLightbox = useCallback(() => setLightboxIndex(null), []);

  const goNext = useCallback(() => {
    setLightboxIndex((i) =>
      i !== null ? (i + 1) % images.length : null,
    );
  }, [images.length]);

  const goPrev = useCallback(() => {
    setLightboxIndex((i) =>
      i !== null ? (i - 1 + images.length) % images.length : null,
    );
  }, [images.length]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") closeLightbox();
      if (e.key === "ArrowRight") goNext();
      if (e.key === "ArrowLeft") goPrev();
    },
    [closeLightbox, goNext, goPrev],
  );

  // Prevent body scroll when lightbox open
  useEffect(() => {
    if (lightboxIndex !== null) {
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = "";
      };
    }
  }, [lightboxIndex]);

  const currentImage = lightboxIndex !== null ? images[lightboxIndex] : null;

  return (
    <>
      <div className={`grid ${gridClasses[columns]} gap-2 ${className}`}>
        {images.map((image, index) => (
          <button
            key={image.src}
            type="button"
            className="group relative aspect-square overflow-hidden rounded-lg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            onClick={() => setLightboxIndex(index)}
          >
            <img
              src={image.src}
              alt={image.alt}
              className="h-full w-full object-cover transition-transform duration-[var(--duration-normal)] group-hover:scale-105"
              loading="lazy"
            />
          </button>
        ))}
      </div>

      {currentImage &&
        createPortal(
          // eslint-disable-next-line jsx-a11y/no-static-element-interactions
          <div
            className="fixed inset-0 z-modal flex items-center justify-center bg-scrim/80"
            onKeyDown={handleKeyDown}
          >
            <button
              className="absolute right-4 top-4 rounded-full bg-inverse-surface/50 p-2 text-inverse-on-surface hover:bg-inverse-surface/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
              onClick={closeLightbox}
              aria-label="Close lightbox"
              autoFocus
            >
              <Icon icon={X} size="lg" />
            </button>

            {images.length > 1 && (
              <>
                <button
                  className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-inverse-surface/50 p-2 text-inverse-on-surface hover:bg-inverse-surface/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                  onClick={goPrev}
                  aria-label="Previous image"
                >
                  <Icon icon={ChevronLeft} size="lg" />
                </button>
                <button
                  className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-inverse-surface/50 p-2 text-inverse-on-surface hover:bg-inverse-surface/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
                  onClick={goNext}
                  aria-label="Next image"
                >
                  <Icon icon={ChevronRight} size="lg" />
                </button>
              </>
            )}

            <img
              src={currentImage.src}
              alt={currentImage.alt}
              className="max-h-[85vh] max-w-[90vw] rounded-lg object-contain"
            />
          </div>,
          document.body,
        )}
    </>
  );
}
