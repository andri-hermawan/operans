import { useState } from 'react'
import { Alert, Button, Form, Input, Typography } from 'antd'
import { Lock, Mail } from 'lucide-react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { LoginForm } from '@/types/auth.types'
import { HOME } from '@/router/routes'
import logo from '@/assets/logo.png'

import styles from './login.module.css'

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const status = useAuthStore((s) => s.status)
  const login = useAuthStore((s) => s.login)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(false)
  const from = (location.state as { from?: string } | null)?.from ?? HOME

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
      <div className={styles.card}>
        <div className={styles.logo}>
          <img src={logo} alt="RMKO" height={84} />
          <h1 className={styles.company}>Operation Management System</h1>
          <Typography.Text type="secondary">Mining | Hauling | Emplacement</Typography.Text>
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

        <Typography.Text type="secondary" className={styles.footer}>
          PT Royaltama Mulia Kontraktorindo Tbk © {new Date().getFullYear()}
        </Typography.Text>
      </div>
    </div>
  )
}
