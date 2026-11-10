import type { Locator } from '@playwright/test'

/**
 * 取 antd Descriptions 某一项的展示值（用于抽屉/详情里的结构化断言）。
 *
 * antd v6 把类名挂在单元格本身：`<th class="ant-descriptions-item-label">检查项数</th>`
 * + `<td class="ant-descriptions-item-content">2</td>`，两者是相邻兄弟。
 * 这里用「label 单元格 + 相邻兄弟」定位，不依赖列的排列顺序。
 */
export function descriptionsValue(scope: Locator, label: string): Locator {
  return scope.locator(
    `.ant-descriptions-item-label:has-text("${label}") + .ant-descriptions-item-content`,
  )
}
