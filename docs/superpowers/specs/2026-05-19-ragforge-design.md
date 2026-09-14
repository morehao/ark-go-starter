# RAGForge 设计文档

## 概述

将 WeKnora 的核心 RAG 后端服务迁移到 ark-go-starter 项目中，建立一个新的 `ragforge` 应用，遵循 ark-go-starter 的代码规范和架构模式。

## 迁移范围

从 WeKnora 迁移以下 8 个核心模块：

| 模块 | 说明 |
|------|------|
| 知识库管理 | KnowledgeBase CRUD + 配置（解析引擎、嵌入模型、索引策略） |
| 知识文档管理 | 文件上传、URL 抓取、手动录入、文档解析、下载/预览 |
| 文档切片管理 | Chunk CRUD + 向量检索（Hybrid Search）|
| FAQ 管理 | FAQ 条目 CRUD、导入导出、相似问题、语义搜索 |
| 对话/会话管理 | Session CRUD、消息加载、标题生成、Knowledge QA |
| 模型管理 | Model Provider CRUD、凭据管理、类型管理 |
| 向量存储管理 | VectorStore 类型管理、CRUD、连接测试 |
| 标签管理 | 知识标签的增删改查 |

## 架构

```
apps/ragforge/
├── app.go                     # AppName + Routers()
├── go.mod                     # Go module (github.com/morehao/ark-go-starter/apps/ragforge)
├── cmd/
│   ├── main.go               # 入口 main()
│   └── init.go               # serverInit() 初始化配置/日志/DB/引擎
├── config/
│   ├── config.go             # Config 结构体 + InitConf()
│   ├── config.yaml           # 开发环境配置
│   └── config.prod.yaml      # 生产环境配置
├── internal/
│   ├── controller/
│   │   ├── ctrkb/            # 知识库控制器
│   │   ├── ctrknowledge/     # 知识文档控制器
│   │   ├── ctrchunk/         # 文档切片控制器
│   │   ├── ctrfaq/           # FAQ 控制器
│   │   ├── ctrsession/       # 会话控制器
│   │   ├── ctrmessage/       # 消息控制器
│   │   ├── ctrmodel/         # 模型控制器
│   │   ├── ctrvectorstore/   # 向量存储控制器
│   │   └── ctrtag/           # 标签控制器
│   ├── service/
│   │   ├── svckb/            # 知识库服务
│   │   ├── svcknowledge/     # 知识文档服务
│   │   ├── svcchunk/         # 切片服务
│   │   ├── svcfaq/           # FAQ 服务
│   │   ├── svcsession/       # 会话服务
│   │   ├── svcmessage/       # 消息服务
│   │   ├── svcqa/            # QA 对话服务（Knowledge Chat）
│   │   ├── svcmodel/         # 模型服务
│   │   ├── svcvectorstore/   # 向量存储服务
│   │   └── svctag/           # 标签服务
│   ├── dto/
│   │   ├── dtokb/            # 知识库 DTO
│   │   ├── dtoknowledge/     # 知识文档 DTO
│   │   ├── dtochunk/         # 切片 DTO
│   │   ├── dtofaq/           # FAQ DTO
│   │   ├── dtosession/       # 会话 DTO
│   │   ├── dtomessage/       # 消息 DTO
│   │   ├── dtomodel/         # 模型 DTO
│   │   ├── dtovectorstore/   # 向量存储 DTO
│   │   └── dtotag/           # 标签 DTO
│   ├── router/
│   │   ├── router.go         # RegisterRouter() 统一入口
│   │   ├── kb.go
│   │   ├── knowledge.go
│   │   ├── chunk.go
│   │   ├── faq.go
│   │   ├── session.go
│   │   ├── message.go
│   │   ├── model.go
│   │   ├── vectorstore.go
│   │   └── tag.go
│   ├── middleware/
│   │   ├── auth.go           # JWT 认证（复用 pkg/token）
│   │   └── rbac.go           # RBAC 权限检查
│   ├── engine/               # AI 引擎层（从 WeKnora 移植）
│   │   ├── llm/              # LLM 调用封装（OpenAI/Ollama 等）
│   │   ├── embedding/        # Embedding 调用封装
│   │   └── rerank/           # Rerank 调用封装
│   └── constant/             # 应用层常量
├── model/                    # GORM 数据模型
│   ├── model.go              # 表名常量
│   ├── knowledge_base.go
│   ├── knowledge.go
│   ├── chunk.go
│   ├── faq.go
│   ├── session.go
│   ├── message.go
│   ├── model_config.go
│   ├── vector_store.go
│   ├── tag.go
│   ├── tenant.go
│   └── user.go
├── dao/                      # 数据访问层（泛型 GenericDao）
│   ├── knowledge_base.go
│   ├── knowledge.go
│   ├── chunk.go
│   ├── faq.go
│   ├── session.go
│   ├── message.go
│   ├── model_config.go
│   ├── vector_store.go
│   ├── tag.go
│   └── chunk_search.go      # 向量检索的特殊查询
├── object/                   # 共享业务对象
│   └── objrag/               # RAG 模块间的共享字段
├── docs/                     # Swagger 文档
└── scripts/Dockerfile

共享包: pkg/code, pkg/dbclient, pkg/testsetup, pkg/token, pkg/gincontext
```

