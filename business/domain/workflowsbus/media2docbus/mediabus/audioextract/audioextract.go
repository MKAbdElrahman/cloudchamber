// Package audioextract turns a recording into the WAV the speech step reads.
// The conversion follows vibe-server: 16 kHz, mono, 16-bit PCM. A file that is
// already that WAV is copied. Every other file goes through ffmpeg. When the
// container holds more than one audio track, those tracks are mixed so a later
// speaker is not dropped.
package audioextract

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound means the ffmpeg binary is not on PATH and was not configured.
var ErrNotFound = errors.New("ffmpeg not found")

const silenceRemove = "silenceremove=stop_periods=-1:stop_duration=0.7:stop_threshold=-45dB"

// Extract writes a 16 kHz mono 16-bit PCM WAV to dst.
// enhance trims long silences after the mix, matching vibe's enhance_audio option.
func Extract(ctx context.Context, ffmpegPath, src, dst string, enhance bool) error {
	return ExtractProgress(ctx, ffmpegPath, src, dst, enhance, nil)
}

// Progress is one sample from ffmpeg. Fraction is encoded media time divided by
// the container duration, in the range 0 to 1. Remaining is wall time still to
// run when ffmpeg reported a speed; it is zero when that estimate is unknown.
type Progress struct {
	Fraction  float64
	Remaining time.Duration
}

// ExtractProgress is Extract with a sample while ffmpeg is encoding.
// report may be nil. A file copied without ffmpeg does not report a sample.
func ExtractProgress(ctx context.Context, ffmpegPath, src, dst string, enhance bool, report func(Progress)) error {
	native, err := IsNativeWAV(src)
	if err != nil {
		return err
	}
	if native && !enhance {
		return copyFile(src, dst)
	}
	return convert(ctx, ffmpegPath, src, dst, enhance, report)
}

// IsNativeWAV reports whether path is already 16 kHz mono 16-bit PCM.
// A file that is not a WAV returns false.
func IsNativeWAV(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var hdr [12]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return false, nil
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return false, nil
	}

	for {
		var chunk [8]byte
		if _, err := io.ReadFull(f, chunk[:]); err != nil {
			return false, nil
		}
		id := string(chunk[0:4])
		size := binary.LittleEndian.Uint32(chunk[4:8])
		if id == "fmt " {
			if size < 16 {
				return false, nil
			}
			var fmtChunk [16]byte
			if _, err := io.ReadFull(f, fmtChunk[:]); err != nil {
				return false, nil
			}
			format := binary.LittleEndian.Uint16(fmtChunk[0:2])
			channels := binary.LittleEndian.Uint16(fmtChunk[2:4])
			rate := binary.LittleEndian.Uint32(fmtChunk[4:8])
			bits := binary.LittleEndian.Uint16(fmtChunk[14:16])
			return format == 1 && channels == 1 && rate == 16_000 && bits == 16, nil
		}
		skip := int64(size)
		if size%2 == 1 {
			skip++
		}
		if _, err := f.Seek(skip, io.SeekCurrent); err != nil {
			return false, nil
		}
	}
}

func convert(ctx context.Context, ffmpegPath, src, dst string, enhance bool, report func(Progress)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bin, err := resolveFFmpeg(ffmpegPath)
	if err != nil {
		return err
	}

	tracks, duration := probe(ctx, bin, src)
	if err := ctx.Err(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, bin, commandArgs(src, dst, tracks, enhance)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	readProgress(stdout, duration, report)
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg WAV conversion failed: %s", msg)
	}
	return nil
}

func commandArgs(input, output string, tracks int, enhance bool) []string {
	args := []string{"-i", input}
	switch {
	case tracks > 1:
		var graph strings.Builder
		for i := range tracks {
			fmt.Fprintf(&graph, "[0:a:%d]", i)
		}
		fmt.Fprintf(&graph, "amix=inputs=%d:duration=longest:normalize=0[mix]", tracks)
		label := "mix"
		if enhance {
			graph.WriteString(";[mix]" + silenceRemove + "[out]")
			label = "out"
		}
		args = append(args, "-filter_complex", graph.String(), "-map", "["+label+"]")
	case enhance:
		args = append(args, "-af", silenceRemove)
	}
	args = append(args, "-progress", "pipe:1", "-nostats", "-ar", "16000", "-ac", "1", "-acodec", "pcm_s16le", "-y", output)
	return args
}

// probe asks ffmpeg which audio streams the container holds and how long it is.
// ffmpeg -i exits non-zero after printing the streams. A failure to start counts as one track.
func probe(ctx context.Context, ffmpeg, input string) (int, time.Duration) {
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-i", input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			return 1, 0
		}
	}
	text := string(out)
	return max(countStreams(text), 1), mediaDuration(text)
}

// mediaDuration reads the Duration field ffmpeg prints for the container.
func mediaDuration(stderr string) time.Duration {
	_, rest, ok := strings.Cut(stderr, "Duration: ")
	if !ok {
		return 0
	}
	stamp, _, _ := strings.Cut(rest, ",")
	return parseClock(strings.TrimSpace(stamp))
}

func parseClock(stamp string) time.Duration {
	parts := strings.Split(stamp, ":")
	if len(parts) != 3 {
		return 0
	}
	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0
	}
	return time.Duration(hours*float64(time.Hour) + minutes*float64(time.Minute) + seconds*float64(time.Second))
}

// readProgress drains ffmpeg's progress pipe. out_time_us and out_time_ms are both microseconds.
// speed is media time divided by wall time, so the remaining estimate is the unencoded media time divided by speed.
func readProgress(r io.Reader, duration time.Duration, report func(Progress)) {
	if report == nil || duration <= 0 {
		_, _ = io.Copy(io.Discard, r)
		return
	}
	total := float64(duration.Microseconds())
	var (
		speed    float64
		out      time.Duration
		fraction float64
		haveOut  bool
	)
	emit := func() {
		if !haveOut {
			return
		}
		sample := Progress{Fraction: fraction}
		if speed > 0 && duration > out {
			sample.Remaining = time.Duration(float64(duration-out) / speed)
		}
		report(sample)
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		key, val, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		switch key {
		case "out_time_us", "out_time_ms":
			parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil || parsed <= 0 || total <= 0 {
				continue
			}
			fraction = parsed / total
			if fraction > 1 {
				fraction = 1
			}
			out = time.Duration(parsed * float64(time.Microsecond))
			haveOut = true
			emit()
		case "speed":
			parsed, ok := parseSpeed(val)
			if !ok {
				speed = 0
				continue
			}
			speed = parsed
			emit()
		}
	}
}

func parseSpeed(val string) (float64, bool) {
	text := strings.TrimSpace(val)
	text = strings.TrimSuffix(text, "x")
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func countStreams(stderr string) int {
	n := 0
	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Stream #") && strings.Contains(line, "Audio:") {
			n++
		}
	}
	return n
}

func resolveFFmpeg(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultFFmpeg()
	}
	if path == "" {
		return "", ErrNotFound
	}
	if strings.ContainsAny(path, `/\`) {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return path, nil
	}
	found, err := exec.LookPath(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNotFound, path)
	}
	return found, nil
}

func defaultFFmpeg() string {
	if env := strings.TrimSpace(os.Getenv("FFMPEG_PATH")); env != "" {
		return env
	}
	if found, err := exec.LookPath("ffmpeg"); err == nil {
		return found
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(exe), "ffmpeg")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return ""
	}
	return candidate
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
