// Package workflow contains deterministic CloudChamber pipeline workflow definitions.
package workflow

const (
	// TaskQueueWorkflow processes workflow tasks for ProcessSourceWorkflow.
	TaskQueueWorkflow = "cloudchamber-workflow"
	// TaskQueueMedia processes source and audio activities.
	TaskQueueMedia = "cloudchamber-media"
	// TaskQueueFrames processes key-frame activities.
	TaskQueueFrames = "cloudchamber-frames"
	// TaskQueueASR processes transcription activities.
	TaskQueueASR = "cloudchamber-asr"
	// TaskQueueRender processes composition and export activities.
	TaskQueueRender = "cloudchamber-render"
)
