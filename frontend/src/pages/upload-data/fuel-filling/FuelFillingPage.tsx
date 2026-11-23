// Record / Upload → Transaction → Fuel Filling (pengisian BBM unit per shift).
import { fuelFillingApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { FuelFilling } from '@/types/daily.types'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitColumns, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const fuelFillingCrud = dailyCrud(fuelFillingApi)
const liter = (name: string, label: string) => ({ name, label, type: 'number' as const, unit: 'L', min: 0 })

const fuelFillingConfig: CrudConfig<FuelFilling> = {
  title: 'Fuel Filling',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'fuel-fillings',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: fuelFillingCrud.list,
  columns: [
    ...dateShiftUnitColumns<FuelFilling>(),
    col.text('equipment_model', 'Model', { mobile: 'hide' }),
    col.number('hm_previous', 'HM Sebelumnya', 2),
    col.number('hm_filling', 'HM Pengisian', 2),
    col.number('total_hours', 'Total (jam)', 2),
    col.number('fuel_usage_liter', 'Pemakaian (L)', 2),
    col.number('fuel_consumption_lph', 'Konsumsi (L/jam)', 2),
    col.number('waste_removal_liter', 'Waste Removal (L)', 2, { mobile: 'hide' }),
    col.number('coal_mining_liter', 'Coal Mining (L)', 2, { mobile: 'hide' }),
    col.number('hauling_liter', 'Hauling (L)', 2, { mobile: 'hide' }),
    col.number('rental_liter', 'Rental (L)', 2, { mobile: 'hide' }),
    col.number('dewatering_liter', 'Dewatering (L)', 2, { mobile: 'hide' }),
    col.number('final_support_liter', 'Final Support (L)', 2, { mobile: 'hide' }),
  ],
  form: {
    width: 720,
    fields: [
      ...dateShiftUnitFields,
      { name: 'equipment_model', label: 'Model' },
      { name: 'hm_previous', label: 'HM Pengisian Sebelumnya', type: 'number', unit: 'jam', min: 0 },
      { name: 'hm_filling', label: 'HM Pengisian', type: 'number', unit: 'jam', min: 0 },
      { name: 'total_hours', label: 'Total Jam', type: 'number', unit: 'jam', min: 0, help: 'Kosong = HM Pengisian − HM Sebelumnya (dihitung otomatis)' },
      liter('fuel_usage_liter', 'Pemakaian BBM'),
      { name: 'fuel_consumption_lph', label: 'Konsumsi BBM', type: 'number', unit: 'L/jam', min: 0, help: 'Kosong = Pemakaian BBM ÷ Total Jam (dihitung otomatis)' },
      liter('waste_removal_liter', 'Waste Removal'),
      liter('coal_mining_liter', 'Coal Mining'),
      liter('hauling_liter', 'Hauling'),
      liter('rental_liter', 'Rental'),
      liter('dewatering_liter', 'Dewatering'),
      liter('final_support_liter', 'Final Support'),
    ],
    toForm: dailyToForm,
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: fuelFillingCrud.create,
    update: fuelFillingCrud.update,
  },
  remove: { action: fuelFillingCrud.remove, label: (r) => `pengisian BBM ${r.equipment_code}` },
  importType: 'fuel-fillings',
  exportName: 'fuel_filling',
}

export default function FuelFillingPage() {
  return <CrudPage config={fuelFillingConfig} />
}
