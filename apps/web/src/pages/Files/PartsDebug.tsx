import { useState } from 'react'
import { App, Button, Input, InputNumber, Space, Table, Tooltip, type TableColumnsType } from 'antd'
import { tokens } from '@ark-go-starter/ui'
import { listFileParts } from '@ark-go-starter/api'
import type { ListPartsResp, MultipartPartInfo } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'

/** 等宽字体样式（file_id 专用） */
const monoStyle = { fontFamily: 'ui-monospace, monospace' as const }

export interface PartsDebugProps {
  // 默认带入的 file_id（当前详情 id；无详情时为空，可手工输入）
  defaultFileId: string
}

/**
 * ④ 分片会话调试：调用 GET /demo/files/{id}/parts 列举已上传分片。
 * 普通上传的文件没有分片会话，会返回空列表或报错，此时用中文空态/提示说明。
 */
export default function PartsDebug({ defaultFileId }: PartsDebugProps) {
  const { message } = App.useApp()
  // 目标 file_id（默认带入当前详情 id）
  const [fileId, setFileId] = useState(defaultFileId)
  // 单页最大条数（可选）
  const [maxParts, setMaxParts] = useState<number | null>(null)
  // 分页游标（可选）
  const [partNumberMarker, setPartNumberMarker] = useState<number | null>(null)
  // 列举结果
  const [result, setResult] = useState<ListPartsResp | null>(null)
  // 请求中状态
  const [loading, setLoading] = useState(false)
  // 是否已查询过（用于空态文案与「不是分片上传」提示的区分）
  const [queried, setQueried] = useState(false)

  /** 列举分片（marker 传入时用于「下一页」） */
  const runList = async (marker?: number) => {
    const id = fileId.trim()
    if (!id) {
      message.warning('请输入文件 ID')
      return
    }
    setLoading(true)
    try {
      const data = await listFileParts(id, {
        maxParts: maxParts ?? undefined,
        partNumberMarker: marker ?? partNumberMarker ?? undefined,
      })
      setResult(data)
      setQueried(true)
      if (!data.parts.length) {
        message.info('该文件不是分片上传，无分片可列举')
      }
    } catch (e) {
      // 普通上传文件列举分片会报错，按「不是分片上传」提示，避免把后端英文原文暴露给用户
      setResult(null)
      setQueried(true)
      message.info(
        localizeError(e, '该文件不是分片上传，无分片可列举'),
      )
    } finally {
      setLoading(false)
    }
  }

  /** 点击「下一页」：用 next_part_number_marker 继续列举 */
  const handleNextPage = () => {
    if (!result?.next_part_number_marker) return
    setPartNumberMarker(result.next_part_number_marker)
    void runList(result.next_part_number_marker)
  }

  const columns: TableColumnsType<MultipartPartInfo> = [
    {
      title: '分片编号',
      dataIndex: 'part_number',
      width: 120,
      render: (value: number) => <span style={monoStyle}>{value}</span>,
    },
    {
      title: 'ETag',
      dataIndex: 'etag',
      ellipsis: { showTitle: false },
      render: (value: string) => (
        <Tooltip title={value}>
          <span style={monoStyle}>{value}</span>
        </Tooltip>
      ),
    },
  ]

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <div style={{ color: tokens.text, fontSize: tokens.fontSizeBody, fontWeight: 500 }}>分片会话调试</div>
      <Space wrap align="end">
        <div>
          <div style={{ color: tokens.textSecondary, fontSize: tokens.fontSizeTag, marginBottom: 4 }}>文件 ID</div>
          <Input
            placeholder="输入要列举分片的文件 ID"
            value={fileId}
            allowClear
            onChange={(e) => setFileId(e.target.value)}
            style={{ ...monoStyle, width: 320 }}
          />
        </div>
        <div>
          <div style={{ color: tokens.textSecondary, fontSize: tokens.fontSizeTag, marginBottom: 4 }}>max_parts</div>
          <InputNumber
            min={1}
            placeholder="可选"
            value={maxParts}
            onChange={(v) => setMaxParts(v)}
            style={{ width: 120 }}
          />
        </div>
        <div>
          <div style={{ color: tokens.textSecondary, fontSize: tokens.fontSizeTag, marginBottom: 4 }}>
            part_number_marker
          </div>
          <InputNumber
            min={1}
            placeholder="可选"
            value={partNumberMarker}
            onChange={(v) => setPartNumberMarker(v)}
            style={{ width: 140 }}
          />
        </div>
        <Button type="primary" loading={loading} onClick={() => void runList()}>
          列举分片
        </Button>
      </Space>

      <Table<MultipartPartInfo>
        rowKey="part_number"
        size="small"
        columns={columns}
        dataSource={result?.parts ?? []}
        loading={loading}
        tableLayout="fixed"
        scroll={{ x: 420 }}
        pagination={false}
        locale={{
          emptyText: queried ? '该文件不是分片上传，无分片可列举' : '尚未列举分片',
        }}
      />

      {result?.is_truncated ? (
        <Space size={8} wrap>
          <span style={{ color: tokens.textSecondary, fontSize: tokens.fontSizeTag }}>
            结果被截断，下一页起点：next_part_number_marker =
            <span style={monoStyle}>{result.next_part_number_marker ?? '-'}</span>
          </span>
          <Button size="small" onClick={handleNextPage} disabled={!result.next_part_number_marker}>
            下一页
          </Button>
        </Space>
      ) : null}
    </Space>
  )
}
