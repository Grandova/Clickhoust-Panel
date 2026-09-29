import Editor, { loader, DiffEditor, useMonaco } from '@monaco-editor/react'
import * as monaco from 'monaco-editor'
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker'

// Bundle the editor and worker locally so private deployments do not need a CDN.
self.MonacoEnvironment = { getWorker: () => new EditorWorker() }
loader.config({ monaco })

export { DiffEditor, useMonaco }
export default Editor
