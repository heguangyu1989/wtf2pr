import test from 'node:test'
import assert from 'node:assert/strict'
import { buildFileTree, filterFiles, getDefaultFilePath, getFileSidePath, getNodeKey, getFilePath, sortFiles } from '../src/utils/fileExplorer.js'

const file = path => ({ oldFile: path, newFile: path, hunks: [] })
const paths = files => files.map(item => getFilePath(item))

test('classifies tests and docs through consumer-visible sorting', () => {
  const input = [file('README.md'), file('LICENSE'), file('src/main.go'), file('internal/git/git_test.go'), file('web/src/a.test.js'), file('web/src/a.spec.ts'), file('__tests__/app.js'), file('docs/system_design.md')]
  assert.deepEqual(paths(sortFiles(input, 'test', 'asc').slice(0, 3)), ['__tests__/app.js', 'internal/git/git_test.go', 'web/src/a.spec.ts'])
  assert.deepEqual(paths(sortFiles(input, 'document', 'asc').slice(0, 3)), ['docs/system_design.md', 'LICENSE', 'README.md'])
})

test('uses the new path and falls back to the old path for deleted files', () => {
  assert.equal(getFilePath(file('src/new.js')), 'src/new.js')
  assert.equal(getFilePath({ oldFile: 'src/deleted.js', newFile: '' }), 'src/deleted.js')
  assert.equal(getFilePath(null), '')
})

test('sorts paths naturally without mutating the input', () => {
  const input = [file('src/file10.js'), file('src/file2.js'), file('src/file1.js')]
  const original = [...input]
  const got = sortFiles(input, 'path', 'asc')
  assert.deepEqual(paths(got), ['src/file1.js', 'src/file2.js', 'src/file10.js'])
  assert.deepEqual(input, original)
  assert.notEqual(got, input)
})

test('sorts tests first with deterministic secondary path ordering', () => {
  const got = sortFiles([file('z/a10.js'), file('a/a10.test.js'), file('a/a2.test.js')], 'test', 'asc')
  assert.deepEqual(paths(got), ['a/a2.test.js', 'a/a10.test.js', 'z/a10.js'])
})

test('sorts documents first and reverses both groups and paths', () => {
  const input = [file('src/z.js'), file('docs/z.md'), file('src/a.js'), file('docs/a.md')]
  const ascending = sortFiles(input, 'document', 'asc')
  assert.deepEqual(paths(ascending), ['docs/a.md', 'docs/z.md', 'src/a.js', 'src/z.js'])
  const descending = sortFiles(input, 'document', 'desc')
  assert.deepEqual(paths(descending), ['src/z.js', 'src/a.js', 'docs/z.md', 'docs/a.md'])
})

test('sorts by extension before natural path ordering', () => {
  const got = sortFiles([file('z/a.go'), file('a/z.js'), file('a/a.js'), file('README.md')], 'type', 'asc')
  assert.deepEqual(paths(got), ['z/a.go', 'a/a.js', 'a/z.js', 'README.md'])
})

test('builds deep trees and reverses names without hiding directories', () => {
  const tree = buildFileTree([file('src/z/file2.js'), file('src/z/file10.js'), file('docs/a.md')], 'path', 'desc')
  assert.equal(tree[0].type, 'directory')
  assert.deepEqual(tree.map(node => node.name), ['src', 'docs'])
  const src = tree[0]
  assert.deepEqual(src.children[0].children.map(node => node.name), ['file10.js', 'file2.js'])
})

test('keeps directories before files and sorts tests inside their directory', () => {
  const tree = buildFileTree([file('src/z.js'), file('src/a/a.js'), file('src/a.test.js')], 'test', 'asc')
  const src = tree[0]
  assert.deepEqual(src.children.map(node => [node.type, node.name]), [
    ['directory', 'a'],
    ['file', 'a.test.js'],
    ['file', 'z.js'],
  ])
})

test('ignores empty path segments when constructing tree nodes', () => {
  const tree = buildFileTree([file('src//nested/file.js')], 'path', 'asc')
  assert.equal(tree[0].path, 'src')
  assert.equal(tree[0].children[0].path, 'src/nested')
  assert.equal(tree[0].children[0].children[0].path, 'src//nested/file.js')
})

test('uses defaults for unknown sorting inputs', () => {
  const got = sortFiles([file('b.js'), file('a.js')], 'unknown', 'unknown')
  assert.deepEqual(paths(got), ['a.js', 'b.js'])
})

test('prefers the first text diff and falls back when only binary files exist', () => {
  const binary = { ...file('assets/a.png'), isBinary: true, hunks: [] }
  const emptyDiff = file('docs/b.md')
  const textDiff = { ...file('src/c.js'), hunks: [{ lines: [{ type: 'addition' }] }] }
  assert.equal(getDefaultFilePath([binary, emptyDiff, textDiff]), 'src/c.js')
  assert.equal(getDefaultFilePath([binary]), 'assets/a.png')
  assert.equal(getDefaultFilePath([]), '')
})

