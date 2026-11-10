# CloudChamber

CloudChamber converts a video or audio file into a readable document by combining the transcript with key visual frames.

A user can create multiple projects. Each project is isolated from every other project: its media, jobs, and documents stay inside that project and are not shared with or mixed into another project.

Inside a project, the user can start multiple jobs. Each job converts exactly one media file into exactly one document. Converting several files means starting one job per file.

## Projects and jobs

| Scope | Rule |
| --- | --- |
| User | Owns many projects. |
| Project | Isolated from other projects. Holds many jobs. |
| Job | Accepts one media file and produces one document. |

## Core Happy Path

The path below is one job inside one project.

1. **Input**: The job receives one video or audio file (for example, an uploaded recording).
2. **Extract audio**: CloudChamber extracts the audio track from that file.
3. **Transcribe**: The audio is converted into a timestamped transcript.
4. **Extract key frames**: For a video, CloudChamber selects meaningful visual frames (such as slides, diagrams, or important scene changes) and aligns them with relevant transcript moments.
5. **Compose document**: The transcript and key frames are organized into one structured, readable document with sections and timestamps.
6. **Export**: The job writes one document in the chosen format, such as **Markdown** or a **Typst-generated PDF**. That document belongs to the same project as the job.

## Example

Project: `Lecture notes`

Job input: one recording, `week-03.mp4`

Job output: one document for that recording, containing the transcript, sectioned content, timestamps, and selected key frames, exported as Markdown or PDF.

A second recording in the same project is a second job with its own document. A different project cannot see either job or either document.
