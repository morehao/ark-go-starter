# RAGForge Implementation Plan - Phase 2: Models + DAOs

> Continuation of Phase 1. Requires Phase 1 to be complete.

---

### Task 2.1: Create model entities

**Files:**
- Create: `apps/ragforge/model/model.go`
- Create: `apps/ragforge/model/tenant.go`
- Create: `apps/ragforge/model/user.go`
- Create: `apps/ragforge/model/knowledge_base.go`
- Create: `apps/ragforge/model/knowledge.go`
- Create: `apps/ragforge/model/chunk.go`
- Create: `apps/ragforge/model/faq.go`
- Create: `apps/ragforge/model/session.go`
- Create: `apps/ragforge/model/message.go`
- Create: `apps/ragforge/model/model_config.go`
- Create: `apps/ragforge/model/vector_store.go`
- Create: `apps/ragforge/model/tag.go`

- [ ] **Step 1: Create model/model.go**

```go
package model

const (
	TableNameTenant        = "rg_tenant"
	TableNameUser          = "rg_user"
	TableNameKnowledgeBase = "rg_knowledge_base"
	TableNameKnowledge     = "rg_knowledge"
	TableNameChunk         = "rg_chunk"
	TableNameFAQ           = "rg_faq"
	TableNameSession       = "rg_session"
	TableNameMessage       = "rg_message"
	TableNameModel         = "rg_model"
	TableNameVectorStore   = "rg_vector_store"
	TableNameTag           = "rg_tag"
)
```

- [ ] **Step 2: Create model/tenant.go**

```go
package model

import "gorm.io/gorm"

type TenantEntity struct {
	gorm.Model
	Name          string `gorm:"column:name;type:varchar(255);not null"`
	Status        string `gorm:"column:status;type:varchar(50);default:active"`
	StorageConfig string `gorm:"column:storage_config;type:jsonb"`
}

func (TenantEntity) TableName() string { return TableNameTenant }
```

- [ ] **Step 3: Create model/user.go**

```go
package model

import "gorm.io/gorm"

type UserRole string

const (
	UserRoleViewer      UserRole = "viewer"
	UserRoleContributor UserRole = "contributor"
	UserRoleAdmin       UserRole = "admin"
	UserRoleOwner       UserRole = "owner"
)

type UserEntity struct {
	gorm.Model
	TenantID     uint     `gorm:"column:tenant_id;type:bigint;not null;index"`
	Username     string   `gorm:"column:username;type:varchar(255);not null"`
	Email        string   `gorm:"column:email;type:varchar(255)"`
	PasswordHash string   `gorm:"column:password_hash;type:varchar(255)"`
	Role         UserRole `gorm:"column:role;type:varchar(50);default:viewer"`
}

func (UserEntity) TableName() string { return TableNameUser }
```

- [ ] **Step 4: Create model/knowledge_base.go**

```go
package model

import "gorm.io/gorm"

type KBType string

const (
	KBTypeNormal KBType = "normal"
	KBTypeWiki   KBType = "wiki"
)

type KnowledgeBaseEntity struct {
	gorm.Model
	TenantID        uint   `gorm:"column:tenant_id;type:bigint;not null;index"`
	Name            string `gorm:"column:name;type:varchar(255);not null"`
	Description     string `gorm:"column:description;type:text"`
	KBType          KBType `gorm:"column:kb_type;type:varchar(50);default:normal"`
	ParserEngine    string `gorm:"column:parser_engine;type:varchar(100)"`
	EmbeddingConfig string `gorm:"column:embedding_config;type:jsonb"`
	IndexStrategy   string `gorm:"column:index_strategy;type:jsonb"`
	CreatorID       uint   `gorm:"column:creator_id;type:bigint"`
}

func (KnowledgeBaseEntity) TableName() string { return TableNameKnowledgeBase }
```

- [ ] **Step 5: Create model/knowledge.go**

