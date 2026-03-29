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
  Icon,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { useStudentSession } from "@/hooks/use-student-session";
import { useVideoDef, useVideoProgress, useUpdateVideoProgress } from "@/hooks/use-video";

export function StudentVideo() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { videoId } = useParams<{ videoId: string }>();
  const { session: studentSession } = useStudentSession();
  const studentId = studentSession?.studentId ?? "";

  const { data: videoDef, isPending: defLoading } = useVideoDef(videoId ?? "");
  const { data: progress } = useVideoProgress(studentId, videoId ?? "");
  const updateProgress = useUpdateVideoProgress(studentId);

  const videoRef = useRef<HTMLVideoElement>(null);
  const progressTimer = useRef<ReturnType<typeof setInterval>>(undefined);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [captionsOn, setCaptionsOn] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    if (progress?.last_position_seconds && videoRef.current) {
      videoRef.current.currentTime = progress.last_position_seconds;
    }
  }, [progress?.last_position_seconds]);

  const saveProgress = useCallback(() => {
    if (!videoId || !videoRef.current) return;
    const v = videoRef.current;
    updateProgress.mutate({
      video_def_id: videoId,
      watched_seconds: Math.round(v.currentTime),
      last_position_seconds: Math.round(v.currentTime),
      completed: v.duration > 0 && v.currentTime / v.duration > 0.9,
    });
  }, [videoId, updateProgress]);

  useEffect(() => {
    if (isPlaying) {
      progressTimer.current = setInterval(saveProgress, 10000);
    } else if (progressTimer.current) {
      clearInterval(progressTimer.current);
    }
    return () => {
      if (progressTimer.current) clearInterval(progressTimer.current);
    };
  }, [isPlaying, saveProgress]);

  function formatTime(s: number) {
    return `${Math.floor(s / 60)}:${Math.floor(s % 60).toString().padStart(2, "0")}`;
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

  const pct = duration > 0 ? Math.round((currentTime / duration) * 100) : 0;

  if (defLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-[400px]" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => { saveProgress(); void navigate(-1); }}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1"><FormattedMessage id="common.back" /></span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {videoDef?.title ?? ""}
        </h1>
      </div>

      <Card className="overflow-hidden p-0">
        <div className="relative bg-inverse-surface">
          <video
            ref={videoRef}
            src={videoDef?.video_url}
            className="w-full aspect-video"
            onPlay={() => setIsPlaying(true)}
            onPause={() => setIsPlaying(false)}
            onTimeUpdate={() => setCurrentTime(videoRef.current?.currentTime ?? 0)}
            onLoadedMetadata={() => setDuration(videoRef.current?.duration ?? 0)}
            onEnded={() => { setIsPlaying(false); saveProgress(); }}
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
          </video>
        </div>
        <div className="p-4 space-y-3">
          <input
            type="range"
            min={0}
            max={Math.floor(duration) || 0}
            value={Math.floor(currentTime)}
            onChange={(e) => {
              if (videoRef.current) {
                videoRef.current.currentTime = Number(e.target.value);
                setCurrentTime(Number(e.target.value));
              }
            }}
            className="w-full accent-primary"
            aria-label={intl.formatMessage({ id: "video.seek" })}
          />
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => { if (videoRef.current) { if (isPlaying) { videoRef.current.pause(); } else { void videoRef.current.play(); } } }}
                className="p-2 rounded-full hover:bg-surface-container-high text-on-surface touch-target"
                aria-label={intl.formatMessage({ id: isPlaying ? "video.pause" : "video.play" })}
              >
                <Icon icon={isPlaying ? Pause : Play} size="md" aria-hidden />
              </button>
              <button
                type="button"
                onClick={() => { if (videoRef.current) { videoRef.current.muted = !isMuted; setIsMuted(!isMuted); } }}
                className="p-2 rounded-full hover:bg-surface-container-high text-on-surface touch-target"
                aria-label={intl.formatMessage({ id: isMuted ? "video.unmute" : "video.mute" })}
              >
                <Icon icon={isMuted ? VolumeX : Volume2} size="md" aria-hidden />
              </button>
              {videoDef?.caption_tracks && videoDef.caption_tracks.length > 0 && (
                <button
                  type="button"
                  onClick={handleToggleCaptions}
                  className="p-2 rounded-full hover:bg-surface-container-high text-on-surface touch-target"
                  aria-label={intl.formatMessage({ id: captionsOn ? "video.captionsOff" : "video.captionsOn" })}
                >
                  <Icon icon={captionsOn ? CaptionsOff : Captions} size="md" aria-hidden />
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
                onClick={() => videoRef.current?.requestFullscreen()}
                className="p-2 rounded-full hover:bg-surface-container-high text-on-surface touch-target"
                aria-label={intl.formatMessage({ id: "video.fullscreen" })}
              >
                <Icon icon={Maximize} size="md" aria-hidden />
              </button>
            </div>
          </div>
        </div>
      </Card>

      <Card className="bg-surface-container-low">
        <div className="flex items-center justify-between mb-2">
          <span className="type-label-md text-on-surface-variant">
            <FormattedMessage id="video.progress" />
          </span>
          <span className="type-label-sm text-on-surface-variant">{pct}%</span>
        </div>
        <ProgressBar value={pct} />
      </Card>
    </div>
  );
}
