// Hasil pencarian kata kunci di dalam isi dokumen. Klik hasil → buka halaman itu di pembaca.
import { Alert, Button, Empty, List, Skeleton, Tag, Typography } from 'antd'

import { MIN_SEARCH_CHARS, useDocumentSearch } from '@/hooks/useDocumentSearch'
import { errorMessage } from '@/services/http/ApiError'
import type { DocumentFilters, SearchHit } from '@/types/aiDocument.types'
import { brand } from '@/utils/colors'
import { splitSnippet } from '@/utils/documents'

interface Props {
  query: string
  filters: DocumentFilters
  onOpen: (hit: SearchHit) => void
}

export default function SearchResults({ query, filters, onOpen }: Props) {
  const { data, isFetching, error, refetch } = useDocumentSearch(query, filters)

  if (query.trim().length < MIN_SEARCH_CHARS) {
    return <Alert type="info" showIcon title={`Ketik minimal ${MIN_SEARCH_CHARS} huruf untuk mencari.`} />
  }
  if (error) {
    return <Alert type="error" showIcon title={errorMessage(error)} action={<Button size="small" onClick={() => refetch()}>Coba lagi</Button>} />
  }
  if (!data) return <Skeleton active paragraph={{ rows: 4 }} />

  return (
    <List
      loading={isFetching}
      dataSource={data.hits}
      locale={{ emptyText: <Empty description={`Tidak ada dokumen yang memuat “${data.query}”`} /> }}
      header={<Typography.Text type="secondary">{data.count} hasil untuk “{data.query}”</Typography.Text>}
      renderItem={(hit) => (
        <List.Item key={`${hit.document_id}-${hit.page_number}`} onClick={() => onOpen(hit)} style={{ cursor: 'pointer' }}>
          <List.Item.Meta
            title={
              <>
                <Typography.Text strong style={{ color: brand.primaryDark }}>{hit.title}</Typography.Text>{' '}
                <Tag>{hit.department_name}</Tag>
                <Tag color="default">Halaman {hit.page_number}</Tag>
              </>
            }
            description={
              // teks React biasa → aman walau isi PDF memuat tag HTML
              <span>
                …{splitSnippet(hit.snippet).map((p, i) => (p.hit ? <mark key={i}>{p.text}</mark> : <span key={i}>{p.text}</span>))}…
              </span>
            }
          />
        </List.Item>
      )}
    />
  )
}
