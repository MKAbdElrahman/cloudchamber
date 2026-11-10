import type { SVGProps } from 'react'

/**
 * Nav-sized cloud-chamber tracks (muon / electron / alpha) for {@link Icon} + {@link NavIcon}.
 * Strokes use `currentColor` so the glyph inherits NavIcon's on-accent color.
 */
export function CloudChamberTracksIcon({
  width = 24,
  height = 24,
  ...props
}: SVGProps<SVGSVGElement>) {
  return (
    <svg
      width={width}
      height={height}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden
      {...props}
    >
      <rect
        x="3"
        y="4"
        width="18"
        height="16"
        rx="3"
        stroke="currentColor"
        strokeWidth="1.25"
        opacity="0.45"
      />
      <ellipse cx="12" cy="18" rx="6" ry="2.25" fill="currentColor" opacity="0.18" />
      <path d="M6 19L17 7" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
      <path
        d="M8 15C10 12 9.5 9.5 13 7"
        stroke="currentColor"
        strokeWidth="1"
        strokeLinecap="round"
        opacity="0.75"
      />
      <path d="M8.5 14.5h4.5" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" opacity="0.85" />
    </svg>
  )
}
