// Tab "Per Role": matriks halaman × role (Viewer, Dispatcher, Admin) untuk satu project.
import { useMemo, useState } from 'react'
import { Button, Card, Checkbox, Popconfirm, Space, Table, Tag, Tooltip, message } from 'antd'
import { useMutation } from '@tanstack/react-query'

import type { AccessChange, AccessRule } from '@/types/access.types'
import { saveAccessRules } from '@/services/api/access.api'
import { errorMessage } from '@/services/http/ApiError'

import styles from './access.module.css'
import { CATALOG, ROLES, canBeGranted, defaultAllowed, indexRules, roleAllowed, type CatalogItem } from './catalog'

const CHUNK = 50

export default function RoleMatrix({ projectId, rules, onSaved }: { projectId: string; rules: AccessRule[]; onSaved: () => void }) {
  const idx = useMemo(() => indexRules(rules), [rules])
  // pilihan yang belum disimpan: "role|kunci" → boleh/tidak
  const [desired, setDesired] = useState<Record<string, boolean>>({})

  const valueOf = (role: (typeof ROLES)[number]['value'], item: CatalogItem) => desired[`${role}|${item.key}`] ?? roleAllowed(role, item, idx.role[role])

  /** Perubahan yang perlu dikirim per role: bandingkan pilihan dengan aturan tersimpan. */
  const changesByRole = useMemo(() => {
    const out: Record<string, AccessChange[]> = {}
    for (const role of ROLES) {
      for (const g of CATALOG) {
        for (const item of g.items) {
          const d = desired[`${role.value}|${item.key}`]
          if (d === undefined) continue
          const target = d === defaultAllowed(role.value, item) ? null : d // sama dengan bawaan = hapus aturan
          const stored = idx.role[role.value]?.[item.key] ?? null
          if (target !== stored) (out[role.value] ??= []).push({ page_key: item.key, allowed: target })
        }
      }
    }
    return out
  }, [desired, idx])
  const pending = Object.values(changesByRole).reduce((n, l) => n + l.length, 0)

  const save = useMutation({
    mutationFn: async (byRole: Record<string, AccessChange[]>) => {
      for (const [role, changes] of Object.entries(byRole)) {
        for (let i = 0; i < changes.length; i += CHUNK) {
          await saveAccessRules({ project_id: projectId, subject_type: 'role', role: role as 'admin', changes: changes.slice(i, i + CHUNK) })
        }
      }
    },
    onSuccess: () => {
      message.success('Hak akses disimpan')
      setDesired({})
      onSaved()
    },
    onError: (e) => message.error(errorMessage(e)),
  })

  /** Kembalikan SEMUA aturan role di project ini ke bawaan. */
  const resetAll = () => {
    const byRole: Record<string, AccessChange[]> = {}
    for (const [role, map] of Object.entries(idx.role)) byRole[role] = Object.keys(map).map((k) => ({ page_key: k, allowed: null }))
    if (Object.values(byRole).some((l) => l.length)) save.mutate(byRole)
    else message.info('Semua sudah memakai aturan bawaan')
  }

  return (
    <>
      <div className={styles.toolbar}>
        <span className={styles.hint}>
          Centang = boleh membuka. Kotak berwarna = berbeda dari aturan bawaan. Superadmin selalu penuh.
        </span>
        <Space>
          <Button disabled={!pending} onClick={() => setDesired({})}>Batalkan</Button>
          <Popconfirm title="Kembalikan semua aturan role di project ini ke bawaan?" okText="Kembalikan" cancelText="Batal" onConfirm={resetAll}>
            <Button danger disabled={save.isPending}>Kembalikan ke bawaan</Button>
          </Popconfirm>
          <Button type="primary" disabled={!pending} loading={save.isPending} onClick={() => save.mutate(changesByRole)}>
            Simpan{pending ? ` (${pending})` : ''}
          </Button>
        </Space>
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
              ...ROLES.map((role) => ({
                title: role.label,
                width: 110,
                align: 'center' as const,
                render: (_: unknown, item: CatalogItem) => {
                  const value = valueOf(role.value, item)
                  const def = defaultAllowed(role.value, item)
                  const locked = !canBeGranted(item) && !def
                  const box = (
                    <span className={value !== def ? styles.changed : undefined}>
                      <Checkbox
                        checked={value}
                        disabled={locked || save.isPending}
                        onChange={(e) => setDesired((d) => ({ ...d, [`${role.value}|${item.key}`]: e.target.checked }))}
                      />
                    </span>
                  )
                  return locked ? <Tooltip title="Halaman data master hanya untuk admin; tidak bisa dibukakan untuk role ini.">{box}</Tooltip> : box
                },
              })),
            ]}
          />
        </Card>
      ))}
      <Tag className={styles.footnote}>Halaman Hak Akses & profil tidak bisa dibatasi dari sini.</Tag>
    </>
  )
}
