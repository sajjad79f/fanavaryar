export type Role = 'admin' | 'supervisor' | 'user'

export interface UserInfo {
  id: number
  username: string
  full_name: string
  job_title: string | null
  role: Role
  is_superadmin: boolean
  /** کدهای مجوز کاربر از سرور — تنها مرجع کنترل نمایش در UI */
  permissions: string[]
}

export interface Paged<T> {
  items: T[]
  total: number
  page: number
  per_page: number
}

export interface LocationRow {
  id: number
  parent_id: number | null
  kind: 'building' | 'floor' | 'room' | 'rack'
  name: string
  code: string | null
  created_at: string
  assets_count: number
}

export interface Asset {
  id: number
  asset_tag: string
  name: string
  category: string
  status: string
  serial_number: string | null
  manufacturer: string | null
  model: string | null
  responsible_person: string | null
  current_location_id: number | null
  location_name: string | null
  notes: string | null
  spec_form_version_id: number | null
  created_at: string
  updated_at: string
}

export interface Movement {
  id: number
  from_location_id: number | null
  from_location_name: string | null
  to_location_id: number | null
  to_location_name: string | null
  reason: string | null
  moved_by_name: string | null
  created_at: string
}

export interface RelatedReport {
  id: number
  code: string | null
  title: string
  report_date: string
  status: string
}

export interface AssetDetail extends Asset {
  spec_version_number: number
  spec_template_name: string
  spec_fields: FieldDef[]
  spec_data: Record<string, unknown>
  movements: Movement[]
  reports: RelatedReport[]
}

export interface FieldDef {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'date' | 'select' | 'checkbox'
  required?: boolean
  options?: string[]
  min?: number
  max?: number
}

export type FormKind = 'report' | 'hardware_spec'

export interface FormTemplate {
  id: number
  code: string
  name: string
  description: string | null
  kind: FormKind
  is_active: boolean
  requires_approval: boolean
  current_version_id: number | null
  created_at: string
}

export interface FormVersion {
  id: number
  template_id: number
  version_number: number
  fields: FieldDef[]
  requires_approval: boolean
  created_by_name: string | null
  created_at: string
}

export interface FormDetail {
  id: number
  code: string
  name: string
  description: string | null
  kind: FormKind
  is_active: boolean
  requires_approval: boolean
  current_version_id: number | null
  created_at: string
  versions: FormVersion[]
}

export interface ReportListItem {
  id: number
  code: string | null
  title: string
  report_date: string
  status: 'pending' | 'approved' | 'rejected'
  created_by_name: string | null
  approved_by_name: string | null
  created_at: string
}

export interface ReportDetail {
  id: number
  code: string | null
  title: string
  report_date: string
  status: 'pending' | 'approved' | 'rejected'
  form_template_version_id: number
  version_number: number
  template_name: string
  fields: FieldDef[]
  form_data: Record<string, unknown>
  created_by_name: string | null
  approved_by_name: string | null
  approved_at: string | null
  reject_reason: string | null
  created_at: string
  assets: { id: number; asset_tag: string; name: string }[]
}

export interface UserItem {
  id: number
  username: string
  full_name: string
  job_title: string | null
  role: Role
  is_active: boolean
  created_at: string
}

export interface ScopeInfo {
  scope_type: 'all' | 'building' | 'floor' | 'room'
  location_id: number | null
}

export interface UserDetail extends UserItem {
  permissions: string[]
  scope: ScopeInfo | null
}

export interface PermissionDef {
  code: string
  module: string
  action: string
  title: string
  sort: number
}

export interface PermissionTemplate {
  id: number
  name: string
  created_at: string
  permissions: string[]
}

export interface AuditItem {
  id: number
  action: string
  entity_type: string | null
  entity_id: number | null
  details: unknown
  ip: string | null
  user_id: number | null
  username: string | null
  full_name: string | null
  created_at: string
}

export interface Stats {
  assets_total: number
  assets_active: number
  reports_pending: number
  reports_month: number
  forms_active: number
  users_active: number
}

export interface StatsResponse {
  stats: Stats
  pending_reports: {
    id: number
    code: string | null
    title: string
    report_date: string
    created_by_name: string | null
    created_at: string
  }[]
  recent_audit: {
    id: number
    action: string
    entity_type: string | null
    username: string | null
    created_at: string
  }[]
}

