// Tab "Per User": pengecualian untuk satu user (ikut role / izinkan / blokir) di satu project.
import { useMemo, useState } from 'react'
import { Alert, Button, Card, Radio, Select, Space, Table, Tag, message } from 'antd'
import { useMutation, useQuery } from '@tanstack/react-query'

import type { AccessChange, AccessRule } from '@/types/access.types'
import type { ProjectRole } from '@/types/auth.types'
import { saveAccessRules } from '@/services/api/access.api'
import { listUsers } from '@/services/api/user.api'
import { errorMessage } from '@/services/http/ApiError'

import styles from './access.module.css'
import { CATALOG, ROLES, canBeGranted, defaultAllowed, indexRules, userAllowed, type CatalogItem, type UserChoice } from './catalog'

const CHUNK = 50

const toChoice = (v: boolean | undefined): UserChoice => (v === undefined ? 'inherit' : v ? 'allow' : 'block')

export default function UserMatrix({ projectId, rules, onSaved }: { projectId: string; rules: AccessRule[]; onSaved: () => void }) {
  const idx = useMemo(() => indexRules(rules), [rules])
  const users = useQuery({ queryKey: ['access-users', projectId], queryFn: () => listUsers(projectId) })
  const members = useMemo(
    () =>
      (users.data ?? [])
        .filter((u) => !u.is_superadmin && u.is_active)
        .map((u) => ({ ...u, role: u.memberships.find((m) => m.project_id === projectId)?.role as ProjectRole | undefined }))
        .filter((u) => !!u.role),
    [users.data, projectId],
  )
  const [userId, setUserId] = useState<string>()
  const user = members.find((u) => u.id === userId)
  // pilihan yang belum disimpan: kunci → pilihan
  const [desired, setDesired] = useState<Record<string, UserChoice>>({})

  const storedOf = (key: string): UserChoice => toChoice(userId ? idx.user[userId]?.[key] : undefined)
  const choiceOf = (key: string): UserChoice => desired[key] ?? storedOf(key)

  const changes = useMemo<AccessChange[]>(() => {
    const out: AccessChange[] = []
    for (const [key, c] of Object.entries(desired)) {
      if (c === toChoice(userId ? idx.user[userId]?.[key] : undefined)) continue
      out.push({ page_key: key, allowed: c === 'inherit' ? null : c === 'allow' })
    }
    return out
  }, [desired, idx, userId])

  const save = useMutation({
    mutationFn: async () => {
      for (let i = 0; i < changes.length; i += CHUNK) {
        await saveAccessRules({ project_id: projectId, subject_type: 'user', user_id: userId, changes: changes.slice(i, i + CHUNK) })
      }
    },
    onSuccess: () => {
      message.success('Hak akses user disimpan')
      setDesired({})
      onSaved()
    },
    onError: (e) => message.error(errorMessage(e)),
  })

  const pick = (id: string) => {
    setUserId(id)
    setDesired({})
  }
  const customCount = userId ? Object.keys(idx.user[userId] ?? {}).length : 0

  return (
    <>
      <div className={styles.toolbar}>
        <Select
          showSearch={{ optionFilterProp: 'label' }}
          style={{ minWidth: 320 }}
          placeholder="Pilih user"
          loading={users.isLoading}
          value={userId}
          onChange={pick}
          options={members.map((u) => ({ value: u.id, label: `${u.name} · ${u.email} (${ROLES.find((r) => r.value === u.role)?.label})` }))}
        />
        <Space>
          <Button disabled={!changes.length} onClick={() => setDesired({})}>Batalkan</Button>
          <Button type="primary" disabled={!changes.length} loading={save.isPending} onClick={() => save.mutate()}>
            Simpan{changes.length ? ` (${changes.length})` : ''}
          </Button>
        </Space>
      </div>

      {!user ? (
        <Alert type="info" showIcon title="Pilih user untuk mengatur pengecualian halaman. Pilihan lain mengikuti role-nya di project ini." />
      ) : (
        <>
          <div className={styles.hint} style={{ marginBottom: 8 }}>
            {user.name} berperan <b>{ROLES.find((r) => r.value === user.role)?.label}</b> di project ini
            {customCount ? <> · <Tag color="blue">{customCount} pengecualian</Tag></> : ' · belum ada pengecualian'}
          </div>
          {CATALOG.map((g) => (
            <Card key={g.title} size="small" title={g.title} className={styles.group}>
              <Table<CatalogItem>
                size="small"
                pagination={false}
                rowKey="key"
                dataSource={g.items}
                columns={[
                  { title: 'Halaman', dataIndex: 'title', render: (t: string, r) => <><b>{t}</b> <span className={styles.key}>{r.key}</span></> },
                  {
                    title: 'Pengaturan',
                    width: 270,
                    render: (_: unknown, item) => {
                      const def = defaultAllowed(user.role!, item)
                      const lockAllow = !canBeGranted(item) && !def
                      return (
                        <Radio.Group
                          size="small"
                          optionType="button"
                          buttonStyle="solid"
                          disabled={save.isPending}
                          value={choiceOf(item.key)}
                          onChange={(e) => setDesired((d) => ({ ...d, [item.key]: e.target.value as UserChoice }))}
                          options={[
                            { label: 'Ikut role', value: 'inherit' },
                            { label: 'Izinkan', value: 'allow', disabled: lockAllow },
                            { label: 'Blokir', value: 'block' },
                          ]}
                        />
                      )
                    },
                  },
                  {
                    title: 'Hasil',
                    width: 90,
                    align: 'center',
                    render: (_: unknown, item) => {
                      const ok = userAllowed(user.role!, item, idx.role[user.role!], choiceOf(item.key))
                      return <Tag color={ok ? 'green' : 'red'}>{ok ? 'Boleh' : 'Tidak'}</Tag>
                    },
                  },
                ]}
              />
            </Card>
          ))}
        </>
      )}
    </>
  )
}
