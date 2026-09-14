import type { ReactNode } from 'react'
import { ConfigProvider, App as AntdApp } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { themeConfig } from './theme'

interface Props {
  children: ReactNode
}

/**
 * 应用外壳：注入 Ant Design 全局主题（中文语言包 + 设计令牌）。
 * 每个前端 app 的 main.tsx 用 <AppShell> 包裹，是全局主题的唯一挂载点。
 * 根容器开启 tabular-nums，让 ID/时间/数字等宽对齐（见 DESIGN.md）。
 */
export function AppShell({ children }: Props) {
  return (
    <div style={{ fontVariantNumeric: 'tabular-nums' }}>
      <ConfigProvider locale={zhCN} theme={themeConfig}>
        <AntdApp>{children}</AntdApp>
      </ConfigProvider>
    </div>
  )
}