export const CATEGORIES: { value: string; label: string }[] = [
  { value: 'server', label: 'سرور' },
  { value: 'desktop', label: 'رایانه رومیزی' },
  { value: 'laptop', label: 'لپ‌تاپ' },
  { value: 'network', label: 'تجهیزات شبکه' },
  { value: 'printer', label: 'چاپگر' },
  { value: 'ups', label: 'UPS' },
  { value: 'other', label: 'سایر' },
]

export const ASSET_STATUSES: { value: string; label: string }[] = [
  { value: 'active', label: 'فعال' },
  { value: 'repair', label: 'در تعمیر' },
  { value: 'storage', label: 'انبار' },
  { value: 'retired', label: 'از رده خارج' },
]

export const LOCATION_KINDS: { value: string; label: string }[] = [
  { value: 'building', label: 'ساختمان' },
  { value: 'floor', label: 'طبقه' },
  { value: 'room', label: 'اتاق' },
  { value: 'rack', label: 'رک' },
]

export const REPORT_STATUSES: { value: string; label: string }[] = [
  { value: 'pending', label: 'در انتظار تایید' },
  { value: 'approved', label: 'تایید شده' },
  { value: 'rejected', label: 'رد شده' },
]

export const FORM_KINDS: { value: FormKind; label: string }[] = [
  { value: 'report', label: 'گزارش فعالیت' },
  { value: 'hardware_spec', label: 'مشخصات سخت‌افزاری تجهیز' },
]

export function formKindLabel(k: FormKind | string | undefined): string {
  return FORM_KINDS.find((x) => x.value === k)?.label ?? k ?? '—'
}

export const ROLES: { value: Role; label: string }[] = [
  { value: 'admin', label: 'مدیر سامانه' },
  { value: 'supervisor', label: 'سرپرست' },
  { value: 'user', label: 'کاربر' },
]

export function labelOf(list: { value: string; label: string }[], value?: string | null): string {
  if (!value) return '—'
  return list.find((x) => x.value === value)?.label ?? value
}

export const MODULE_LABELS: Record<string, string> = {
  assets: 'تجهیزات',
  locations: 'مکان‌ها',
  forms: 'فرم‌ها',
  reports: 'گزارش‌ها',
  audit: 'تاریخچه',
  users: 'کاربران',
  stats: 'آمار',
}

export const SCOPE_TYPE_LABELS: { value: ScopeInfo['scope_type']; label: string }[] = [
  { value: 'all', label: 'کل سازمان' },
  { value: 'building', label: 'ساختمان مشخص' },
  { value: 'floor', label: 'طبقه مشخص' },
  { value: 'room', label: 'اتاق مشخص' },
]

export const ACTION_LABELS: Record<string, string> = {
  'auth.login': 'ورود به سامانه',
  'auth.logout': 'خروج از سامانه',
  'auth.change_password': 'تغییر گذرواژه',
  'user.create': 'ایجاد کاربر',
  'user.update': 'ویرایش کاربر',
  'user.reset_password': 'بازنشانی گذرواژه کاربر',
  'user.set_permissions': 'تعیین مجوزهای کاربر',
  'user.set_scope': 'تعیین محدوده دسترسی کاربر',
  'perm.template_create': 'ایجاد الگوی دسترسی',
  'perm.template_update': 'ویرایش الگوی دسترسی',
  'perm.template_delete': 'حذف الگوی دسترسی',
  'location.create': 'ایجاد مکان',
  'location.update': 'ویرایش مکان',
  'location.delete': 'حذف مکان',
  'asset.create': 'ثبت تجهیز',
  'asset.update': 'ویرایش تجهیز',
  'asset.move': 'جابه‌جایی تجهیز',
  'form.create': 'ایجاد فرم',
  'form.update': 'ویرایش فرم',
  'form.new_version': 'ثبت نسخه جدید فرم',
  'report.create': 'ثبت گزارش',
  'report.approve': 'تایید گزارش',
  'report.reject': 'رد گزارش',
}

export function actionLabel(a: string): string {
  return ACTION_LABELS[a] ?? a
}
