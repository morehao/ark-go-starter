import { useState } from 'react'
import { App, Button, Descriptions, Input, Space, Tooltip } from 'antd'
import { tokens } from '@ark-go-starter/ui'
import {
  presignDownload,
  redirectFileURL,
  redirectFileURLByQuery,
  serveFileURL,
  serveFileURLByQuery,
} from '@ark-go-starter/api'
import type { PresignURLResp } from '@ark-go-starter/types'
import { localizeError } from '../../utils/errorMessage'
import { toSameOriginURL } from './multipart'

/** 等宽字体样式（URL 专用） */
const monoStyle = { fontFamily: 'ui-monospace, monospace' as const }

/** 以新标签打开同源 URL（noopener 防被打开页面回控当前页） */
function openTab(url: string) {
  window.open(url, '_blank', 'noopener')
}

export interface AccessMethodsProps {
  // 当前详情里的文件 ID（用于 serve/redirect/presign 的路径形式）
  fileId: string
  // 当前详情里的存储 URI（作为 query 形式的默认 storage_uri 回填）
  storageUri: string
}

/**
 * ③ 下载与访问方式：
 * 把 ginupload 提供的四种取回方式逐一可视化（serve 直出 / redirect 302 / presign-url / query 形式），
 * 并注明 /objects/{bucket}/{key} 是分片直传的底层端点、不提供手工入口。
 */
export default function AccessMethods({ fileId, storageUri }: AccessMethodsProps) {
  const { message } = App.useApp()
  // presign-url 返回结果（url/method/expires_in 回显）
  const [presign, setPresign] = useState<PresignURLResp | null>(null)
  // presign-url 请求中状态
  const [presignLoading, setPresignLoading] = useState(false)
  // query 形式的 file_id 输入
  const [queryFileId, setQueryFileId] = useState(fileId)
  // query 形式的 storage_uri 输入
  const [queryStorageUri, setQueryStorageUri] = useState(storageUri)

  /** presign-url：调用接口并把返回结果回显；「打开」时用同源改写 */
  const handlePresign = async () => {
    if (!fileId) return
    setPresignLoading(true)
    try {
      const data = await presignDownload(fileId)
      setPresign(data)
      message.success('已获取预签名下载地址')
    } catch (e) {
      message.error(localizeError(e, '获取预签名地址失败'))
    } finally {
      setPresignLoading(false)
    }
  }

  /** query 形式：至少填一个（file_id 或 storage_uri），拼出同源相对路径打开 */
  const resolveQueryArgs = (): { fileId?: string; storageUri?: string } | null => {
    const id = queryFileId.trim()
    const uri = queryStorageUri.trim()
    if (!id && !uri) {
      message.warning('请至少填写 file_id 或 storage_uri 其中一个')
      return null
    }
    return { fileId: id || undefined, storageUri: uri || undefined }
  }

  const handleServeQuery = () => {
    const args = resolveQueryArgs()
    if (args) openTab(serveFileURLByQuery(args))
  }

  const handleRedirectQuery = () => {
    const args = resolveQueryArgs()
    if (args) openTab(redirectFileURLByQuery(args))
  }

  /** 打开预签名地址：local 驱动是后端绝对地址，需同源改写后再导航 */
  const handleOpenPresign = () => {
    if (!presign) return
    openTab(toSameOriginURL(presign.url))
  }

  return (
    <Descriptions bordered column={1} size="small" title="下载与访问方式" styles={{ label: { width: 160, color: tokens.textSecondary } }}>
      <Descriptions.Item label="① serve 直出（路径）">
        <Space direction="vertical" size={4}>
          <Button size="small" onClick={() => openTab(serveFileURL(fileId))}>
            打开 serve 直出
          </Button>
          <span style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
            local 驱动直接把文件流写回浏览器：{serveFileURL(fileId)}
          </span>
        </Space>
      </Descriptions.Item>

      <Descriptions.Item label="② redirect 302（路径）">
        <Space direction="vertical" size={4}>
          <Button size="small" onClick={() => openTab(redirectFileURL(fileId))}>
            打开 redirect 302
          </Button>
          <span style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
            浏览器跟随 302 跳转到实际文件 URL：{redirectFileURL(fileId)}
          </span>
        </Space>
      </Descriptions.Item>

      <Descriptions.Item label="③ presign-url">
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Button size="small" loading={presignLoading} onClick={handlePresign}>
            获取预签名下载地址
          </Button>
          {presign ? (
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <div>
                <span style={{ color: tokens.textSecondary }}>URL：</span>
                <Tooltip title={presign.url}>
                  <span style={{ ...monoStyle, wordBreak: 'break-all' }}>{presign.url}</span>
                </Tooltip>
                <Button size="small" type="link" onClick={handleOpenPresign}>
                  打开
                </Button>
              </div>
              <div style={{ color: tokens.textSecondary, fontSize: 12 }}>
                方法：<span style={monoStyle}>{presign.method}</span>
                {'　'}有效期：{presign.expires_in} 秒
              </div>
            </Space>
          ) : null}
        </Space>
      </Descriptions.Item>

      <Descriptions.Item label="④ serve / redirect（query）">
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <Input
            placeholder="file_id（可选）"
            value={queryFileId}
            allowClear
            onChange={(e) => setQueryFileId(e.target.value)}
            style={monoStyle}
          />
          <Input
            placeholder="storage_uri（可选）"
            value={queryStorageUri}
            allowClear
            onChange={(e) => setQueryStorageUri(e.target.value)}
            style={monoStyle}
          />
          <Space wrap>
            <Button size="small" onClick={handleServeQuery}>
              打开 serve?query
            </Button>
            <Button size="small" onClick={handleRedirectQuery}>
              打开 redirect?query
            </Button>
          </Space>
          <span style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
            query 形式兼容 file_id 或 storage_uri，至少填写一个。
          </span>
        </Space>
      </Descriptions.Item>

      <Descriptions.Item label="存储协议端点">
        <span style={{ color: tokens.textSecondary, fontSize: 12 }}>
          /objects/&#123;bucket&#125;/&#123;key&#125; 是分片 PUT/GET 直传所走的底层存储协议端点，
          由预签名地址内部使用，前端不提供手工调用入口。
        </span>
      </Descriptions.Item>
    </Descriptions>
  )
}
