import type { On } from 'claude-code'
import { expect, mock, test } from 'claude-code/testing'

import { parseStatus } from '../hooks/git'

const PANE = {
  plugin: 'sidegit',
  component: 'Pane',
  requestId: 'sidegit',
  props: {
    title: 'sidegit',
    isFocused: false,
    bodyColumns: 60,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 40 },
    view: {},
  },
} as const

// Two repos in one folder that is not a repo itself, and a hidden one to skip.
const FIND = '/work/api/.git\n/work/web/.git\n/work/.cache/old/.git\n'
const GIT: Record<string, Record<string, string>> = {
  '/work/api': {
    status: '# branch.oid abc\x00# branch.head main\x00# branch.ab +1 -0\x001 .M N... 100644 100644 100644 h1 h2 src/server.ts\x00? notes.md\x00',
    diff: '3\t1\tsrc/server.ts\x00',
  },
  '/work/web': {
    status: '# branch.head feature/x\x00# branch.ab +0 -0\x001 A. N... 000000 100644 100644 h0 h2 app/page.tsx\x00',
    diff: '12\t0\tapp/page.tsx\x00',
  },
}

// The tool inputs of every connected MCP server join one union that tsc cannot resolve
// $.tool.call against, so the tests call it through this narrower type.
type CallTool = (input: { tool: string } & Record<string, string>) => Promise<unknown>

// The pane opens and closes the mod asked for, in order.
const paneCalls: string[] = []

function world(on: On) {
  const clock = mock.clock(on, { now: 100_000 })
  const ok = (stdout: string) => ({ value: { exitCode: 0, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })
  paneCalls.length = 0
  on('session.start', (_, e) => ({ cwd: e.cwd }))
  on('session.root', () => ({ value: '/work' }))
  on('session.surfaces', () => ({ value: ['terminal'] as const }))
  on('command.register', () => ({ value: { command: 'sidegit' } }))
  on('ui.open', () => {
    paneCalls.push('open')
    return { value: { isPlaced: true } }
  })
  on('ui.close', () => {
    paneCalls.push('close')
    return { value: undefined }
  })
  on('ui.scroll', () => ({}))
  on('tool.call', () => ({ result: {} }))
  on('process.run', (_, e) => {
    const [cmd, , , repo, sub] = e.argv
    if (cmd === 'find') return ok(FIND)
    return ok(GIT[repo ?? '']?.[sub ?? ''] ?? '')
  })
  on('fs.stat', () => ({ value: { kind: 'file', size: 4, mtimeMs: 40_000, isLink: false } }))
  on('fs.read', () => ({ value: 'a\nb\n' }))
  return clock
}

test('one tree shows the changes of every repo in the folder, on every surface', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()

  for (const surface of ['terminal', 'desktop', 'vscode', 'mobile'] as const) {
    const ui = await $.ui.mount({ ...PANE, surface })
    expect((await ui.find({ key: 'repo:/work/api' }))?.text).toBe('▾ api [main] (2) ↑1')
    expect((await ui.find({ key: 'repo:/work/web' }))?.text).toBe('▾ web [feature/x] (1)')
    expect(await ui.find({ key: 'repo:/work/.cache/old' })).toBeUndefined()
    expect((await ui.find({ key: 'dir:/work/api/src' }))?.text).toBe('▾ src')
    expect((await ui.find({ key: 'row:/work/api/src/server.ts' }))?.text).toMatch(/│ └ M server\.ts\s*\+3 -1\s+1m$/)
    expect((await ui.find({ key: 'row:/work/api/notes.md' }))?.text).toMatch(/└ \? notes\.md\s*\+2\s+1m$/)
    expect((await ui.find({ key: 'row:/work/web/app/page.tsx' }))?.text).toMatch(/A page\.tsx\s*\+12\s+1m$/)
    expect((await ui.find({ type: 'Text', text: /repo\(s\)/ }))?.text).toBe(' 2 repo(s) | 3 change(s)')
    await ui.unmount()
  }
})

test('the file Claude edits is marked, and the mark goes after the call', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  await ($.tool.call as unknown as CallTool)({ tool: 'Edit', file_path: '/work/api/src/server.ts', old_string: 'a', new_string: 'b' })
  await clock.settle()
  expect((await ui.find({ key: 'row:/work/api/src/server.ts' }))?.text).toMatch(/M ✎ server\.ts/)
  expect((await ui.find({ type: 'Text', text: /repo\(s\)/ }))?.text).toBe(' 2 repo(s) | 3 change(s) | ✎ api/src/server.ts')

  await clock.advance(9_000)
  expect((await ui.find({ key: 'row:/work/api/src/server.ts' }))?.text).not.toMatch(/✎/)
  await ui.unmount()
})

test('an edit opens the collapsed rows that hold the file', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })

  await ui.press({ key: 'repo:/work/api' })
  expect((await ui.find({ key: 'repo:/work/api' }))?.text).toMatch(/^▸ api/)
  expect(await ui.find({ key: 'row:/work/api/src/server.ts' })).toBeUndefined()

  await ($.tool.call as unknown as CallTool)({ tool: 'Write', file_path: '/work/api/src/server.ts', content: 'b' })
  await clock.settle()
  expect((await ui.find({ key: 'repo:/work/api' }))?.text).toMatch(/^▾ api/)
  expect(await ui.find({ key: 'row:/work/api/src/server.ts' })).toBeDefined()
  await ui.unmount()
})

test('git status with -z keeps spaces in paths and skips the rename source', () => {
  const status = parseStatus('# branch.head (detached)\x002 R. N... 100644 100644 100644 h1 h2 R100 new name.ts\x00old name.ts\x00? a b.md\x00')
  expect(status.branch).toBe('HEAD')
  expect(status.files.map(f => [f.status, f.staged, f.path])).toEqual([
    ['R', true, 'new name.ts'],
    ['?', false, 'a b.md'],
  ])
})

test('the pane has the terminal background, from the top to the last row', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()

  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.drawn()).toMatchObject({ type: 'Box', props: { backgroundColor: '#282A36', minHeight: 40 } })
  await ui.unmount()
})

test('on the main screen the pane from session start closes, and draws nothing', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()
  expect(paneCalls).toEqual(['open'])

  const ui = await $.ui.mount({ ...PANE, surface: 'terminal', props: { ...PANE.props, placement: 'inline' } })
  await clock.settle()
  expect(paneCalls).toEqual(['open', 'close'])
  expect(await ui.find({ key: 'repo:/work/api' })).toBeUndefined()
  await ui.unmount()
})

test('/sidegit opens the pane only where the terminal docks it', async ($, on) => {
  const clock = world(on)
  await $.session.start({ cwd: '/work', surface: 'terminal', isInteractive: true })
  await clock.settle()
  paneCalls.length = 0
  const run = (isFullscreen: boolean, columns: number) =>
    $.command.run({ command: 'sidegit', args: '', origin: { kind: 'composer' }, presentation: { isFullscreen, columns } })

  expect((await run(false, 200)).text).toBe('sidegit shows on the side only in the fullscreen layout. Run /tui fullscreen, then /sidegit.')
  expect((await run(true, 100)).text).toBe('sidegit shows on the side only in a terminal at least 110 columns wide. Widen the terminal, then run /sidegit.')
  expect(paneCalls).toEqual([])

  expect((await run(true, 200)).text).toBe('sidegit pane opened.')
  expect(paneCalls).toEqual(['open'])
})
