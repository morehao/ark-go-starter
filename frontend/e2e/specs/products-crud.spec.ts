import { test, expect } from '../fixtures/base'
import { uniqueName, deleteProduct, listProducts } from '../support/api'

/**
 * 商品 CRUD 主链路：新建 → 列表出现 → 编辑 → 行更新 → 删除 → 行消失。
 *
 * 说明：
 * - 导航：当前应用用 state 切页，从默认的「商品管理」页开始即可。
 * - 表格行动态，用唯一名称定位行，再在该行内点「编辑/删除」，避免依赖自增 ID。
 * - 造数后登记到 createdIDs，afterEach 兜底清理，防止中断留下脏数据。
 */
test.describe('商品管理 · CRUD', () => {
  test.describe.configure({ mode: 'serial' })

  const createdIDs: number[] = []

  test.afterEach(async () => {
    // 兜底清理：删除本用例通过接口造出的商品（页面创建的由用例内 UI 删除）
    for (const id of createdIDs.splice(0)) {
      await deleteProduct(id).catch(() => undefined)
    }
  })

  test('新建 → 编辑 → 删除 全流程', async ({ page }) => {
    const name = uniqueName('e2e-product')
    const updatedName = `${name}-upd`

    await page.goto('/')
    // 等商品表格首屏加载完成（表头「商品管理」标题可见）
    await expect(page.getByRole('heading', { name: '商品管理', level: 5 })).toBeVisible()

    // 新建
    await page.getByTestId('product-create-btn').click()
    const modal = page.getByRole('dialog')
    await expect(modal.getByText('新建商品')).toBeVisible()
    await modal.getByPlaceholder('请输入商品名称').fill(name)
    await modal.getByPlaceholder('请输入分类 ID').fill('22')
    await modal.getByPlaceholder('请输入价格').fill('19.9')
    await modal.getByRole('button', { name: /确\s*定|OK/i }).click()

    // 成功后弹窗关闭，列表出现该行
    await expect(modal).toBeHidden()
    const row = page.getByRole('row', { name: new RegExp(name) })
    await expect(row).toBeVisible()
    await expect(row).toContainText('22')

    // 记录 ID 便于兜底清理
    const data = await listProducts(1, 100)
    const created = data.list.find((p) => p.name === name)
    if (created) createdIDs.push(created.productID)

    // 编辑（在行内点编辑）
    await row.getByRole('button', { name: '编辑' }).click()
    const editModal = page.getByRole('dialog')
    await expect(editModal.getByText('编辑商品')).toBeVisible()
    await editModal.getByPlaceholder('请输入商品名称').fill(updatedName)
    await editModal.getByPlaceholder('请输入价格').fill('29.9')
    await editModal.getByRole('button', { name: /确\s*定|OK/i }).click()
    await expect(editModal).toBeHidden()

    // 行名更新
    const updatedRow = page.getByRole('row', { name: new RegExp(updatedName) })
    await expect(updatedRow).toBeVisible()
    await expect(updatedRow).toContainText('29.9')

    // 删除（Popconfirm 二次确认）
    await updatedRow.getByRole('button', { name: '删除' }).click()
    await page.getByRole('button', { name: /确\s*定|OK/i }).click()

    // 行消失
    await expect(page.getByRole('row', { name: new RegExp(updatedName) })).toHaveCount(0)

    // 已删除则无需兜底清理
    createdIDs.length = 0
  })
})
