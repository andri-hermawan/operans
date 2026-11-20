import { useState } from 'react'
import { Alert, Button, Form, Input, Typography } from 'antd'
import { Lock, Mail } from 'lucide-react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { LoginForm } from '@/types/auth.types'
import logo from '@/assets/logo.png'
import rmkoWhite from '@/assets/rmko/RMKO_White.png'

import styles from './login.module.css'

const FEATURES = ['Live Tracking', 'Geofencing', 'Abnormal Alerts', 'Abnormal Distribution Map', 'Position History', 'Fuel History', 'Speed Heatmap']

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const status = useAuthStore((s) => s.status)
  const login = useAuthStore((s) => s.login)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(false)
  const from = (location.state as { from?: string } | null)?.from ?? '/'

  if (status === 'signedIn') return <Navigate to={from} replace />

  const submit = async (v: LoginForm) => {
    setError(undefined)
    setLoading(true)
    try {
      await login(v.email.trim(), v.password)
      navigate(from, { replace: true })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <aside className={styles.brand}>
        <img src={rmkoWhite} alt="RMKO" className={styles.brandLogo} />
        <div className={styles.brandBody}>
          <h1 className={styles.title}>OPERANS</h1>
          <p className={styles.by}>
            <span>by</span> PT Royaltama Mulia Kontraktorindo Tbk
          </p>
          <p className={styles.tagline}>Operation Management System</p>
          <p className={styles.scope}>Mining | Hauling | Emplacement</p>
          <ul className={styles.chips}>
            {FEATURES.map((f) => (
              <li key={f}>{f}</li>
            ))}
          </ul>
        </div>
      </aside>

      <main className={styles.panel}>
        <div className={styles.card}>
          <div className={styles.logo}>
            <img src={logo} alt="OPERANS" className={styles.mark} />
            <h2 className={styles.heading}>Selamat datang</h2>
            <Typography.Text type="secondary">Masuk untuk melanjutkan ke OPERANS</Typography.Text>
          </div>

          {error && <Alert type="error" title={error} showIcon className={styles.error} />}

          <Form<LoginForm> layout="vertical" size="large" requiredMark={false} onFinish={submit} disabled={loading}>
            <Form.Item name="email" label="Email" rules={[{ required: true, message: 'Email wajib diisi' }, { type: 'email', message: 'Format email tidak valid' }]}>
              <Input prefix={<Mail size={16} />} placeholder="nama@perusahaan.com" autoComplete="email" autoFocus />
            </Form.Item>
            <Form.Item name="password" label="Password" rules={[{ required: true, message: 'Password wajib diisi' }]}>
              <Input.Password prefix={<Lock size={16} />} placeholder="••••••••" autoComplete="current-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading} className={styles.submit}>
              {loading ? 'Memproses...' : 'Masuk'}
            </Button>
          </Form>

          <div className={styles.footer}>
            <Typography.Text type="secondary">OPERANS by RMKO © {new Date().getFullYear()}</Typography.Text>
          </div>
        </div>
      </main>
    </div>
  )
}
