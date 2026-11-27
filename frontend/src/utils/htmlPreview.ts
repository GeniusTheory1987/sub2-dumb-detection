/**
 * Builds an iframe srcdoc document that runs inline animation scripts while
 * keeping the generated page isolated: no same-origin access, no network.
 */
const PREVIEW_CSP = [
  "default-src 'none'",
  'img-src data: blob:',
  'media-src data: blob:',
  "style-src 'unsafe-inline'",
  "script-src 'unsafe-inline' 'unsafe-eval'",
  'font-src data:',
  "connect-src 'none'",
  "form-action 'none'",
  "base-uri 'none'"
].join('; ')

const PREVIEW_CSP_META = `<meta http-equiv="Content-Security-Policy" content="${PREVIEW_CSP}">`

export function buildPreviewDocument(html: string): string {
  const content = html.trim()
  if (!content) return ''

  const head = /<head(?:\s[^>]*)?>/i.exec(content)
  if (head) {
    const at = head.index + head[0].length
    return content.slice(0, at) + PREVIEW_CSP_META + content.slice(at)
  }

  const htmlTag = /<html(?:\s[^>]*)?>/i.exec(content)
  if (htmlTag) {
    const at = htmlTag.index + htmlTag[0].length
    return content.slice(0, at) + `<head>${PREVIEW_CSP_META}</head>` + content.slice(at)
  }

  return PREVIEW_CSP_META + content
}