```go
package model

import "gorm.io/gorm"

type KnowledgeType string

const (
	KnowledgeTypeFile   KnowledgeType = "file"
	KnowledgeTypeURL    KnowledgeType = "url"
	KnowledgeTypeManual KnowledgeType = "manual"
)

type ParseStatus string

const (
	ParseStatusPending ParseStatus = "pending"
	ParseStatusParsing ParseStatus = "parsing"
	ParseStatusSuccess ParseStatus = "success"
	ParseStatusFailed  ParseStatus = "failed"
)

type KnowledgeEntity struct {
	gorm.Model
	KbID        uint          `gorm:"column:kb_id;type:bigint;not null;index"`
	TenantID    uint          `gorm:"column:tenant_id;type:bigint;not null;index"`
	Type        KnowledgeType `gorm:"column:type;type:varchar(50);not null"`
	Title       string        `gorm:"column:title;type:varchar(500)"`
	Content     string        `gorm:"column:content;type:text"`
	FileURL     string        `gorm:"column:file_url;type:text"`
	SourceURL   string        `gorm:"column:source_url;type:text"`
	ParseStatus ParseStatus   `gorm:"column:parse_status;type:varchar(50);default:pending"`
	FileSize    int64         `gorm:"column:file_size;type:bigint"`
	CreatorID   uint          `gorm:"column:creator_id;type:bigint"`
}

func (KnowledgeEntity) TableName() string { return TableNameKnowledge }

type KnowledgeEntityList []KnowledgeEntity

func (l KnowledgeEntityList) ToMap() map[uint]KnowledgeEntity {
	m := make(map[uint]KnowledgeEntity, len(l))
	for _, item := range l {
		m[item.ID] = item
	}
	return m
}
```

- [ ] **Step 6: Create model/chunk.go**

```go
package model

import "gorm.io/gorm"

type ChunkEntity struct {
	gorm.Model
	KnowledgeID uint   `gorm:"column:knowledge_id;type:bigint;not null;index"`
	KbID        uint   `gorm:"column:kb_id;type:bigint;not null;index"`
	TenantID    uint   `gorm:"column:tenant_id;type:bigint;not null;index"`
	Content     string `gorm:"column:content;type:text;not null"`
	SeqID       int    `gorm:"column:seq_id;type:int"`
	Tokens      int    `gorm:"column:tokens;type:int"`
	MetaInfo    string `gorm:"column:meta_info;type:jsonb"`
}

func (ChunkEntity) TableName() string { return TableNameChunk }

type ChunkEntityList []ChunkEntity

func (l ChunkEntityList) ToMap() map[uint]ChunkEntity {
	m := make(map[uint]ChunkEntity, len(l))
	for _, item := range l {
		m[item.ID] = item
	}
	return m
}
```

- [ ] **Step 7: Create model/faq.go**

```go
package model

import "gorm.io/gorm"

type FAQStatus string

const (
	FAQStatusActive   FAQStatus = "active"
	FAQStatusInactive FAQStatus = "inactive"
)

type FAQEntity struct {
	gorm.Model
	KbID             uint      `gorm:"column:kb_id;type:bigint;not null;index"`
	TenantID         uint      `gorm:"column:tenant_id;type:bigint;not null;index"`
	Question         string    `gorm:"column:question;type:text;not null"`
	Answer           string    `gorm:"column:answer;type:text;not null"`
	SimilarQuestions string    `gorm:"column:similar_questions;type:jsonb"`
	Tags             string    `gorm:"column:tags;type:jsonb"`
	Status           FAQStatus `gorm:"column:status;type:varchar(50);default:active"`
	CreatorID        uint      `gorm:"column:creator_id;type:bigint"`
}

func (FAQEntity) TableName() string { return TableNameFAQ }

type FAQEntityList []FAQEntity
```

- [ ] **Step 8: Create model/session.go**