## 数据模型

### 表命名规范
- 前缀 `rg_` 标识 ragforge 应用的表
- 遵守 ark-go-starter 的下划线命名约定

### 核心表

**rg_tenant** - 多租户隔离
- id, name, status, storage_config(JSONB), created_at, updated_at, deleted_at

**rg_user** - 基础用户（后续对接 IAM）
- id, tenant_id(FK), username, email, password_hash, role, created_at, updated_at, deleted_at

**rg_knowledge_base** - 知识库
- id, tenant_id, name, description, type, parser_engine, embedding_config(JSONB), indexing_strategy(JSONB), creator_id, created_at, updated_at, deleted_at

**rg_knowledge** - 知识文档
- id, kb_id(FK), tenant_id, type(file/url/manual), title, content, file_url, source_url, parse_status, file_size, creator_id, created_at, updated_at, deleted_at

**rg_chunk** - 文档切片（使用 pgvector）
- id, knowledge_id(FK), kb_id, tenant_id, content, vector(vector(1536)), seq_id, tokens, meta_info(JSONB), created_at, updated_at, deleted_at

**rg_faq** - FAQ 条目
- id, kb_id(FK), tenant_id, question, answer, similar_questions(JSONB), tags(JSONB), status, creator_id, created_at, updated_at, deleted_at

**rg_session** - 对话会话
- id, tenant_id, user_id, kb_id, title, description, is_pinned, created_at, updated_at, deleted_at

**rg_message** - 对话消息
- id, session_id(FK), tenant_id, role(user/assistant/system), content, metadata(JSONB), token_count, created_at

**rg_model** - 模型配置
- id, tenant_id, name, type(chat/embedding/rerank/vlm/asr), provider(openai/deepseek/ollama/azure), model_name, config(JSONB), status, created_at, updated_at, deleted_at

**rg_vector_store** - 向量存储配置
- id, tenant_id, name, engine_type(pgvector/milvus/qdrant), config(JSONB), status, created_at, updated_at, deleted_at

**rg_tag** - 知识标签
- id, kb_id(FK), tenant_id, name, color, created_at, updated_at, deleted_at

## API 设计

遵循 ark-go-starter 的 4 层路径模式: `/v1/ragforge/{module}/{operation}`

### 知识库
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/kb/create | 创建知识库 |
| DELETE | /v1/ragforge/kb/delete | 删除知识库 |
| PUT | /v1/ragforge/kb/update | 更新知识库 |
| GET | /v1/ragforge/kb/detail | 知识库详情 |
| POST | /v1/ragforge/kb/pageList | 知识库分页列表 |
| POST | /v1/ragforge/kb/copy | 克隆知识库 |

### 知识文档
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/knowledge/createFile | 文件上传创建知识 |
| POST | /v1/ragforge/knowledge/createUrl | URL 创建知识 |
| POST | /v1/ragforge/knowledge/createManual | 手动录入知识 |
| DELETE | /v1/ragforge/knowledge/delete | 删除知识 |
| PUT | /v1/ragforge/knowledge/update | 更新知识 |
| GET | /v1/ragforge/knowledge/detail | 知识详情 |
| POST | /v1/ragforge/knowledge/pageList | 知识分页列表 |
| POST | /v1/ragforge/knowledge/reparse | 重解析文档 |
| GET | /v1/ragforge/knowledge/download | 下载文档 |

### Chunk
| Method | Path | 说明 |
|--------|------|------|
| GET | /v1/ragforge/chunk/pageList | 切片分页列表 |
| GET | /v1/ragforge/chunk/detail | 切片详情 |
| PUT | /v1/ragforge/chunk/update | 更新切片 |
| DELETE | /v1/ragforge/chunk/delete | 删除切片 |
| POST | /v1/ragforge/chunk/search | 语义搜索切片 |

