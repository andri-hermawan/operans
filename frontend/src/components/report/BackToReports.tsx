import { Button } from 'antd'
import { ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

/** Tombol di halaman laporan: kembali ke daftar laporan. */
export default function BackToReports() {
  const navigate = useNavigate()
  return <Button icon={<ArrowLeft size={14} />} onClick={() => navigate('/report')}>Back to Reports</Button>
}
