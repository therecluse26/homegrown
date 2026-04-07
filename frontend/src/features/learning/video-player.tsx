import { useState, useRef, useCallback, useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate } from "react-router";
import {
  ArrowLeft,
  Play,
  Pause,
  Volume2,
  VolumeX,
  Maximize,
  CheckCircle,
  Captions,
  CaptionsOff,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import { useVideoDef, useVideoProgress, useUpdateVideoProgress } from "@/hooks/use-video";

// ─── Main component ──────────────────────────────────────────────────────────

export function VideoPlayer() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { videoId } = useParams<{ videoId: string }>();
  const { data: students } = useStudents();

  const studentId = students?.[0]?.id ?? "";

  const { data: videoDef, isPending: defLoading } = useVideoDef(videoId ?? "");
  const { data: progress, isPending: progressLoading } = useVideoProgress(
    studentId,
    videoId ?? "",
  );
  const updateProgress = useUpdateVideoProgress(studentId);

  const videoRef = useRef<HTMLVideoElement>(null);
  const progressTimer = useRef<ReturnType<typeof setInterval>>(undefined);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [captionsOn, setCaptionsOn] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  // Restore last position
  useEffect(() => {
    if (progress?.last_position_seconds && videoRef.current) {
      videoRef.current.currentTime = progress.last_position_seconds;
      setCurrentTime(progress.last_position_seconds);
    }
  }, [progress?.last_position_seconds]);

  // Periodic progress save (every 10 seconds while playing)
  const saveProgress = useCallback(() => {
    if (!videoId || !videoRef.current) return;
    const video = videoRef.current;
    const completed = video.duration > 0 && video.currentTime / video.duration > 0.9;
    updateProgress.mutate({
      video_def_id: videoId,
      watched_seconds: Math.round(video.currentTime),
      last_position_seconds: Math.round(video.currentTime),
      completed,
    });
  }, [videoId, updateProgress]);

  useEffect(() => {
    if (isPlaying) {
      progressTimer.current = setInterval(saveProgress, 10000);
    } else {
      if (progressTimer.current) clearInterval(progressTimer.current);
    }
    return () => {
      if (progressTimer.current) clearInterval(progressTimer.current);
    };
  }, [isPlaying, saveProgress]);

  function handlePlayPause() {
    if (!videoRef.current) return;
    if (isPlaying) {
      videoRef.current.pause();
    } else {
      void videoRef.current.play();
    }
  }

  function handleToggleMute() {
    if (!videoRef.current) return;
    videoRef.current.muted = !isMuted;
    setIsMuted(!isMuted);
  }

  function handleFullscreen() {
    if (!videoRef.current) return;
    void videoRef.current.requestFullscreen();
  }

  function handleToggleCaptions() {
    if (!videoRef.current) return;
    const tracks = videoRef.current.textTracks;
    const next = !captionsOn;
    for (let i = 0; i < tracks.length; i++) {
      const t = tracks[i];
      if (t) t.mode = next ? "showing" : "hidden";
    }
    setCaptionsOn(next);
  }

  function handleTimeUpdate() {
    if (!videoRef.current) return;
    setCurrentTime(videoRef.current.currentTime);
  }

  function handleLoadedMetadata() {
    if (!videoRef.current) return;
    setDuration(videoRef.current.duration);
  }

  function handleSeek(e: React.ChangeEvent<HTMLInputElement>) {
    if (!videoRef.current) return;
    const time = Number(e.target.value);
    videoRef.current.currentTime = time;
    setCurrentTime(time);
  }

  function handleVideoEnd() {
    setIsPlaying(false);
    saveProgress();
  }

  function formatTime(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, "0")}`;
  }

  const progressPct =
    duration > 0 ? Math.round((currentTime / duration) * 100) : 0;

  if (!videoId) {
    return (
      <EmptyState message={intl.formatMessage({ id: "video.notFound" })} />
    );
  }

  if (defLoading || progressLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-[400px]" />
      </div>
    );
  }

  if (!videoDef) {
    return (
      <EmptyState message={intl.formatMessage({ id: "video.notFound" })} />
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => {
            saveProgress();
            void navigate("/learning");
          }}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {videoDef?.title ?? ""}
        </h1>
      </div>

      {/* Video container */}
      <Card className="overflow-hidden p-0">
        <div className="relative bg-inverse-surface">
          <video
            ref={videoRef}
            src={videoDef?.video_url}
            className="w-full aspect-video"
            onPlay={() => setIsPlaying(true)}
            onPause={() => setIsPlaying(false)}
            onTimeUpdate={handleTimeUpdate}
            onLoadedMetadata={handleLoadedMetadata}
            onEnded={handleVideoEnd}
            playsInline
            crossOrigin="anonymous"
          >
            {videoDef?.caption_tracks?.map((track, i) => (
              <track
                key={track.srclang}
                kind={track.kind}
                src={track.src}
                srcLang={track.srclang}
                label={track.label}
                default={i === 0}
              />
            ))}
            <FormattedMessage id="video.unsupported" />
          </video>
        </div>

        {/* Controls */}
        <div className="p-4 space-y-3">
          {/* Seek bar */}
          <input
            type="range"
            min={0}
            max={Math.floor(duration) || 0}
            value={Math.floor(currentTime)}
            onChange={handleSeek}
            className="w-full accent-primary"
            aria-label={intl.formatMessage({ id: "video.seek" })}
          />

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handlePlayPause}
                className="p-2 rounded-full hover:bg-surface-container-high transition-colors text-on-surface touch-target"
                aria-label={intl.formatMessage({
                  id: isPlaying ? "video.pause" : "video.play",
                })}
              >
                <Icon icon={isPlaying ? Pause : Play} size="md" aria-hidden />
              </button>

              <button
                type="button"
                onClick={handleToggleMute}
                className="p-2 rounded-full hover:bg-surface-container-high transition-colors text-on-surface touch-target"
                aria-label={intl.formatMessage({
                  id: isMuted ? "video.unmute" : "video.mute",
                })}
              >
                <Icon
                  icon={isMuted ? VolumeX : Volume2}
                  size="md"
                  aria-hidden
                />
              </button>

              {videoDef?.caption_tracks && videoDef.caption_tracks.length > 0 && (
                <button
                  type="button"
                  onClick={handleToggleCaptions}
                  className="p-2 rounded-full hover:bg-surface-container-high transition-colors text-on-surface touch-target"
                  aria-label={intl.formatMessage({
                    id: captionsOn ? "video.captionsOff" : "video.captionsOn",
                  })}
                >
                  <Icon
                    icon={captionsOn ? CaptionsOff : Captions}
                    size="md"
                    aria-hidden
                  />
                </button>
              )}

              <span className="type-label-sm text-on-surface-variant">
                {formatTime(currentTime)} / {formatTime(duration)}
              </span>
            </div>

            <div className="flex items-center gap-2">
              {progress?.completed && (
                <span className="inline-flex items-center gap-1 type-label-sm text-primary">
                  <Icon icon={CheckCircle} size="xs" aria-hidden />
                  <FormattedMessage id="video.completed" />
                </span>
              )}

              <button
                type="button"
                onClick={handleFullscreen}
                className="p-2 rounded-full hover:bg-surface-container-high transition-colors text-on-surface touch-target"
                aria-label={intl.formatMessage({ id: "video.fullscreen" })}
              >
                <Icon icon={Maximize} size="md" aria-hidden />
              </button>
            </div>
          </div>
        </div>
      </Card>

      {/* Progress bar */}
      <Card className="bg-surface-container-low">
        <div className="flex items-center justify-between mb-2">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage id="video.progress" />
          </span>
          <span className="type-label-sm text-on-surface-variant">
            {progressPct}%
          </span>
        </div>
        <ProgressBar value={progressPct} />
      </Card>

      {/* Description */}
      {videoDef?.description && (
        <Card>
          <h2 className="type-title-md text-on-surface font-semibold mb-2">
            <FormattedMessage id="video.about" />
          </h2>
          <p className="type-body-md text-on-surface-variant">
            {videoDef.description}
          </p>
        </Card>
      )}
    </div>
  );
}
