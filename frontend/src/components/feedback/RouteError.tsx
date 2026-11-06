import { Button, Result } from 'antd'
import { useRouteError } from 'react-router-dom'

/** Galat tak terduga saat menampilkan halaman (bukan galat API). */
export default function RouteError() {
  const error = useRouteError() as Error | undefined
  return (
    <Result
      status="error"
      title="Terjadi kesalahan pada halaman"
      subTitle={error?.message}
      extra={<Button type="primary" onClick={() => window.location.reload()}>Muat ulang</Button>}
    />
  )
}
