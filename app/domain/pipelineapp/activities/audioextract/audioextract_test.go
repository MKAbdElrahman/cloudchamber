package audioextract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandArgsKeepsOneTrack(t *testing.T) {
	args := commandArgs("in.mp4", "out.wav", 1, false)

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-i in.mp4") || !strings.Contains(joined, "-ar 16000 -ac 1 -acodec pcm_s16le -y out.wav") {
		t.Fatalf("args = %v", args)
	}
	if strings.Contains(joined, "filter_complex") || strings.Contains(joined, "silenceremove") {
		t.Fatalf("single track should not filter, args = %v", args)
	}
}

func TestCommandArgsMixesEveryTrack(t *testing.T) {
	args := commandArgs("in.mp4", "out.wav", 2, false)
	joined := strings.Join(args, " ")

	want := "-filter_complex [0:a:0][0:a:1]amix=inputs=2:duration=longest:normalize=0[mix] -map [mix]"
	if !strings.Contains(joined, want) {
		t.Fatalf("args = %v", args)
	}
	if strings.Contains(joined, "silenceremove") {
		t.Fatalf("enhance is off, args = %v", args)
	}
}

func TestCommandArgsTrimsSilenceAfterTheMix(t *testing.T) {
	args := commandArgs("in.mp4", "out.wav", 2, true)
	joined := strings.Join(args, " ")

	want := "[0:a:0][0:a:1]amix=inputs=2:duration=longest:normalize=0[mix];[mix]silenceremove=stop_periods=-1:stop_duration=0.7:stop_threshold=-45dB[out]"
	if !strings.Contains(joined, want) || !strings.Contains(joined, "-map [out]") {
		t.Fatalf("args = %v", args)
	}
}

func TestCommandArgsTrimsSilenceOnOneTrack(t *testing.T) {
	args := commandArgs("in.wav", "out.wav", 1, true)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-af silenceremove=stop_periods=-1:stop_duration=0.7:stop_threshold=-45dB") {
		t.Fatalf("args = %v", args)
	}
}

func TestCountStreamsReadsAudioLines(t *testing.T) {
	stderr := `
Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'two-tracks.mp4':
  Stream #0:0[0x1](und): Video: h264
  Stream #0:1[0x2](und): Audio: aac (LC)
  Stream #0:2[0x3](und): Audio: aac (LC)
`
	if got := countStreams(stderr); got != 2 {
		t.Fatalf("tracks = %d", got)
	}
	if got := countStreams("no streams"); got != 0 {
		t.Fatalf("tracks = %d", got)
	}
}

func TestMediaDurationReadsTheContainerClock(t *testing.T) {
	stderr := "Input #0, mov,mp4, from 'clip.mp4':\n  Duration: 00:01:30.50, start: 0.000000, bitrate: 100 kb/s\n"
	if got := mediaDuration(stderr); got != 90*time.Second+500*time.Millisecond {
		t.Fatalf("duration = %s", got)
	}
	if got := mediaDuration("no clock"); got != 0 {
		t.Fatalf("duration = %s", got)
	}
}

func TestReadProgressReportsAFraction(t *testing.T) {
	var got Progress
	readProgress(strings.NewReader("out_time_us=500000\n"), time.Second, func(sample Progress) {
		got = sample
	})
	if got.Fraction != 0.5 || got.Remaining != 0 {
		t.Fatalf("progress = %+v", got)
	}
}

func TestReadProgressEstimatesRemainingFromSpeed(t *testing.T) {
	var got Progress
	readProgress(strings.NewReader("out_time_us=500000\nspeed=2x\n"), time.Second, func(sample Progress) {
		got = sample
	})
	if got.Fraction != 0.5 || got.Remaining != 250*time.Millisecond {
		t.Fatalf("progress = %+v", got)
	}
}

func TestIsNativeWAV(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "native.wav")
	if err := os.WriteFile(native, wavHeader(16000, 1, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := IsNativeWAV(native)
	if err != nil || !ok {
		t.Fatalf("native = %v err=%v", ok, err)
	}

	other := filepath.Join(dir, "other.wav")
	if err := os.WriteFile(other, wavHeader(8000, 1, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err = IsNativeWAV(other)
	if err != nil || ok {
		t.Fatalf("8 kHz = %v err=%v", ok, err)
	}

	video := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(video, []byte("not a wav"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err = IsNativeWAV(video)
	if err != nil || ok {
		t.Fatalf("mp4 = %v err=%v", ok, err)
	}
}

func TestExtractCopiesANativeWAV(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.wav")
	dst := filepath.Join(dir, "out.wav")
	body := wavHeader(16000, 1, 16)
	if err := os.WriteFile(src, body, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Extract(t.Context(), "/no/such/ffmpeg", src, dst, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("copied %d bytes", len(got))
	}
}

func TestExtractConvertsWithFFmpeg(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "args.log")
	bin := filepath.Join(dir, "ffmpeg")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$LOG"
case " $* " in
*" -hide_banner "*)
  echo "  Stream #0:0: Video: h264" >&2
  echo "  Stream #0:1: Audio: aac" >&2
  echo "  Stream #0:2: Audio: aac" >&2
  exit 1
  ;;
esac
out=
for a in "$@"; do out=$a; done
printf 'RIFF' > "$out"
exit 0
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "clip.mp4")
	dst := filepath.Join(dir, "out.wav")
	if err := os.WriteFile(src, []byte("mp4"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LOG", logPath)
	if err := Extract(t.Context(), bin, src, dst, false); err != nil {
		t.Fatal(err)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(logged)
	if !strings.Contains(text, "amix=inputs=2:duration=longest:normalize=0[mix]") {
		t.Fatalf("log = %s", text)
	}
	if !strings.Contains(text, "-ar 16000 -ac 1 -acodec pcm_s16le") {
		t.Fatalf("log = %s", text)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "RIFF" {
		t.Fatalf("output %q err=%v", got, err)
	}
}

func TestExtractReportsMissingFFmpeg(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(src, []byte("mp4"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Extract(t.Context(), filepath.Join(dir, "missing"), src, filepath.Join(dir, "out.wav"), false)
	if err == nil || !strings.Contains(err.Error(), "ffmpeg not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestExtractReportsFFmpegFailure(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ffmpeg")
	script := `#!/bin/sh
case " $* " in
*" -hide_banner "*)
  echo "  Stream #0:0: Audio: mp3" >&2
  exit 1
  ;;
esac
echo "boom: unsupported codec" >&2
exit 1
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "clip.mp3")
	if err := os.WriteFile(src, []byte("mp3"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Extract(t.Context(), bin, src, filepath.Join(dir, "out.wav"), false)
	if err == nil || !strings.Contains(err.Error(), "unsupported codec") {
		t.Fatalf("err = %v", err)
	}
}

func wavHeader(rate, channels, bits uint32) []byte {
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	header[16] = 16
	header[20] = 1
	header[22] = byte(channels)
	header[24] = byte(rate)
	header[25] = byte(rate >> 8)
	block := channels * bits / 8
	byteRate := rate * block
	header[28] = byte(byteRate)
	header[32] = byte(block)
	header[34] = byte(bits)
	copy(header[36:], "data")
	return header
}
