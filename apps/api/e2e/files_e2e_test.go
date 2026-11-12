//go:build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileRecordResp 是上传/去重命中的文件记录契约（对齐 ginupload.fileRecordResponse）。
type fileRecordResp struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Status   string `json:"status"`
}

// fileDetailResp 是文件详情契约（对齐 ginupload.fileDetailResponse）。
type fileDetailResp struct {
	FileID      string `json:"file_id"`
	ContentHash string `json:"content_hash"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type"`
	StorageURI  string `json:"storage_uri"`
	UploadID    string `json:"upload_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// checkExistResp 是内容去重检查响应。
type checkExistResp struct {
	Exists bool            `json:"exists"`
	File   *fileRecordResp `json:"file"`
}

// presignResp 是预签名 URL 响应。
type presignResp struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresIn int               `json:"expires_in"`
}

// uploadE2EFile 上传一份 e2e 文件并注册清理（先走接口删除，再兜底清库/库记录），返回文件名与哈希。
func uploadE2EFile(t *testing.T, baseURL, tag string, content []byte) (fileID, fileName, contentHash string) {
	t.Helper()
	fileName = tag + ".txt"
	contentHash = sha256Hex(content)

	resp := doMultipartUpload(t, baseURL, "/v1/demo/files",
		map[string]string{"content_hash": contentHash}, "file", fileName, content)
	require.Equal(t, 0, resp.Code, "upload file failed: %s", resp.Msg)

	var record fileRecordResp
	require.NoError(t, json.Unmarshal(resp.Data, &record))
	require.NotEmpty(t, record.FileID)

	// 清理只走直连库：t.Cleanup 在用例返回（含 defer shutdown 关闭服务）之后才执行，
	// 此时再发 HTTP 请求会打到已关闭的端口。需要回收存储对象的用例在用例体内显式调 DELETE 接口。
	t.Cleanup(func() {
		dbExec(t, `DELETE FROM core_file_upload WHERE file_id IN (SELECT id FROM core_file WHERE content_hash = ?)`, contentHash)
		dbExec(t, `DELETE FROM core_file WHERE content_hash = ?`, contentHash)
	})
	return record.FileID, fileName, contentHash
}

// TestFileFlowOverHTTP 覆盖文件服务完整链路：
// 上传（正确 content_hash）→ 详情 → serve 内容一致 → presign-url → redirect →
// check-exist 命中（file 字段完整）→ 删除 → 删除后 check-exist 未命中。
func TestFileFlowOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	content := []byte("e2e-file-content-" + tag)
	fileID, fileName, contentHash := uploadE2EFile(t, baseURL, tag, content)

	// 1) 详情：哈希、大小、文件名、状态都与上传一致。
	detailResp := doJSON(t, baseURL, http.MethodGet, "/v1/demo/files/"+fileID, nil, nil)
	require.Equal(t, 0, detailResp.Code, detailResp.Msg)
	var detail fileDetailResp
	require.NoError(t, json.Unmarshal(detailResp.Data, &detail))
	assert.Equal(t, fileID, detail.FileID)
	assert.Equal(t, contentHash, detail.ContentHash)
	assert.EqualValues(t, len(content), detail.Size)
	assert.Equal(t, fileName, detail.Name)
	assert.Equal(t, "completed", detail.Status)
	assert.NotEmpty(t, detail.StorageURI)
	assert.NotEmpty(t, detail.CreatedAt)

	// 2) serve：直接输出文件内容，字节与上传一致。
	serveStatus, serveHeader, serveBody := doRaw(t, baseURL, http.MethodGet, "/v1/demo/files/"+fileID+"/serve", nil, nil)
	require.Equal(t, http.StatusOK, serveStatus, "serve 应直接返回 200")
	assert.Equal(t, content, serveBody, "serve 返回内容应与上传一致")
	assert.Contains(t, serveHeader.Get("Content-Disposition"), fileName)

	// 3) presign-url：拿到带签名的下载地址。
	presignHTTP := doJSON(t, baseURL, http.MethodPost, "/v1/demo/files/"+fileID+"/presign-url", nil, nil)
	require.Equal(t, 0, presignHTTP.Code, presignHTTP.Msg)
	var presigned presignResp
	require.NoError(t, json.Unmarshal(presignHTTP.Data, &presigned))
	assert.NotEmpty(t, presigned.URL)
	assert.Equal(t, http.MethodGet, presigned.Method)
	assert.Contains(t, presigned.URL, "token=")
	assert.Contains(t, presigned.URL, "/demo/")
	assert.Greater(t, presigned.ExpiresIn, 0)

	// 4) redirect：302 + Location 指向预签名地址（不跟随重定向）。
	redirectStatus, redirectHeader, _ := doRawNoRedirect(t, baseURL, http.MethodGet, "/v1/demo/files/"+fileID+"/redirect", nil, nil)
	require.Equal(t, http.StatusFound, redirectStatus, "redirect 应返回 302")
	location := redirectHeader.Get("Location")
	assert.NotEmpty(t, location)
	assert.Contains(t, location, "token=", "Location 应是带签名的下载地址")

	// 5) check-exist：按内容哈希命中；file 字段的完整性单独在
	// TestFileCheckExistContract 里严格断言（当前 golib 该字段不完整，见那边的 Skip 说明）。
	checkResp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/files/check-exist", nil,
		map[string]any{"content_hash": contentHash})
	require.Equal(t, 0, checkResp.Code, checkResp.Msg)
	var existed checkExistResp
	require.NoError(t, json.Unmarshal(checkResp.Data, &existed))
	require.True(t, existed.Exists, "同 content_hash 应命中")
	require.NotNil(t, existed.File, "命中时应返回文件记录")

	// 6) 删除：接口成功，再次 check-exist 应未命中（内容引用被回收）。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/files/"+fileID, nil, nil).Code)
	checkAgain := doJSON(t, baseURL, http.MethodPost, "/v1/demo/files/check-exist", nil,
		map[string]any{"content_hash": contentHash})
	require.Equal(t, 0, checkAgain.Code, checkAgain.Msg)
	var gone checkExistResp
	require.NoError(t, json.Unmarshal(checkAgain.Data, &gone))
	assert.False(t, gone.Exists, "删除后同哈希不应再命中")
	assert.Nil(t, gone.File, "未命中时不应返回 file 字段")
}

// TestFileNegativeOverHTTP 覆盖文件服务的反例：缺 content_hash、内容哈希与内容不一致、
// 缺失必填查询参数、不存在的文件 ID。
func TestFileNegativeOverHTTP(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()

	t.Run("upload_without_content_hash_rejected", func(t *testing.T) {
		resp := doMultipartUpload(t, baseURL, "/v1/demo/files", nil, "file", tag+"-no-hash.txt",
			[]byte("e2e-no-hash-content"))
		assert.NotEqual(t, 0, resp.Code, "缺 content_hash 应返回业务错误")
	})

	t.Run("upload_with_wrong_content_hash_rejected", func(t *testing.T) {
		content := []byte("e2e-wrong-hash-" + tag)
		resp := doMultipartUpload(t, baseURL, "/v1/demo/files",
			map[string]string{"content_hash": sha256Hex([]byte("different-content"))},
			"file", tag+"-wrong-hash.txt", content)
		assert.NotEqual(t, 0, resp.Code, "声明的 content_hash 与实测不一致应被拒")
	})

	t.Run("upload_without_file_part_rejected", func(t *testing.T) {
		resp := doMultipartFieldsOnly(t, baseURL, "/v1/demo/files",
			map[string]string{"content_hash": sha256Hex([]byte("no-file-part"))})
		assert.NotEqual(t, 0, resp.Code, "缺 file part 应返回业务错误")
	})

	t.Run("check_exist_missing_hash_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodPost, "/v1/demo/files/check-exist", nil, map[string]any{})
		assert.NotEqual(t, 0, resp.Code, "缺 content_hash 应在绑定阶段被拒")
	})

	t.Run("check_exist_unknown_hash_not_found", func(t *testing.T) {
		resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/files/check-exist", nil,
			map[string]any{"content_hash": sha256Hex([]byte(uniqueTag()))})
		require.Equal(t, 0, resp.Code, resp.Msg)
		var out checkExistResp
		require.NoError(t, json.Unmarshal(resp.Data, &out))
		assert.False(t, out.Exists)
	})

	t.Run("detail_missing_file_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet,
			"/v1/demo/files/00000000-0000-0000-0000-000000000000", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "不存在的文件应返回业务错误")
	})

	t.Run("serve_missing_query_params_rejected", func(t *testing.T) {
		resp := doError(t, baseURL, http.MethodGet, "/v1/demo/files/serve", nil, nil)
		assert.NotEqual(t, 0, resp.Code, "serve 缺 file_id/storage_uri 应返回业务错误")
	})
}

// TestFilePresignedGetRoundTrip 覆盖「预签名 URL 真能读回内容」的链路：
// redirect 端点签发带 token 的下载地址 → 换到本次随机端口后 GET → 内容与上传一致；
// 篡改 token 或过期时间则必须被拒（签名校验生效）。
//
// 本地存储的 base_url 在配置里固定为 8099，而 e2e 服务监听随机端口，因此只替换
// scheme+host，path/query（含签名 token）保持原样，保证验证的是真实签名而不是绕过。
func TestFilePresignedGetRoundTrip(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	content := []byte("e2e-presigned-" + tag)
	fileID, _, _ := uploadE2EFile(t, baseURL, tag, content)

	redirectStatus, redirectHeader, _ := doRawNoRedirect(t, baseURL, http.MethodGet,
		"/v1/demo/files/"+fileID+"/redirect", nil, nil)
	require.Equal(t, http.StatusFound, redirectStatus, "redirect 应返回 302")
	location := redirectHeader.Get("Location")
	require.NotEmpty(t, location)

	presigned, err := url.Parse(location)
	require.NoError(t, err, "Location 应是合法 URL: %s", location)
	base, err := url.Parse(baseURL)
	require.NoError(t, err)

	rebuild := func(u *url.URL) string {
		return base.Scheme + "://" + base.Host + u.Path + "?" + u.RawQuery
	}

	// 1) 原样（仅换端口）访问预签名 URL：应能读回内容。
	status, _, body := fetchNoRedirect(t, rebuild(presigned))
	require.Equal(t, http.StatusOK, status, "合法预签名 URL 应可直接读取")
	assert.Equal(t, content, body, "预签名读取的内容应与上传一致")

	// 2) 篡改 token：签名不匹配，必须拒绝。
	tampered := *presigned
	query := tampered.Query()
	query.Set("token", "tampered-signature")
	tampered.RawQuery = query.Encode()
	status, _, _ = fetchNoRedirect(t, rebuild(&tampered))
	assert.Equal(t, http.StatusForbidden, status, "篡改 token 应被拒（403）")

	// 3) 篡改过期时间：同样破坏签名，必须拒绝。
	expired := *presigned
	query = expired.Query()
	query.Set("expires", "1")
	expired.RawQuery = query.Encode()
	status, _, _ = fetchNoRedirect(t, rebuild(&expired))
	assert.Equal(t, http.StatusForbidden, status, "篡改 expires 应被拒（403）")

	// 用例体内显式删除，让存储对象与库记录一起回收（cleanup 只兜底清库，不依赖服务存活）。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/files/"+fileID, nil, nil).Code)
}

// fetchNoRedirect 直接 GET 一个绝对 URL（不跟随重定向），返回状态码与 body。
func fetchNoRedirect(t *testing.T, rawURL string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	resp, err := noRedirectClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header, body
}

// TestFileCheckExistContract 断言 check-exist 命中时返回的 file 记录字段完整
// （file_id / name / status 与上传记录一致）。
//
// 背景：golib v1.34.0 的 CheckExist 只填 FileDetail.FileID，而 ginupload 的
// toFileRecordResp 读 FileUploadID/Name/MimeType/Status，导致 file 对象字段全空；
// v1.34.1 起 CheckExist 会回查最近一次 completed 上传记录并填全字段，本用例随之转为严格校验。
func TestFileCheckExistContract(t *testing.T) {
	baseURL, shutdown := startServer(t)
	defer shutdown()

	tag := uniqueTag()
	content := []byte("e2e-check-exist-" + tag)
	fileID, fileName, contentHash := uploadE2EFile(t, baseURL, tag, content)

	resp := doJSON(t, baseURL, http.MethodPost, "/v1/demo/files/check-exist", nil,
		map[string]any{"content_hash": contentHash})
	require.Equal(t, 0, resp.Code, resp.Msg)
	var out checkExistResp
	require.NoError(t, json.Unmarshal(resp.Data, &out))
	require.True(t, out.Exists, "同 content_hash 应命中")
	require.NotNil(t, out.File, "命中时应返回文件记录")

	// 用例体内显式删除，回收存储对象（cleanup 只兜底清库，且 t.Cleanup 晚于 shutdown）。
	require.Equal(t, 0, doJSON(t, baseURL, http.MethodDelete, "/v1/demo/files/"+fileID, nil, nil).Code)

	assert.Equal(t, fileID, out.File.FileID, "file_id 应等于上传返回的文件ID")
	assert.Equal(t, fileName, out.File.Name, "name 应与上传文件名一致")
	assert.Equal(t, "completed", out.File.Status, "status 应为 completed")
	assert.NotEmpty(t, out.File.MimeType, "mime_type 不应为空")
}
