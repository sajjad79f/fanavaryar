import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { LocationRow } from '../api/types'
import { LOCATION_KINDS, labelOf } from '../api/types'
import { PageHeader, Modal, Alert } from '../components/ui'
import { can, useAuth } from '../context/AuthContext'

interface LocationNode extends LocationRow {
  children: LocationNode[]
}

function buildTree(rows: LocationRow[]): LocationNode[] {
  const nodes = new Map<number, LocationNode>(rows.map((r) => [r.id, { ...r, children: [] }]))
  const roots: LocationNode[] = []
  for (const node of nodes.values()) {
    if (node.parent_id !== null && nodes.has(node.parent_id)) {
      nodes.get(node.parent_id)!.children.push(node)
    } else {
      roots.push(node)
    }
  }
  const sortRec = (list: LocationNode[]) => {
    list.sort((a, b) => a.name.localeCompare(b.name, 'fa'))
    list.forEach((n) => sortRec(n.children))
  }
  sortRec(roots)
  return roots
}

export default function Locations() {
  const { user } = useAuth()
  const isAdmin = can(user, 'manage_locations')
  const [rows, setRows] = useState<LocationRow[]>([])
  const [error, setError] = useState('')
  const [modal, setModal] = useState<
    { kind: 'create'; parent: LocationNode | null } | { kind: 'rename'; node: LocationNode } | null
  >(null)

  const load = useCallback(() => {
    api
      .get<LocationRow[]>('/locations')
      .then(setRows)
      .catch((e) => setError(e instanceof Error ? e.message : 'خطا'))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const tree = buildTree(rows)

  return (
    <>
      <PageHeader
        title="مکان‌ها"
        subtitle="ساختار ساختمان، طبقه، اتاق و رک"
        actions={
          isAdmin && (
            <button className="btn btn-primary" onClick={() => setModal({ kind: 'create', parent: null })}>
              افزودن ساختمان
            </button>
          )
        }
      />
      {error && <Alert kind="error">{error}</Alert>}
      <div className="card tree-card">
        {tree.length === 0 ? (
          <p className="muted">هنوز مکانی ثبت نشده است؛ با «افزودن ساختمان» شروع کنید.</p>
        ) : (
          tree.map((node) => (
            <TreeNode
              key={node.id}
              node={node}
              isAdmin={isAdmin}
              onAdd={(parent) => setModal({ kind: 'create', parent })}
              onRename={(node2) => setModal({ kind: 'rename', node: node2 })}
              onDelete={async (node2) => {
                if (!window.confirm(`حذف «${node2.name}»؟`)) return
                try {
                  await api.delete(`/locations/${node2.id}`)
                  load()
                } catch (e) {
                  setError(e instanceof Error ? e.message : 'خطا')
                }
              }}
            />
          ))
        )}
      </div>

      {modal?.kind === 'create' && (
        <LocationFormModal
          parent={modal.parent}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
      {modal?.kind === 'rename' && (
        <RenameModal
          node={modal.node}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null)
            load()
          }}
        />
      )}
    </>
  )
}

function TreeNode({
  node,
  isAdmin,
  onAdd,
  onRename,
  onDelete,
}: {
  node: LocationNode
  isAdmin: boolean
  onAdd: (n: LocationNode) => void
  onRename: (n: LocationNode) => void
  onDelete: (n: LocationNode) => void
}) {
  const [open, setOpen] = useState(true)
  return (
    <div className="tree-node">
      <div className="tree-row">
        <span className={`kind-chip kind-${node.kind}`}>{labelOf(LOCATION_KINDS, node.kind)}</span>
        <button className="tree-toggle" onClick={() => setOpen(!open)} disabled={node.children.length === 0}>
          {node.children.length > 0 ? (open ? '−' : '+') : '•'}
        </button>
        <span className="tree-name">{node.name}</span>
        {node.code && <span className="muted small mono">({node.code})</span>}
        <span className="muted small">{node.assets_count} تجهیز</span>
        {isAdmin && (
          <span className="row-actions">
            {node.kind !== 'rack' && (
              <button className="btn btn-ghost small" onClick={() => onAdd(node)}>
                افزودن زیرمجموعه
              </button>
            )}
            <button className="btn btn-ghost small" onClick={() => onRename(node)}>تغییر نام</button>
            <button className="btn btn-danger-ghost small" onClick={() => onDelete(node)}>حذف</button>
          </span>
        )}
      </div>
      {open && node.children.map((c) => <TreeNode key={c.id} node={c} isAdmin={isAdmin} onAdd={onAdd} onRename={onRename} onDelete={onDelete} />)}
    </div>
  )
}

const CHILD_KIND: Record<string, string> = {
  building: 'floor',
  floor: 'room',
  room: 'rack',
}

function LocationFormModal({
  parent,
  onClose,
  onSaved,
}: {
  parent: LocationNode | null
  onClose: () => void
  onSaved: () => void
}) {
  const kind = parent ? CHILD_KIND[parent.kind] : 'building'
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!name.trim()) {
      setError('نام الزامی است')
      return
    }
    setBusy(true)
    try {
      await api.post('/locations', {
        name: name.trim(),
        kind,
        parent_id: parent?.id,
        code: code.trim() || undefined,
      })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title={parent ? `افزودن ${labelOf(LOCATION_KINDS, kind)} به «${parent.name}»` : 'افزودن ساختمان'} onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>نام *</label>
          <input className="input" autoFocus value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="form-row">
          <label>کد (اختیاری)</label>
          <input className="input" dir="ltr" value={code} onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? '...' : 'ذخیره'}</button>
        </div>
      </form>
    </Modal>
  )
}

function RenameModal({
  node,
  onClose,
  onSaved,
}: {
  node: LocationNode
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(node.name)
  const [code, setCode] = useState(node.code ?? '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!name.trim()) {
      setError('نام الزامی است')
      return
    }
    setBusy(true)
    try {
      await api.patch(`/locations/${node.id}`, { name: name.trim(), code: code.trim() || undefined })
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'خطا')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="تغییر نام مکان" onClose={onClose}>
      {error && <Alert kind="error">{error}</Alert>}
      <form onSubmit={submit}>
        <div className="form-row">
          <label>نام *</label>
          <input className="input" autoFocus value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="form-row">
          <label>کد</label>
          <input className="input" dir="ltr" value={code} onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>انصراف</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? '...' : 'ذخیره'}</button>
        </div>
      </form>
    </Modal>
  )
}
