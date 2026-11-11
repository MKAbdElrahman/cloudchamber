/**
 * One inset for a page Layout. Header, content, and panels inherit it.
 * Do not set a different padding on LayoutContent — that shifts the content line.
 *
 * The shell fills the viewport. These measures cap the line inside a region:
 * 960 for a library, 640 for settings and reading. A region with a side panel
 * does not take a measure, because that pulls the panel into the same strip.
 */
export const WORKSPACE_CONTENT_PADDING = 6 as const

/** Library and other content pages. */
export const PAGE_CONTENT_WIDTH = 960

/** Settings, and a document when it is the only region. */
export const TEXT_CONTENT_WIDTH = 640
