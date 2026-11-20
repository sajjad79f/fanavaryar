import { useEffect, useState } from 'react'
import { isoToJalaliText, parseJalali } from '../jalali'

/** ورودی تاریخ شمسی — مقدار داخلی ISO میلادی است */
export function JalaliInput({
  iso,
  onIso,
  disabled,
}: {
  iso: string
  onIso: (iso: string) => void
  disabled?: boolean
}) {
  const [text, setText] = useState(isoToJalaliText(iso))
  const [error, setError] = useState(false)

  useEffect(() => {
    setText(isoToJalaliText(iso))
    setError(false)
  }, [iso])

  function handleChange(value: string) {
    setText(value)
    if (value.trim() === '') {
      setError(false)
      onIso('')
      return
    }
    const parsed = parseJalali(value)
    if (parsed) {
      setError(false)
      onIso(parsed)
    } else {
      setError(true)
    }
  }

  return (
    <div className="jalali-input">
      <input
        className={`input${error ? ' input-error' : ''}`}
        type="text"
        inputMode="numeric"
        placeholder="1404/08/28"
        dir="ltr"
        disabled={disabled}
        value={text}
        onChange={(e) => handleChange(e.target.value)}
      />
      {error && <span className="hint-error">قالب تاریخ: 1404/08/28</span>}
    </div>
  )
}
