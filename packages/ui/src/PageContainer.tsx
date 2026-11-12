import type { ReactNode } from 'react'
import { Typography } from 'antd'
import { tokens } from './theme'

interface Props {
  title: ReactNode
  description?: ReactNode
  extra?: ReactNode
  children: ReactNode
}

/**
 * 管理页统一骨架：页标题 + 描述 + 右上操作区，下方白色内容卡。
 * 内容卡为 hairline-only（1px 冷调边框 + 12px 圆角 + 20px 内边距），不使用阴影。
 */
export function PageContainer({ title, description, extra, children }: Props) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 16,
        }}
      >
        <div>
          <Typography.Title level={4} style={{ margin: 0 }}>
            {title}
          </Typography.Title>
          {description ? (
            <Typography.Text type="secondary">{description}</Typography.Text>
          ) : null}
        </div>
        {extra}
      </div>
      <div
        style={{
          background: tokens.cardBg,
          border: `1px solid ${tokens.border}`,
          borderRadius: 12,
          padding: 20,
        }}
      >
        {children}
      </div>
    </div>
  )
}
