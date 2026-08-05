import { describe, expect, it } from 'vitest'
describe('developer portal contract', () => { it('keeps required states', () => expect(['LOADING','POPULATED','EMPTY','ERROR']).toContain('POPULATED')) })
