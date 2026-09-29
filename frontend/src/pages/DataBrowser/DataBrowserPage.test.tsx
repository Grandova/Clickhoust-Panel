// @vitest-environment jsdom
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { DataBrowserPage } from './DataBrowserPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({ api: { listDatabases: vi.fn(), listTables: vi.fn(), browseData: vi.fn() } }))

const result = { data: { data: {
  columns: [{ name: 'id', type: 'UInt64' }, { name: 'value', type: 'String' }],
  rows: [{ id: 0, value: 'first row' }], total_rows: 1, total_rows_estimate: 1, elapsed_ms: 1,
} } }

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', { value: () => ({ matches: false, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {} }) })
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  const getStyle = window.getComputedStyle
  vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => getStyle(element))
})

beforeEach(() => {
  vi.mocked(api.listDatabases).mockResolvedValue({ data: { data: [{ name: 'default' }] } } as never)
  vi.mocked(api.listTables).mockResolvedValue({ data: { data: [{ name: 'events' }] } } as never)
  vi.mocked(api.browseData).mockResolvedValue(result as never)
})
afterEach(() => { cleanup(); vi.clearAllMocks() })

describe('data browser', () => {
  it('sorts using the selected column and new direction', async () => {
    render(<DataBrowserPage />)
    await screen.findByText('first row')
    fireEvent.click(screen.getByRole('columnheader', { name: /id UInt64/ }))
    await waitFor(() => expect(api.browseData).toHaveBeenLastCalledWith(expect.objectContaining({ sort_field: 'id', sort_order: 'ASC', page: 1 })))
  })

  it('clears stale rows when a refresh fails', async () => {
    render(<DataBrowserPage />)
    await screen.findByText('first row')
    vi.mocked(api.browseData).mockRejectedValueOnce(new Error('offline'))
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }))
    await screen.findByText('查询失败，请检查连接或筛选条件后重试。')
    expect(screen.queryByText('first row')).toBeNull()
  })

  it('ignores an older query that finishes after a newer query', async () => {
    render(<DataBrowserPage />)
    await screen.findByText('first row')
    let finish!: (value: never) => void
    vi.mocked(api.browseData).mockReturnValueOnce(new Promise((resolve) => { finish = resolve }) as never)
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }))
    vi.mocked(api.browseData).mockResolvedValueOnce({ data: { data: { ...result.data.data, rows: [{ id: 1, value: 'newer row' }] } } } as never)
    fireEvent.click(screen.getByRole('button', { name: /刷新/ }))
    await screen.findByText('newer row')
    finish(result as never)
    await waitFor(() => expect(screen.queryByText('first row')).toBeNull())
    expect(screen.getByText('newer row')).toBeTruthy()
  })
})
