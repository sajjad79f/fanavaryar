import { useEffect, useState } from 'react'
import { api, kindTitles } from '../api/client'
import type { LocationNode } from '../api/client'

export default function Home() {
  const [tree, setTree] = useState<LocationNode[] | null>(null)

  useEffect(() => {
    api
      .get<LocationNode[]>('/api/locations?tree=true')
      .then(setTree)
      .catch(() => setTree([]))
  }, [])

  const buildingCount = tree?.length ?? 0

  return (
    <div>
      <h1>داشبورد</h1>
      <p>
        خوش آمدید. این نسخه اسکلت فاز ۱ است: هسته کاربران و مجوزها، مکان‌ها،
        تجهیزات و گزارش‌ها آماده است و ماژول‌های فرم‌ساز، اسناد و برنامه‌ریزی
        طبق سند معماری در فازهای بعدی اضافه می‌شوند.
      </p>
      <div className="cards">
        <div className="card">
          <div className="card-num">{buildingCount}</div>
          <div>ساختمان ثبت‌شده</div>
        </div>
        <div className="card">
          <div className="card-num">—</div>
          <div>تجهیزات (از صفحه تجهیزات ببینید)</div>
        </div>
        <div className="card">
          <div className="card-num">—</div>
          <div>گزارش‌های در انتظار تأیید</div>
        </div>
      </div>

      {tree && tree.length > 0 && (
        <>
          <h2>نمای کلی مکان‌ها</h2>
          <LocationTreeView nodes={tree} />
        </>
      )}
    </div>
  )
}

function LocationTreeView({ nodes }: { nodes: LocationNode[] }) {
  return (
    <ul className="tree">
      {nodes.map((n) => (
        <li key={n.id}>
          <strong>{n.name}</strong> <span className="muted">({kindTitles[n.kind]})</span>
          {n.children && n.children.length > 0 && <LocationTreeView nodes={n.children} />}
        </li>
      ))}
    </ul>
  )
}
