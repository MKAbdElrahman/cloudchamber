/**
 * One inset for a page Layout. Header, content, and panels inherit it.
 * Do not set a different padding on LayoutContent — that shifts the content line.
 *
 * Width is not capped here. The page fills the AppShell main region so a wide
 * screen uses the space beside the side nav, and a narrow screen uses the
 * full width once the nav has moved into the mobile drawer. A form that
 * should stay on a short line sets its own maxWidth.
 */
export const WORKSPACE_CONTENT_PADDING = 6 as const
