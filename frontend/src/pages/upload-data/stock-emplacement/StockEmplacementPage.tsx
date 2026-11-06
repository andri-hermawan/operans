// Record / Upload → Stock Emplacement (stok cargo harian per partner, ton).
import { stockApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { StockEmplacement } from '@/types/daily.types'
import type { FieldDef } from '@/types/form.types'
import CrudPage from '@/components/crud/CrudPage'
import { dailyDefaults } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

type Key = keyof StockEmplacement & string

/** Kolom setelah Partner & Operational Date, urutan sama dengan tabel / template Excel (Info di antara stok dan Shift 1). */
const fields: { key: Key; title: string; numeric: boolean }[] = [
  { key: 'stock_cargo_initial', title: 'Stock Cargo Awal', numeric: true },
  { key: 'stock_cargo_scrapper', title: 'Stock Cargo Scrapper', numeric: true },
  { key: 'stock_cargo_crusher', title: 'Stock Cargo Stockpile Crusher', numeric: true },
  { key: 'info', title: 'Info', numeric: false },
  { key: 'shift_1', title: 'Shift 1', numeric: true },
  { key: 'shift_2', title: 'Shift 2', numeric: true },
  { key: 'hauling_cargo', title: 'Cargo Hauling', numeric: true },
  { key: 'total_cargo_initial', title: 'Total Cargo Awal', numeric: true },
  { key: 'tls_out_trainset_1', title: 'TLS Out Trainset 1', numeric: true },
  { key: 'tls_out_trainset_2', title: 'TLS Out Trainset 2', numeric: true },
  { key: 'tls_out_trainset_3', title: 'TLS Out Trainset 3', numeric: true },
  { key: 'tls_out_trainset_4', title: 'TLS Out Trainset 4', numeric: true },
  { key: 'tls_out_trainset_5', title: 'TLS Out Trainset 5', numeric: true },
  { key: 'total_tls_out', title: 'Total TLS Out', numeric: true },
  { key: 'remaining_cargo', title: 'Sisa Cargo', numeric: true },
]

const stockConfig: CrudConfig<StockEmplacement> = {
  title: 'Stock Emplacement',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'stock-emplacements',
  filters: [{ type: 'date' }],
  defaultFilters: dailyDefaults,
  list: (f, ctx) => stockApi.list(ctx.projectId, f.date, f.date),
  columns: [
    col.rowNo(),
    col.code('partner', 'Partner', { mobile: 'title' }),
    col.plainDate('operational_date', 'Operational Date'),
    ...fields.map((f) =>
      f.numeric ? col.number<StockEmplacement>(f.key, f.title, 2, { mobile: 'hide' }) : col.text<StockEmplacement>(f.key, f.title),
    ),
  ],
  form: {
    width: 720,
    fields: [
      { name: 'partner', label: 'Partner', required: true },
      { name: 'operational_date', label: 'Operational Date', type: 'date', required: true },
      ...fields.map(
        (f): FieldDef =>
          f.numeric
            ? { name: f.key, label: f.title, type: 'number', unit: 'ton', min: 0 }
            : { name: f.key, label: f.title, help: 'Maks. 10 karakter (mis. OK)' },
      ),
    ],
    initial: (_c, f) => ({ operational_date: f.date }),
    create: (p, ctx) => stockApi.create({ ...p, project_id: ctx.projectId }),
    update: (row, p) => stockApi.update(row.id, p),
  },
  remove: { action: (row) => stockApi.remove(row.id), label: (r) => `stok ${r.partner} ${r.operational_date}` },
  importType: 'stock-emplacements',
  exportName: 'stock_emplacement',
}

export default function StockEmplacementPage() {
  return <CrudPage config={stockConfig} />
}
