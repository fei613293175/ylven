import { describe, expect, it } from 'vitest'
import { catalogPayload, createCatalogDraft } from './catalog.js'

describe('P04 model catalog forms', () => {
  it('preserves versions for conflict-safe provider edits', () => {
    const draft = createCatalogDraft('provider', { id: 'openai', name: 'OpenAI', enabled: true, sort_order: 2, version: 7 })
    expect(catalogPayload('provider', draft)).toEqual({ id: 'openai', name: 'OpenAI', enabled: true, sort_order: 2, version: 7 })
  })

  it('parses only object-shaped upstream parameters', () => {
    const draft = createCatalogDraft('reasoning', { model_id: 'reasoner', profile_id: 'deep', label: '深度', upstream_parameters: { reasoning_effort: 'high' }, version: 3 })
    expect(catalogPayload('reasoning', draft).upstream_parameters).toEqual({ reasoning_effort: 'high' })
    draft.upstream_parameters_text = '[]'
    expect(() => catalogPayload('reasoning', draft)).toThrow('JSON 对象')
  })

  it('requires immutable probe evidence', () => {
    const draft = createCatalogDraft('probe', { model_id: 'reasoner', capability_id: 'vision', probe_source: 'contract-test' })
    expect(() => catalogPayload('probe', draft)).toThrow('证据引用')
  })
})