### FAQ
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/faq/create | 创建 FAQ |
| PUT | /v1/ragforge/faq/update | 更新 FAQ |
| DELETE | /v1/ragforge/faq/delete | 删除 FAQ |
| GET | /v1/ragforge/faq/detail | FAQ 详情 |
| POST | /v1/ragforge/faq/pageList | FAQ 分页列表 |
| POST | /v1/ragforge/faq/search | FAQ 语义搜索 |
| POST | /v1/ragforge/faq/import | 导入 FAQ |

### 会话
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/session/create | 创建会话 |
| DELETE | /v1/ragforge/session/delete | 删除会话 |
| GET | /v1/ragforge/session/detail | 会话详情 |
| POST | /v1/ragforge/session/pageList | 会话分页列表 |
| PUT | /v1/ragforge/session/update | 更新会话 |
| POST | /v1/ragforge/session/generateTitle | 生成会话标题 |
| POST | /v1/ragforge/session/knowledgeChat | 知识问答（流式 SSE） |
| POST | /v1/ragforge/session/stop | 停止生成 |

### 消息
| Method | Path | 说明 |
|--------|------|------|
| GET | /v1/ragforge/message/list | 消息列表 |
| DELETE | /v1/ragforge/message/delete | 删除消息 |
| POST | /v1/ragforge/message/search | 搜索消息 |

### 模型
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/model/create | 创建模型 |
| PUT | /v1/ragforge/model/update | 更新模型 |
| DELETE | /v1/ragforge/model/delete | 删除模型 |
| GET | /v1/ragforge/model/detail | 模型详情 |
| POST | /v1/ragforge/model/pageList | 模型分页列表 |
| GET | /v1/ragforge/model/providers | 提供商类型列表 |
| POST | /v1/ragforge/model/test | 测试模型连接 |

### 向量存储
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/vectorStore/create | 创建向量存储 |
| PUT | /v1/ragforge/vectorStore/update | 更新向量存储 |
| DELETE | /v1/ragforge/vectorStore/delete | 删除向量存储 |
| GET | /v1/ragforge/vectorStore/detail | 向量存储详情 |
| POST | /v1/ragforge/vectorStore/pageList | 向量存储分页列表 |
| GET | /v1/ragforge/vectorStore/types | 向量库类型列表 |
| POST | /v1/ragforge/vectorStore/test | 测试连接 |

### 标签
| Method | Path | 说明 |
|--------|------|------|
| POST | /v1/ragforge/tag/create | 创建标签 |
| PUT | /v1/ragforge/tag/update | 更新标签 |
| DELETE | /v1/ragforge/tag/delete | 删除标签 |
| GET | /v1/ragforge/tag/list | 标签列表 |

## Engine 层

从 WeKnora 移植 LLM/Embedding/Rerank 调用封装，作为独立 engine 层供 Service 调用：

```
internal/engine/
├── llm/
│   ├── llm.go              # LLM 接口定义
│   ├── openai.go           # OpenAI 兼容接口实现
│   └── ollama.go           # Ollama 接口实现
├── embedding/
│   ├── embedding.go        # Embedding 接口定义
│   ├── openai.go
│   └── ollama.go
└── rerank/
    ├── rerank.go           # Rerank 接口定义
    ├── cohere.go
    └── bge.go
```

## 依赖关系

- `github.com/gin-gonic/gin` - Web 框架
- `gorm.io/gorm` + `gorm.io/driver/postgres` - ORM + PostgreSQL
- `github.com/morehao/golib` - 内部工具库
- `github.com/morehao/ark-go-starter/pkg` - 共享包（code, dbclient, gincontext 等）
- `github.com/sashabaranov/go-openai` - OpenAI API 客户端
- `github.com/pgvector/pgvector-go` - pgvector Go 绑定
- `github.com/stretchr/testify` - 测试

## 初始化流程

1. `config.InitConf()` - 加载配置
2. `glog.InitLogger()` - 初始化日志
3. `dbclient.InitMultiDB()` - 初始化 PostgreSQL（含 pgvector）
4. `InitEngine()` - 初始化 AI 引擎工厂
5. `gin.New()` + `ragforge.Routers(engine)` - 注册路由
6. `engine.Run()` - 启动服务

## 多租户隔离策略

- 所有数据表包含 `tenant_id` 字段
- DAO 层查询自动注入 `tenant_id` 过滤
- 中间件从 JWT Token 解析 `tenant_id` 注入 Context
- 未来对接 IAM 时只需替换 JWT 解析逻辑和用户同步逻辑
