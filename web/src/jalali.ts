// تبدیل تاریخ میلادی ↔ شمسی (الگوریتم رایج jdf — هم‌سان با پیاده‌سازی بک‌اند در backend/src/jalali.rs)

const G_D_M = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334]

export function toJalali(gy: number, gm: number, gd: number): [number, number, number] {
  const gy2 = gm > 2 ? gy + 1 : gy
  let days =
    355666 + 365 * gy + Math.floor((gy2 + 3) / 4) - Math.floor((gy2 + 99) / 100) +
    Math.floor((gy2 + 399) / 400) + gd + G_D_M[gm - 1]
  let jy = -1595 + 33 * Math.floor(days / 12053)
  days %= 12053
  jy += 4 * Math.floor(days / 1461)
  days %= 1461
  if (days > 365) {
    jy += Math.floor((days - 1) / 365)
    days = (days - 1) % 365
  }
  if (days < 186) {
    return [jy, 1 + Math.floor(days / 31), 1 + (days % 31)]
  }
  return [jy, 7 + Math.floor((days - 186) / 30), 1 + ((days - 186) % 30)]
}

export function toGregorian(jyIn: number, jm: number, jd: number): [number, number, number] {
  const jy = jyIn + 1595
  let days: number =
    -355668 + 365 * jy + Math.floor(jy / 33) * 8 + Math.floor(((jy % 33) + 3) / 4) + jd +
    (jm < 7 ? (jm - 1) * 31 : (jm - 7) * 30 + 186)
  let gy: number = 400 * Math.floor(days / 146097)
  days %= 146097
  if (days > 36524) {
    days -= 1
    gy += 100 * Math.floor(days / 36524)
    days %= 36524
    if (days >= 365) days += 1
  }
  gy += 4 * Math.floor(days / 1461)
  days %= 1461
  if (days > 365) {
    gy += Math.floor((days - 1) / 365)
    days = (days - 1) % 365
  }
  let gd = days + 1
  const leap = (gy % 4 === 0 && gy % 100 !== 0) || gy % 400 === 0
  const months = [0, 31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]
  let gm = 0
  while (gm < 13 && gd > months[gm]) {
    gd -= months[gm]
    gm += 1
  }
  return [gy, gm, gd]
}

export const JALALI_MONTHS = [
  'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور',
  'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند',
]

const pad = (n: number) => String(n).padStart(2, '0')

function isoParts(value: string): [number, number, number] | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value)
  if (m) return [+m[1], +m[2], +m[3]]
  const d = new Date(value)
  if (isNaN(d.getTime())) return null
  return [d.getFullYear(), d.getMonth() + 1, d.getDate()]
}

/** نمایش تاریخ شمسی از مقدار ISO (تاریخ یا زمان کامل) — مثال: 1404/08/28 */
export function formatJalali(value?: string | null): string {
  if (!value) return '—'
  const p = isoParts(value)
  if (!p) return value
  const [jy, jm, jd] = toJalali(p[0], p[1], p[2])
  return `${jy}/${pad(jm)}/${pad(jd)}`
}

/** نمایش تاریخ شمسی بلند — مثال: ۲۸ آبان ۱۴۰۴ */
export function formatJalaliLong(value?: string | null): string {
  if (!value) return '—'
  const p = isoParts(value)
  if (!p) return value
  const [jy, jm, jd] = toJalali(p[0], p[1], p[2])
  return `${jd} ${JALALI_MONTHS[jm - 1]} ${jy}`
}

/** تاریخ و ساعت شمسی — مثال: 1404/08/28 14:30 */
export function formatJalaliDateTime(value?: string | null): string {
  if (!value) return '—'
  const p = isoParts(value)
  if (!p) return value
  const dt = new Date(value)
  const time = isNaN(dt.getTime()) ? '' : ` ${pad(dt.getHours())}:${pad(dt.getMinutes())}`
  const [jy, jm, jd] = toJalali(p[0], p[1], p[2])
  return `${jy}/${pad(jm)}/${pad(jd)}${time}`
}

/** تبدیل ورودی شمسی «1404/08/28» به ISO میلادی؛ ورودی نامعتبر → null */
export function parseJalali(text: string): string | null {
  const m = /^(\d{4})\/(\d{1,2})\/(\d{1,2})$/.exec(text.trim())
  if (!m) return null
  const jy = +m[1], jm = +m[2], jd = +m[3]
  if (jm < 1 || jm > 12 || jd < 1 || jd > 31 || jy < 1100 || jy > 1600) return null
  const [gy, gm, gd] = toGregorian(jy, jm, jd)
  const back = toJalali(gy, gm, gd)
  if (back[0] !== jy || back[1] !== jm || back[2] !== jd) return null // مثل 30 اسفند سال غیر کبیسه
  return `${gy}-${pad(gm)}-${pad(gd)}`
}

/** تبدیل ISO میلادی به نمایش شمسی «1404/08/28» */
export function isoToJalaliText(iso?: string | null): string {
  if (!iso) return ''
  const p = isoParts(iso)
  if (!p) return ''
  const [jy, jm, jd] = toJalali(p[0], p[1], p[2])
  return `${jy}/${pad(jm)}/${pad(jd)}`
}

/** تاریخ امروز به‌صورت ISO میلادی */
export function todayIso(): string {
  const d = new Date()
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
