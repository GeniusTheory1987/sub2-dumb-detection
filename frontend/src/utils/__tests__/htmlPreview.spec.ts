import { describe, expect, it } from 'vitest'
import { buildPreviewDocument } from '../htmlPreview'

const CSP_META = '<meta http-equiv="Content-Security-Policy"'

describe('buildPreviewDocument', () => {
  it('injects the CSP into an existing head', () => {
    const doc = buildPreviewDocument(
      '<!DOCTYPE html><html><head><title>ride</title></head><body><svg></svg></body></html>'
    )
    expect(doc).toContain(`<head>${CSP_META}`)
    expect(doc).toContain('<title>ride</title>')
  })

  it('creates a head when the document has only an html tag', () => {
    const doc = buildPreviewDocument('<html><body>pelican</body></html>')
    expect(doc).toBe(`<html><head>${CSP_META} content="default-src 'none'; img-src data: blob:; media-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline' 'unsafe-eval'; font-src data:; connect-src 'none'; form-action 'none'; base-uri 'none'"></head><body>pelican</body></html>`)
  })

  it('prepends the CSP for a fragment', () => {
    const doc = buildPreviewDocument('<svg><circle r="5"/></svg>')
    expect(doc.startsWith(CSP_META)).toBe(true)
    expect(doc).toContain('<circle r="5"/>')
  })

  it('returns an empty document for empty input', () => {
    expect(buildPreviewDocument('   ')).toBe('')
  })
})
