import { describe, expect, it } from 'vitest'
import { canAccess } from './router.js'
describe('admin RBAC', () => { it('denies without admin permission', () => expect(canAccess('/admin/018', [])).toBe(false)); it('allows registered page', () => expect(canAccess('/admin/018', ['admin:read'])).toBe(true)); it('registers P01-W05 pages', () => ['/admin/027','/admin/037','/admin/038','/admin/039','/admin/040'].forEach((path) => expect(canAccess(path, ['admin:read'])).toBe(true))) })
