import type { ReactNode } from 'react'
import { Typography } from 'antd'
import { tokens } from './theme'

interface Props {
  title: ReactNode
  description?: ReactNode
  extra?: ReactNode
  /** 标题下方的概要区（如指标卡条），透传渲染，页面自备内容 */
  summary?: ReactNode
  /** 内容卡上方、概要区下方的工具条（如分段 Tab / 搜索筛选区） */
  toolbar?: ReactNode
  children: ReactNode
}

/**
 * 管理页统一骨架：页标题 + 描述 + 右上操作区，下方白色内容卡。
 * 内容卡为 hairline-only（1px 边框 + 8px 圆角 + 20px 内边距），不使用阴影。
 *
 * - summary：标题与内容卡之间的概要区（如指标卡），页面自备；
 * - toolbar：内容卡顶部工具条（分段 Tab / 搜索区），与内容卡共享边框，保证 D 版
 *   「分段 Tab 紧贴表格」的观感。
 */
export function PageContainer({
  title,
  description,
  extra,
  summary,
  toolbar,
  children,
}: Props) {
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
      {summary}
      <div
        style={{
          background: tokens.cardBg,
          border: `1px solid ${tokens.border}`,
          borderRadius: tokens.radiusLg,
          overflow: toolbar ? 'hidden' : undefined,
        }}
      >
        {toolbar ? (
          <div
            style={{
              padding: '10px 16px',
              borderBottom: `1px solid ${tokens.border}`,
            }}
          >
            {toolbar}
          </div>
        ) : null}
        <div style={{ padding: 20 }}>{children}</div>
      </div>
    </div>
  )
}
