import { describe, expect, it } from 'vitest'

import { formatFileSize, splitSnippet } from './documents'

describe('splitSnippet', () => {
  it('memisahkan kata yang cocok', () => {
    expect(splitSnippet('langkah [[keselamatan]] kerja')).toEqual([
      { text: 'langkah ', hit: false },
      { text: 'keselamatan', hit: true },
      { text: ' kerja', hit: false },
    ])
  })

  it('beberapa kata cocok dan penanda di ujung', () => {
    expect(splitSnippet('[[a]] b [[c]]')).toEqual([
      { text: 'a', hit: true },
      { text: ' b ', hit: false },
      { text: 'c', hit: true },
    ])
  })

  it('tanpa penanda atau penanda tidak lengkap = teks biasa', () => {
    expect(splitSnippet('tidak ada')).toEqual([{ text: 'tidak ada', hit: false }])
    expect(splitSnippet('rusak [[tanpa penutup')).toEqual([{ text: 'rusak [[tanpa penutup', hit: false }])
    expect(splitSnippet('')).toEqual([])
  })

  it('tag HTML di isi dokumen tetap teks biasa (tidak dijalankan)', () => {
    const parts = splitSnippet('<script>alert(1)</script> [[apd]]')
    expect(parts[0]).toEqual({ text: '<script>alert(1)</script> ', hit: false })
    expect(parts[1]).toEqual({ text: 'apd', hit: true })
  })
})

describe('formatFileSize', () => {
  it('satuan sesuai ukuran', () => {
    expect(formatFileSize(500)).toBe('500 B')
    expect(formatFileSize(1536)).toBe('1,5 KB')
    expect(formatFileSize(3_000_000)).toBe('2,9 MB')
  })
})
