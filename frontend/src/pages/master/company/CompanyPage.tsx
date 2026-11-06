// System Data → Company (hanya Super Admin yang mengubah).
import { createCompany, listCompanies, updateCompany } from '@/services/api/company.api'
import type { Company } from '@/types/company.types'
import type { CrudConfig } from '@/types/crud.types'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const companyConfig: CrudConfig<Company> = {
  title: 'Company',
  resource: 'organization',
  rowKey: 'id',
  queryKey: 'companies',
  list: listCompanies,
  columns: [
    col.code('company_code', 'Code', { mobile: 'title' }),
    col.text('company_name', 'Name'),
    col.number('project_count', 'Project'),
    col.active(),
    col.dateTime('created_at', 'Created At', { mobile: 'hide' }),
  ],
  form: {
    fields: [
      { name: 'company_code', label: 'Code', type: 'upper', required: true },
      { name: 'company_name', label: 'Name', required: true },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    create: (p) => createCompany(p),
    update: (r, p) => updateCompany(r.id, p),
  },
  exportName: 'company',
}

export default function CompanyPage() {
  return <CrudPage config={companyConfig} />
}
