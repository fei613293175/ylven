import { describe, expect, it } from 'vitest'
import { canAccess } from './router.js'
describe('admin RBAC', () => { it('denies without admin permission', () => expect(canAccess('/admin/018', [])).toBe(false)); it('allows registered page', () => expect(canAccess('/admin/018', ['admin:read'])).toBe(true)) })
