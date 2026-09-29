// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { UsersPage } from './UsersPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({ api: {
  listUsers: vi.fn(), listDatabases: vi.fn(), getConnectionSettings: vi.fn(),
  grantPrivileges: vi.fn(), revokePrivileges: vi.fn(), alterUser: vi.fn(), createUser: vi.fn(),
} }))

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', { value: () => ({ matches: false, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {} }) })
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  const getStyle = window.getComputedStyle
  vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => getStyle(element))
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listUsers).mockResolvedValue({ data: { data: [{ name: 'yoshino', storage: 'users_xml', host_ip: ['::/0'], default_database: 'default', grants: ['GRANT SELECT ON default.* TO yoshino'] }] } } as never)
  vi.mocked(api.listDatabases).mockResolvedValue({ data: { data: [{ name: 'default' }] } } as never)
  vi.mocked(api.getConnectionSettings).mockResolvedValue({ data: { data: { user: 'default' } } } as never)
  vi.mocked(api.grantPrivileges).mockResolvedValue({} as never)
  vi.mocked(api.revokePrivileges).mockResolvedValue({} as never)
  vi.mocked(api.alterUser).mockResolvedValue({ data: { message: '密码已更新' } } as never)
  vi.mocked(api.createUser).mockResolvedValue({} as never)
})
afterEach(cleanup)

it('shows grants and supports granting all privileges to a user', async () => {
  render(<UsersPage />)
  expect(await screen.findByText(/读取数据/)).toBeTruthy()
  fireEvent.click(screen.getByRole('button', { name: /权限/ }))
  const dialog = within(screen.getByRole('dialog'))
  fireEvent.click(dialog.getByRole('button', { name: /授予全部权限/ }))
  fireEvent.click(dialog.getByRole('button', { name: /执.*行/ }))
  await waitFor(() => expect(api.grantPrivileges).toHaveBeenCalledWith({
    username: 'yoshino', database: '*', table: '*',
    privileges: ['ALL'],
    with_grant_option: true,
  }))
})

it('changes password and calls alterUser with the new password', async () => {
  render(<UsersPage />)
  fireEvent.click(await screen.findByRole('button', { name: /改密码/ }))
  const dialog = within(screen.getByRole('dialog'))
  fireEvent.change(dialog.getByLabelText('新密码'), { target: { value: 'changed-password' } })
  fireEvent.click(dialog.getByRole('button', { name: '保存密码' }))
  await waitFor(() => expect(api.alterUser).toHaveBeenCalledWith({ username: 'yoshino', new_password: 'changed-password' }))
})

it('creates a user and grants all privileges by default, handling retry if grant fails', async () => {
  vi.mocked(api.grantPrivileges).mockRejectedValueOnce(new Error('Not enough privileges'))
  render(<UsersPage />)
  await screen.findByText('yoshino')
  fireEvent.click(screen.getByRole('button', { name: /新建用户/ }))
  const dialog = within(screen.getByRole('dialog'))
  fireEvent.change(dialog.getByLabelText('用户名'), { target: { value: 'newuser' } })
  fireEvent.change(dialog.getByLabelText('连接密码'), { target: { value: 'test-password' } })
  fireEvent.click(dialog.getByRole('button', { name: '创建并授权' }))
  expect(await screen.findByText('数据库权限 · newuser')).toBeTruthy()
  expect(api.createUser).toHaveBeenCalledWith({
    username: 'newuser',
    password: 'test-password',
    allowed_hosts: [],
    default_database: 'default',
    all_privileges: true,
  })
  fireEvent.click(within(screen.getByText('数据库权限 · newuser').closest('[role="dialog"]') as HTMLElement).getByRole('button', { name: /执.*行/ }))
  await waitFor(() => expect(api.grantPrivileges).toHaveBeenCalledTimes(2))
})
