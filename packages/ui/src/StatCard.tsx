import type { ReactNode } from 'react'
import { tokens } from './theme'

interface Props {
  /** 指标名称 */
  label: ReactNode
  /** 指标数值（已格式化好） */
  value: ReactNode
  /** 数值下方的说明，如「近 7 天 +4.2%」 */
  hint?: ReactNode
  /** 左侧彩条颜色，默认主色；可传 tokens.success / tokens.warning 等 */
  accent?: string
  /** 说明文字是否为警示语义（如「3 个分类为空」） */
  danger?: boolean
}

/** 单个指标卡：左侧 3px 彩条 + 名称 + 大号数值 + 说明，用于页面 summary 区。 */
export function StatCard({ label, value, hint, accent, danger }: Props) {
  return (
    <div
      style={{
        position: 'relative',
        background: tokens.cardBg,
        border: `1px solid ${tokens.border}`,
        borderRadius: 12,
        padding: '14px 16px 14px 18px',
        overflow: 'hidden',
      }}
    >
      <span
        style={{
          position: 'absolute',
          left: 0,
          top: 0,
          bottom: 0,
          width: 3,
          background: accent ?? tokens.primary,
        }}
      />
      <div style={{ fontSize: 12.5, color: tokens.textSecondary }}>{label}</div>
      <div
        style={{
          fontSize: 25,
          fontWeight: 700,
          letterSpacing: '-0.03em',
          marginTop: 8,
          lineHeight: 1.1,
          color: tokens.text,
        }}
      >
        {value}
      </div>
      {hint ? (
        <div
          style={{
            fontSize: 12,
            marginTop: 8,
            color: danger ? tokens.error : tokens.success,
          }}
        >
          {hint}
        </div>
      ) : null}
    </div>
  )
}

interface RowProps {
  children: ReactNode
}

/** 指标卡容器：等宽四列网格（窄屏换行），由页面在 PageContainer.summary 中传入。 */
export function StatRow({ children }: RowProps) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
        gap: 14,
      }}
    >
      {children}
    </div>
  )
}