```go
package model

import "gorm.io/gorm"

type SessionEntity struct {
	gorm.Model
	TenantID    uint   `gorm:"column:tenant_id;type:bigint;not null;index"`
	UserID      uint   `gorm:"column:user_id;type:bigint;not null;index"`
	KbID        uint   `gorm:"column:kb_id;type:bigint"`
	Title       string `gorm:"column:title;type:varchar(500)"`
	Description string `gorm:"column:description;type:text"`
	IsPinned    bool   `gorm:"column:is_pinned;type:boolean;default:false"`
}

func (SessionEntity) TableName() string { return TableNameSession }

type SessionEntityList []SessionEntity
```

- [ ] **Step 9: Create model/message.go**

```go
package model

import "gorm.io/gorm"

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

type MessageEntity struct {
	gorm.Model
	SessionID  uint        `gorm:"column:session_id;type:bigint;not null;index"`
	TenantID   uint        `gorm:"column:tenant_id;type:bigint;not null;index"`
	Role       MessageRole `gorm:"column:role;type:varchar(50);not null"`
	Content    string      `gorm:"column:content;type:text;not null"`
	Metadata   string      `gorm:"column:metadata;type:jsonb"`
	TokenCount int         `gorm:"column:token_count;type:int"`
}

func (MessageEntity) TableName() string { return TableNameMessage }

type MessageEntityList []MessageEntity
```

- [ ] **Step 10: Create model/model_config.go**

```go
package model

import "gorm.io/gorm"

type ModelType string

const (
	ModelTypeChat      ModelType = "chat"
	ModelTypeEmbedding ModelType = "embedding"
	ModelTypeRerank    ModelType = "rerank"
	ModelTypeVLM       ModelType = "vlm"
	ModelTypeASR       ModelType = "asr"
)

type ModelEntity struct {
	gorm.Model
	TenantID  uint      `gorm:"column:tenant_id;type:bigint;not null;index"`
	Name      string    `gorm:"column:name;type:varchar(255);not null"`
	ModelType ModelType `gorm:"column:model_type;type:varchar(50);not null"`
	Provider  string    `gorm:"column:provider;type:varchar(100);not null"`
	ModelName string    `gorm:"column:model_name;type:varchar(255);not null"`
	Config    string    `gorm:"column:config;type:jsonb"`
	Status    string    `gorm:"column:status;type:varchar(50);default:active"`
}

func (ModelEntity) TableName() string { return TableNameModel }

type ModelEntityList []ModelEntity
```

- [ ] **Step 11: Create model/vector_store.go**

```go
package model

import "gorm.io/gorm"

type VectorStoreEntity struct {
	gorm.Model
	TenantID   uint   `gorm:"column:tenant_id;type:bigint;not null;index"`
	Name       string `gorm:"column:name;type:varchar(255);not null"`
	EngineType string `gorm:"column:engine_type;type:varchar(100);not null"`
	Config     string `gorm:"column:config;type:jsonb"`
	Status     string `gorm:"column:status;type:varchar(50);default:active"`
}

func (VectorStoreEntity) TableName() string { return TableNameVectorStore }

type VectorStoreEntityList []VectorStoreEntity
```

- [ ] **Step 12: Create model/tag.go**

```go
package model

import "gorm.io/gorm"

type TagEntity struct {
	gorm.Model
	KbID     uint   `gorm:"column:kb_id;type:bigint;not null;index"`
	TenantID uint   `gorm:"column:tenant_id;type:bigint;not null;index"`
	Name     string `gorm:"column:name;type:varchar(255);not null"`
	Color    string `gorm:"column:color;type:varchar(50)"`
}

func (TagEntity) TableName() string { return TableNameTag }

type TagEntityList []TagEntity
```

- [ ] **Step 13: Create object/objrag/rag.go**

```go
package objrag

type OperatorBaseInfo struct {
	CreatedBy uint `json:"createdBy"`
	UpdatedBy uint `json:"updatedBy"`
}

type TenantBase struct {
	TenantID uint `json:"tenantID"`
}
```

