import AlertList from '@/components/alert/AlertList'

export default function FuelAlertPage() {
  return <AlertList title="Fuel Alert" types={['fuel_decrease_engine_on', 'fuel_decrease_engine_off']} />
}
