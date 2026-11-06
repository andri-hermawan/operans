// System Data → Project (hanya Super Admin yang mengubah).
import { createProject, listProjects, updateProject } from '@/services/api/project.api'
import type { CrudConfig } from '@/types/crud.types'
import type { Project } from '@/types/project.types'
import { companyOptions } from '@/utils/optionSources'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const TIMEZONES = ['Asia/Jakarta', 'Asia/Makassar', 'Asia/Jayapura'].map((v) => ({ value: v, label: v }))
const projectConfig: CrudConfig<Project> = {
  title: 'Project',
  resource: 'organization',
  rowKey: 'id',
  queryKey: 'projects',
  subtitle: 'Peta area (GeoJSON) tiap project diunggah dari menu peta (tahap F3).',
  list: listProjects,
  columns: [
    col.text('company_name', 'Company'),
    col.code('project_code', 'Code', { mobile: 'title' }),
    col.text('project_name', 'Name'),
    col.text('timezone', 'Timezone'),
    col.active(),
    col.dateTime('created_at', 'Created At', { mobile: 'hide' }),
  ],
  form: {
    fields: [
      { name: 'company_id', label: 'Company', type: 'select', options: companyOptions, required: true },
      { name: 'project_code', label: 'Code', type: 'upper', required: true },
      { name: 'project_name', label: 'Name', required: true },
      { name: 'timezone', label: 'Timezone', type: 'select', options: TIMEZONES, required: true, initial: 'Asia/Jakarta', help: 'Dasar jam operasional & shift' },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    create: (p) => createProject(p),
    update: (r, p) => updateProject(r.id, p),
  },
  exportName: 'project',
}

export default function ProjectPage() {
  return <CrudPage config={projectConfig} />
}