- [ ] **Step 14: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```

- [ ] **Step 15: Commit**

```bash
git add apps/ragforge/model/ apps/ragforge/object/
git commit -m "feat(ragforge): add model entities and shared objects"
```

---

### Task 2.2: Create DAO layer

**Files:**
- Create: `apps/ragforge/dao/knowledge_base.go`
- Create: `apps/ragforge/dao/knowledge.go`
- Create: `apps/ragforge/dao/chunk.go`
- Create: `apps/ragforge/dao/faq.go`
- Create: `apps/ragforge/dao/session.go`
- Create: `apps/ragforge/dao/message.go`
- Create: `apps/ragforge/dao/model_config.go`
- Create: `apps/ragforge/dao/vector_store.go`
- Create: `apps/ragforge/dao/tag.go`

- [ ] **Step 1: Create dao/knowledge_base.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type KnowledgeBaseCond struct {
	*genericdao.BaseCond
	TenantID uint
	Name     string
}

func (c *KnowledgeBaseCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName+".name like ?", "%"+c.Name+"%")
	}
}

type KnowledgeBaseDao struct {
	*genericdao.GenericDao[model.KnowledgeBaseEntity, model.KnowledgeBaseEntity]
}

func NewKnowledgeBaseDao() *KnowledgeBaseDao {
	return &KnowledgeBaseDao{
		GenericDao: genericdao.NewGenericDao[model.KnowledgeBaseEntity, model.KnowledgeBaseEntity](
			model.TableNameKnowledgeBase, "KnowledgeBaseDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 2: Create dao/knowledge.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type KnowledgeCond struct {
	*genericdao.BaseCond
	KbID        uint
	TenantID    uint
	KnowledgeType string
	ParseStatus   string
}

func (c *KnowledgeCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.KbID > 0 {
		db.Where(tableName + ".kb_id = ?", c.KbID)
	}
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.KnowledgeType != "" {
		db.Where(tableName + ".type = ?", c.KnowledgeType)
	}
	if c.ParseStatus != "" {
		db.Where(tableName + ".parse_status = ?", c.ParseStatus)
	}
}

type KnowledgeDao struct {
	*genericdao.GenericDao[model.KnowledgeEntity, model.KnowledgeEntityList]
}

func NewKnowledgeDao() *KnowledgeDao {
	return &KnowledgeDao{
		GenericDao: genericdao.NewGenericDao[model.KnowledgeEntity, model.KnowledgeEntityList](
			model.TableNameKnowledge, "KnowledgeDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 3: Create dao/chunk.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type ChunkCond struct {
	*genericdao.BaseCond
	KnowledgeID uint
	KbID        uint
	TenantID    uint
	Content     string
}

func (c *ChunkCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.KnowledgeID > 0 {
		db.Where(tableName + ".knowledge_id = ?", c.KnowledgeID)
	}
	if c.KbID > 0 {
		db.Where(tableName + ".kb_id = ?", c.KbID)
	}
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
}

type ChunkDao struct {
	*genericdao.GenericDao[model.ChunkEntity, model.ChunkEntityList]
}

func NewChunkDao() *ChunkDao {
	return &ChunkDao{
		GenericDao: genericdao.NewGenericDao[model.ChunkEntity, model.ChunkEntityList](
			model.TableNameChunk, "ChunkDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 4: Create dao/faq.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type FAQCond struct {
	*genericdao.BaseCond
	KbID     uint
	TenantID uint
	Question string
	Status   string
}

func (c *FAQCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.KbID > 0 {
		db.Where(tableName + ".kb_id = ?", c.KbID)
	}
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.Question != "" {
		db.Where(tableName+".question like ?", "%"+c.Question+"%")
	}
	if c.Status != "" {
		db.Where(tableName + ".status = ?", c.Status)
	}
}

type FAQDao struct {
	*genericdao.GenericDao[model.FAQEntity, model.FAQEntityList]
}

func NewFAQDao() *FAQDao {
	return &FAQDao{
		GenericDao: genericdao.NewGenericDao[model.FAQEntity, model.FAQEntityList](
			model.TableNameFAQ, "FAQDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 5: Create dao/session.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type SessionCond struct {
	*genericdao.BaseCond
	TenantID uint
	UserID   uint
	KbID     uint
}

func (c *SessionCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID > 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
	if c.KbID > 0 {
		db.Where(tableName + ".kb_id = ?", c.KbID)
	}
}

type SessionDao struct {
	*genericdao.GenericDao[model.SessionEntity, model.SessionEntityList]
}

func NewSessionDao() *SessionDao {
	return &SessionDao{
		GenericDao: genericdao.NewGenericDao[model.SessionEntity, model.SessionEntityList](
			model.TableNameSession, "SessionDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 6: Create dao/message.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type MessageCond struct {
	*genericdao.BaseCond
	SessionID uint
	TenantID  uint
	Role      string
}

func (c *MessageCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.SessionID > 0 {
		db.Where(tableName + ".session_id = ?", c.SessionID)
	}
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
}

type MessageDao struct {
	*genericdao.GenericDao[model.MessageEntity, model.MessageEntityList]
}

func NewMessageDao() *MessageDao {
	return &MessageDao{
		GenericDao: genericdao.NewGenericDao[model.MessageEntity, model.MessageEntityList](
			model.TableNameMessage, "MessageDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 7: Create dao/model_config.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type ModelCond struct {
	*genericdao.BaseCond
	TenantID  uint
	ModelType string
	Provider  string
	Status    string
}

func (c *ModelCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.ModelType != "" {
		db.Where(tableName + ".model_type = ?", c.ModelType)
	}
	if c.Provider != "" {
		db.Where(tableName + ".provider = ?", c.Provider)
	}
}

type ModelDao struct {
	*genericdao.GenericDao[model.ModelEntity, model.ModelEntityList]
}

func NewModelDao() *ModelDao {
	return &ModelDao{
		GenericDao: genericdao.NewGenericDao[model.ModelEntity, model.ModelEntityList](
			model.TableNameModel, "ModelDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 8: Create dao/vector_store.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type VectorStoreCond struct {
	*genericdao.BaseCond
	TenantID   uint
	EngineType string
	Status     string
}

func (c *VectorStoreCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.EngineType != "" {
		db.Where(tableName + ".engine_type = ?", c.EngineType)
	}
}

type VectorStoreDao struct {
	*genericdao.GenericDao[model.VectorStoreEntity, model.VectorStoreEntityList]
}

func NewVectorStoreDao() *VectorStoreDao {
	return &VectorStoreDao{
		GenericDao: genericdao.NewGenericDao[model.VectorStoreEntity, model.VectorStoreEntityList](
			model.TableNameVectorStore, "VectorStoreDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 9: Create dao/tag.go**

```go
package dao

import (
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"gorm.io/gorm"
)

type TagCond struct {
	*genericdao.BaseCond
	KbID     uint
	TenantID uint
}

func (c *TagCond) BuildCondition(db *gorm.DB, tableName string) {
	c.BaseCond.BuildCondition(db, tableName)
	if c.KbID > 0 {
		db.Where(tableName + ".kb_id = ?", c.KbID)
	}
	if c.TenantID > 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
}

type TagDao struct {
	*genericdao.GenericDao[model.TagEntity, model.TagEntityList]
}

func NewTagDao() *TagDao {
	return &TagDao{
		GenericDao: genericdao.NewGenericDao[model.TagEntity, model.TagEntityList](
			model.TableNameTag, "TagDao",
			dbclient.DefaultDB,
		),
	}
}
```

- [ ] **Step 10: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```
Expected: Build succeeds.

- [ ] **Step 11: Commit**

```bash
git add apps/ragforge/dao/
git commit -m "feat(ragforge): add DAO layer with GenericDao wrappers"
```
