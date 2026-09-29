// @vitest-environment jsdom
import { afterEach, beforeAll, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { SettingsPage } from './SettingsPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({ api: { getConnectionSettings: vi.fn(), getSecuritySettings: vi.fn(), getDashboardSummary: vi.fn() } }))
beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', { value: () => ({ matches: false, addListener() {}, removeListener() {} }) })
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
})
afterEach(() => { cleanup(); vi.clearAllMocks() })

it('loads only connection settings and changes standard ports for TLS', async () => {
  vi.mocked(api.getConnectionSettings).mockResolvedValue({ data: { data: { host: 'db.internal', protocol: 'native', port: 9000, user: 'default', database: 'default' } } } as never)
  render(<SettingsPage />)
  await screen.findByDisplayValue('db.internal')
  expect(api.getSecuritySettings).not.toHaveBeenCalled()
  expect(api.getDashboardSummary).not.toHaveBeenCalled()
  fireEvent.click(screen.getByRole('checkbox', { name: /TLS/ }))
  await waitFor(() => expect((screen.getByRole('spinbutton') as HTMLInputElement).value).toBe('9440'))
})

it('preserves a custom port when TLS changes', async () => {
  vi.mocked(api.getConnectionSettings).mockResolvedValue({ data: { data: { host: 'db.internal', protocol: 'native', port: 19000, user: 'default', database: 'default' } } } as never)
  render(<SettingsPage />)
  await screen.findByDisplayValue('db.internal')
  fireEvent.click(screen.getByRole('checkbox', { name: /TLS/ }))
  expect((screen.getByRole('spinbutton') as HTMLInputElement).value).toBe('19000')
})
