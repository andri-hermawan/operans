import { useQuery } from '@tanstack/react-query'

import { getDocumentPage, getDocumentPageImage, listAiDocuments } from '@/services/api/aiDocument.api'
import type { DocumentFilters } from '@/types/aiDocument.types'

import { useProjectKey } from './useProjectKey'

/** Daftar dokumen yang BOLEH dilihat user (hak akses dijalankan server: project + role). */
export function useAiDocuments(filters: DocumentFilters) {
  const { pid, key } = useProjectKey('ai-documents', filters)
  return useQuery({
    queryKey: key,
    enabled: !!pid,
    queryFn: () => listAiDocuments(filters),
    // dokumen yang sedang dibaca OCR di server: daftar menyegarkan diri sampai selesai
    refetchInterval: (q) => (q.state.data?.some((d) => d.processing_status === 'processing') ? 4000 : false),
  })
}

/** Gambar satu halaman dokumen (PNG bertanda air pembaca); hanya diambil saat tampilan gambar dibuka. */
export function useDocumentPageImage(documentId: string | undefined, page: number, enabled: boolean) {
  const { pid, key } = useProjectKey('ai-page-image', documentId, page)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!documentId && enabled,
    staleTime: 5 * 60_000,
    gcTime: 5 * 60_000,
    queryFn: () => getDocumentPageImage(documentId!, page),
  })
}

/** Teks satu halaman dokumen (isi dokumen tidak berubah → disimpan di cache beberapa menit). */
export function useDocumentPage(documentId: string | undefined, page: number) {
  const { pid, key } = useProjectKey('ai-page', documentId, page)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!documentId,
    staleTime: 5 * 60_000,
    queryFn: () => getDocumentPage(documentId!, page),
  })
}
