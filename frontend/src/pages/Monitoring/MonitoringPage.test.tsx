// @vitest-environment jsdom
import { afterEach, beforeAll, expect, it, vi } from 'vitest'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import * as echarts from 'echarts/core'
import { MonitoringPage } from './MonitoringPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({ api: { getRealtimeMetrics: vi.fn() } }))

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', { value: () => ({ matches: false, addListener() {}, removeListener() {} }) })
  Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(600)
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(260)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (this: HTMLCanvasElement) {
    return new Proxy({ canvas: this, measureText: (text: string) => ({ width: text.length * 7 }) }, {
      get: (target, key) => key in target ? target[key as keyof typeof target] : () => {},
    }) as never
  })
})
afterEach(cleanup)

it('mounts all four real charts and renders the returned metrics', async () => {
  vi.mocked(api.getRealtimeMetrics).mockResolvedValue({ data: { data: {
    ch: { query_per_second: 7, current_connections: 3 },
    host: { cpu_percent: 25, mem_percent: 40, net_rx_bytes_sec: 1024 },
  } } } as never)
  const { container } = render(<MonitoringPage isDark={false} />)
  expect(screen.getByText('实时监控 · 当前页面采样')).toBeTruthy()
  await waitFor(() => {
    const charts = [...container.querySelectorAll<HTMLElement>('.echarts-for-react')]
    expect(charts).toHaveLength(4)
    const options = charts.map((chart) => echarts.getInstanceByDom(chart)?.getOption())
    expect((options[0]?.series as { data: number[] }[])?.[0]?.data).toEqual([7])
    expect((options[1]?.series as { data: number[] }[])?.[0]?.data).toEqual([3])
    expect((options[2]?.series as { data: number[] }[])?.[0]?.data).toEqual([25])
    expect((options[3]?.series as { data: number[] }[])?.[0]?.data).toEqual([1])
  })
})
