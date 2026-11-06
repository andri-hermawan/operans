// Bantuan tampilan dokumen SOP (fungsi biasa, tanpa tampilan).
import type { ProcessingStatus, ReaderRole } from '@/types/aiDocument.types'

export const READER_ROLE_LABEL: Record<ReaderRole, string> = { dispatcher: 'Dispatcher', viewer: 'Viewer' }

export const STATUS_LABEL: Record<ProcessingStatus, string> = {
  uploaded: 'Diunggah',
  processing: 'Diproses',
  ready: 'Siap',
  failed: 'Gagal',
}

/** 1536 → "1,5 KB"; 3_000_000 → "2,9 MB". */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1).replace('.', ',')} KB`
  return `${(bytes / 1024 / 1024).toFixed(1).replace('.', ',')} MB`
}

export interface SnippetPart {
  text: string
  hit: boolean
}

/**
 * Pecah potongan hasil pencarian jadi bagian biasa & bagian cocok. Server menandai kata cocok dengan [[ ]].
 * Hasilnya ditampilkan sebagai TEKS React (bukan HTML), sehingga isi PDF tidak bisa menyisipkan tag.
 */
export function splitSnippet(snippet: string): SnippetPart[] {
  const parts: SnippetPart[] = []
  let rest = snippet
  while (rest) {
    const start = rest.indexOf('[[')
    const end = start < 0 ? -1 : rest.indexOf(']]', start + 2)
    if (start < 0 || end < 0) {
      parts.push({ text: rest, hit: false })
      break
    }
    if (start > 0) parts.push({ text: rest.slice(0, start), hit: false })
    parts.push({ text: rest.slice(start + 2, end), hit: true })
    rest = rest.slice(end + 2)
  }
  return parts
}
