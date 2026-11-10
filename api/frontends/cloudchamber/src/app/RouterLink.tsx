import { forwardRef, type AnchorHTMLAttributes, type ReactNode } from 'react'
import { Link as ReactRouterLink } from 'react-router-dom'

type RouterLinkProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  href?: string
  to?: string
  children?: ReactNode
}

/** Bridges Astryx `Link` / `LinkProvider` to React Router client navigation. */
export const RouterLink = forwardRef<HTMLAnchorElement, RouterLinkProps>(function RouterLink(
  { href, to, children, ...rest },
  ref,
) {
  const destination = to ?? href ?? '/'
  return (
    <ReactRouterLink ref={ref} to={destination} {...rest}>
      {children}
    </ReactRouterLink>
  )
})
