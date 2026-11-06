import { App } from 'antd'

import { getDocumentFile } from '@/services/api/aiDocument.api'
import { errorMessage } from '@/services/http/ApiError'

/**
 * Cetak PDF asli sebuah dokumen (hanya superadmin; server menolak role lain dan mencatat tiap pencetakan).
 *   const print = usePrintDocument();  <Button onClick={() => print(doc.id)}>Cetak</Button>
 * PDF dibuka di tab baru; cetak lewat tombol cetak pembaca PDF browser (Ctrl+P).
 */
export function usePrintDocument() {
  const { message } = App.useApp()
  return async (documentId: string) => {
    // tab dibuka SAAT diklik (sebelum menunggu unduhan) supaya tidak diblokir pop-up blocker
    const tab = window.open('', '_blank')
    try {
      const blob = await getDocumentFile(documentId)
      const url = URL.createObjectURL(blob)
      if (tab) tab.location.href = url
      else window.open(url, '_blank')
      setTimeout(() => URL.revokeObjectURL(url), 10 * 60_000)
    } catch (e) {
      tab?.close()
      message.error(errorMessage(e))
    }
  }
}
