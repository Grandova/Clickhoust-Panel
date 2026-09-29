// @vitest-environment jsdom
import { expect, it } from 'vitest'
import { build } from 'vite'
import path from 'node:path'

it('renders the production chart component as a React element', async () => {
  const entry = path.resolve('monitoring-render-test.ts')
  const result = await build({
    configFile: false,
    logLevel: 'silent',
    oxc: { jsx: { development: false } },
    plugins: [{
      name: 'monitoring-render-test',
      resolveId(id) { if (id === entry) return id },
      load(id) {
        if (id === entry) return `
          import React from 'react';
          import { renderToString } from 'react-dom/server';
          import { MonitoringPage } from ${JSON.stringify(path.resolve('src/pages/Monitoring/MonitoringPage.tsx'))};
          export const html = renderToString(React.createElement(MonitoringPage, { isDark: false }));
        `
      },
    }],
    build: {
      write: false,
      minify: false,
      lib: { entry, formats: ['iife'], name: 'MonitoringRenderTest' },
    },
    define: { 'process.env.NODE_ENV': JSON.stringify('production') },
  })
  const output = (Array.isArray(result) ? result[0] : result) as { output: { type: string; code?: string }[] }
  const code = output.output.find((item) => item.type === 'chunk')!.code!
  // Execute browser-format build output; source-only imports hide CJS/ESM interop regressions.
  const html = new Function(`${code}; return MonitoringRenderTest.html;`)() as string
  expect(html).toContain('实时监控')
  expect(html.match(/class="echarts-for-react /g)).toHaveLength(4)
}, 30000)
