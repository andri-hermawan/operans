import { useMutation, useQueryClient } from '@tanstack/react-query'

import {
  createDepartment,
  deactivateDocument,
  reprocessDocument,
  setDocumentRoles,
  uploadAiDocument,
} from '@/services/api/aiDocument.api'
import type { ReaderRole } from '@/types/aiDocument.types'

import { useProjectKey } from './useProjectKey'

/** Aksi pengelolaan dokumen (hanya admin): setelah berhasil, daftar & pencarian dimuat ulang. */
export function useDocumentActions() {
  const qc = useQueryClient()
  const { pid } = useProjectKey()
  const refresh = (...kinds: string[]) => Promise.all(kinds.map((k) => qc.invalidateQueries({ queryKey: [pid, k] })))

  return {
    upload: useMutation({ mutationFn: uploadAiDocument, onSuccess: () => refresh('ai-documents', 'ai-search') }),
    setRoles: useMutation({
      mutationFn: (v: { id: string; roles: ReaderRole[] }) => setDocumentRoles(v.id, v.roles),
      onSuccess: () => refresh('ai-documents', 'ai-search'),
    }),
    deactivate: useMutation({
      mutationFn: deactivateDocument,
      onSuccess: () => refresh('ai-documents', 'ai-search', 'ai-page'),
    }),
    reprocess: useMutation({
      mutationFn: reprocessDocument,
      onSuccess: () => refresh('ai-documents', 'ai-search', 'ai-page', 'ai-page-image'),
    }),
    createDepartment: useMutation({ mutationFn: createDepartment, onSuccess: () => refresh('ai-departments') }),
  }
}
