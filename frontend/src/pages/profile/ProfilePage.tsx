import { App, Button, Card, Col, Descriptions, Form, Input, Row, Tag } from 'antd'
import { useNavigate } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { PasswordForm } from '@/types/auth.types'
import { useChangePassword } from '@/hooks/useChangePassword'
import PageHeader from '@/components/ui/PageHeader'

export default function ProfilePage() {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const role = useAuthStore((s) => s.role)
  const project = useAuthStore((s) => s.project)
  const memberships = useAuthStore((s) => s.memberships)
  const logout = useAuthStore((s) => s.logout)

  // core-api mengakhiri semua sesi setelah password diganti → login ulang.
  const change = useChangePassword({
    onSuccess: async () => {
      message.success('Password diganti. Silakan login kembali.')
      await logout()
      navigate('/login')
    },
    onError: (e) => message.error(errorMessage(e)),
  })

  return (
    <>
      <PageHeader title="Profil Saya" />
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="Akun">
            <Descriptions column={1} size="small" items={[
              { key: 'name', label: 'Nama', children: user?.name },
              { key: 'email', label: 'Email', children: user?.email },
              { key: 'role', label: 'Role', children: role },
              { key: 'project', label: 'Project aktif', children: project ? `${project.project_code} — ${project.project_name}` : '-' },
              { key: 'all', label: 'Semua project', children: memberships.map((m) => <Tag key={m.project_id}>{m.project_code} · {m.role}</Tag>) },
            ]} />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Ganti Password">
            <Form<PasswordForm> layout="vertical" onFinish={(v) => change.mutate(v)} disabled={change.isPending}>
              <Form.Item name="current_password" label="Password saat ini" rules={[{ required: true, message: 'Wajib diisi' }]}>
                <Input.Password autoComplete="current-password" />
              </Form.Item>
              <Form.Item name="new_password" label="Password baru" rules={[{ required: true, message: 'Wajib diisi' }, { min: 8, message: 'Minimal 8 karakter' }]}>
                <Input.Password autoComplete="new-password" />
              </Form.Item>
              <Form.Item
                name="confirm"
                label="Ulangi password baru"
                dependencies={['new_password']}
                rules={[
                  { required: true, message: 'Wajib diisi' },
                  ({ getFieldValue }) => ({ validator: (_, v) => (v === getFieldValue('new_password') ? Promise.resolve() : Promise.reject(new Error('Password tidak sama'))) }),
                ]}
              >
                <Input.Password autoComplete="new-password" />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={change.isPending}>Simpan Password</Button>
            </Form>
          </Card>
        </Col>
      </Row>
    </>
  )
}
