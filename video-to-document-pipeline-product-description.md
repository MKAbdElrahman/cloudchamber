# CloudChamber

CloudChamber converts a video or audio source into a readable document by combining the transcript with key visual frames.

## Core Happy Path

1. **Input**: The user provides a video or audio source (for example, an uploaded recording or a supported URL).
2. **Extract audio**: CloudChamber extracts the audio track from the source.
3. **Transcribe**: The audio is converted into a timestamped transcript.
4. **Extract key frames**: CloudChamber selects meaningful visual frames from the video (such as slides, diagrams, or important scene changes) and aligns them with relevant transcript moments.
5. **Compose document**: The transcript and key frames are organized into a structured, readable document with sections and timestamps.
6. **Export**: The user receives the output in their chosen format, such as **Markdown** or a **Typst-generated PDF**.

## Example

Input: `https://www.youtube.com/watch?v=ciRTdOIhNCc`

Output: a structured document containing the transcript, sectioned content, timestamps, and selected key frames, exported as Markdown or PDF.