test('selects old and new paths for renames with added/deleted fallbacks', () => {
  const renamed = { oldFile: 'src/old.js', newFile: 'src/new.js' }
  assert.equal(getFileSidePath(renamed, 'old'), 'src/old.js')
  assert.equal(getFileSidePath(renamed, 'new'), 'src/new.js')
  assert.equal(getFileSidePath({ oldFile: '', newFile: 'src/added.js' }, 'old'), 'src/added.js')
  assert.equal(getFileSidePath({ oldFile: 'src/deleted.js', newFile: '' }, 'new'), 'src/deleted.js')
})

test('distinguishes file and directory keys when a file becomes a directory', () => {
  assert.notEqual(getNodeKey({ type: 'file', path: 'a' }), getNodeKey({ type: 'directory', path: 'a' }))
})

test('empty glob filters preserve every file without mutating the input', () => {
  const input = [file('src/main.go'), file('README.md')]
  const original = [...input]
  const got = filterFiles(input, ' , , ', ' , ')
  assert.deepEqual(paths(got), ['src/main.go', 'README.md'])
  assert.deepEqual(input, original)
  assert.notEqual(got, input)
})

test('combines include and exclude lists with OR and applies excludes last', () => {
  const input = [
    file('main.go'),
    file('main_test.go'),
    file('internal/git/git_test.go'),
    file('docs/guide.md'),
    file('README.md'),
    file('web/app.test.js'),
    file('web/app.js'),
  ]
  const got = filterFiles(input, ' **/*_test.go, , *.md, **/*.test.js ', 'docs/**, internal/**')
  assert.deepEqual(paths(got), ['main_test.go', 'README.md', 'web/app.test.js'])
})

test('supports globstars, single-segment wildcards and question marks', () => {
  const input = [
    file('main.go'),
    file('src/main.go'),
    file('src/main_test.go'),
    file('src/nested/util.go'),
    file('src/nested/util_test.go'),
  ]
  assert.deepEqual(paths(filterFiles(input, '**/*.go')), paths(input))
  assert.deepEqual(paths(filterFiles(input, 'src/*.go')), ['src/main.go', 'src/main_test.go'])
  assert.deepEqual(paths(filterFiles(input, 'src/*')), ['src/main.go', 'src/main_test.go', 'src/nested/util.go', 'src/nested/util_test.go'])
  assert.deepEqual(paths(filterFiles(input, 'src/????.go')), ['src/main.go'])
})

test('keeps brace commas intact and supports character classes and literal regex characters', () => {
  const input = [
    file('README.md'),
    file('docs/guide.md'),
    file('docs/guide.mdx'),
    file('docs/guide.txt'),
    file('src/file1.js'),
    file('src/filea.js'),
    file('src/file10.js'),
    file('src/a+b(1).js'),
  ]
  assert.deepEqual(paths(filterFiles(input, '*.{md,mdx}')), ['README.md', 'docs/guide.md', 'docs/guide.mdx'])
  assert.deepEqual(paths(filterFiles(input, 'src/file[0-9].js')), ['src/file1.js'])
  assert.deepEqual(paths(filterFiles(input, 'src/file[!0-9].js')), ['src/filea.js'])
  assert.deepEqual(paths(filterFiles(input, 'src/a+b(1).js')), ['src/a+b(1).js'])
  assert.doesNotThrow(() => filterFiles(input, 'src/file[0-9.js, src/{main.go'))
  assert.deepEqual(filterFiles(input, 'src/file[0-9.js, src/{main.go'), [])
  assert.doesNotThrow(() => filterFiles(input, 'src/file[9-0].js'))
  assert.deepEqual(filterFiles(input, 'src/file[9-0].js'), [])
})

test('distinguishes root-relative paths from directory names at any depth', () => {
  const input = [file('src/main.go'), file('nested/src/main.go'), file('docs/guide.md'), file('nested/docs/guide.md')]
  assert.deepEqual(paths(filterFiles(input, 'src')), ['src/main.go', 'nested/src/main.go'])
  assert.deepEqual(paths(filterFiles(input, './src/*.go')), ['src/main.go'])
  assert.deepEqual(paths(filterFiles(input, 'docs/**')), ['docs/guide.md'])
  assert.deepEqual(paths(filterFiles(input, '**/docs/**')), ['docs/guide.md', 'nested/docs/guide.md'])

  const mixed = [file('README.md'), file('nested/guide.md'), file('docs/guide.txt')]
  assert.deepEqual(paths(filterFiles(mixed, '{*.md,docs/**}')), ['README.md', 'nested/guide.md', 'docs/guide.txt'])
})

test('filters renamed files by their new path and deleted files by their old path', () => {
  const renamed = { oldFile: 'src/old.go', newFile: 'src/new.js', hunks: [] }
  const deleted = { oldFile: 'src/old_test.go', newFile: '', hunks: [] }
  assert.deepEqual(filterFiles([renamed, deleted], '*.js'), [renamed])
  assert.deepEqual(filterFiles([renamed, deleted], '*_test.go'), [deleted])
})
