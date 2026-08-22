# Benzhi - 文章自动摘要与关键词提取系统

## 项目简介

Benzhi 是一个基于 Go 1.22 标准库实现的文章自动摘要与关键词提取 HTTP 服务。
系统提供单篇文章同步分析、多篇文章批量异步任务处理、历史结果查询、存储快照导出以及运行时指标采集等能力。

核心特性：

- 纯标准库实现，无三方依赖，易于构建与部署。
- 支持单篇文章同步提交并即时返回摘要 + 关键词。
- 支持多篇文章批量提交，通过内存任务队列 + 多 worker 异步消费。
- 提供文章、结果、任务三类资源的查询接口（分页）。
- 内置健康检查 / 就绪检查、指标快照、存储快照导出端点。
- 中间件链覆盖：CORS、Panic 恢复、请求日志、统一响应格式。

---

## 目录结构说明

```
.
├── cmd/
│   └── server/
│       └── main.go                  # 服务入口：加载配置、装配依赖、启动 HTTP 与 worker 池
├── internal/
│   ├── cache/                       # 通用缓存组件（LRU、分片、结果缓存）
│   │   ├── cache.go
│   │   ├── lru.go
│   │   ├── result_cache.go
│   │   └── sharded.go
│   ├── config/                      # 应用配置：结构、默认值、环境变量加载、校验
│   │   ├── config.go
│   │   ├── limits.go
│   │   ├── loader.go
│   │   └── schema.go
│   ├── export/                      # 导出格式化器（CSV / JSON / 纯文本）
│   │   ├── csv.go
│   │   ├── exporter.go
│   │   ├── json.go
│   │   └── text.go
│   ├── handler/                     # HTTP 处理器：路由装配 + 各端点实现 + 中间件
│   │   ├── article_handler.go       # 文章相关接口（Submit / Get / List / GetResult / ListResults）
│   │   ├── task_handler.go          # 批量任务相关接口（SubmitBatch / GetTask / ListTasks）
│   │   ├── health_handler.go        # /health 与 /ready
│   │   ├── metrics_handler.go       # /api/v1/metrics
│   │   ├── export_handler.go        # /api/v1/export
│   │   ├── static_handler.go        # 前端静态资源服务
│   │   ├── router.go                # 路由组装入口 Router()
│   │   ├── middleware.go            # CORS / Recovery / Logging 中间件
│   │   ├── error.go                 # 错误码映射
│   │   ├── status.go                # 状态记录器
│   │   └── validator.go             # 请求参数辅助工具
│   ├── metrics/                     # 运行时指标（计数器、量规、快照）
│   │   ├── counter.go
│   │   ├── gauge.go
│   │   └── metrics.go
│   ├── model/                       # 领域模型、DTO 与错误定义
│   │   ├── article.go
│   │   ├── dto.go
│   │   ├── errors.go
│   │   ├── keyword.go
│   │   ├── meta.go
│   │   ├── pagination.go
│   │   ├── request.go
│   │   ├── result.go
│   │   ├── sentence.go
│   │   ├── status.go
│   │   ├── task.go
│   │   └── validation.go
│   ├── pipeline/                    # 文本处理流水线（阶段 + 结果）
│   │   ├── pipeline.go
│   │   ├── result.go
│   │   └── stage.go
│   ├── ratelimit/                   # 令牌桶限速器及 HTTP 中间件
│   │   ├── limiter.go
│   │   ├── middleware.go
│   │   └── token_bucket.go
│   ├── search/                      # 搜索索引、查询、排序、分词
│   │   ├── index.go
│   │   ├── query.go
│   │   ├── ranker.go
│   │   └── tokenizer.go
│   ├── service/                     # 业务服务层
│   │   ├── analyzer.go              # 文本分析器（组合预处理 + TF-IDF + TextRank + 摘要）
│   │   ├── article_service.go       # 单篇文章提交 / 查询服务
│   │   ├── batch_service.go         # 批量任务处理
│   │   ├── dedupe.go                # 去重逻辑
│   │   ├── export_service.go        # 导出服务
│   │   ├── history_service.go       # 历史查询
│   │   ├── keywords.go              # 关键词提取辅助
│   │   ├── options.go               # 可选项模式
│   │   ├── pipeline_builder.go      # 流水线构建器
│   │   ├── preprocess_service.go    # 文本预处理
│   │   ├── rank.go                  # 排序逻辑
│   │   ├── report_service.go        # 报告生成
│   │   ├── similarity.go            # 相似度计算
│   │   ├── statistics.go            # 统计服务
│   │   ├── summarize_service.go     # 摘要生成
│   │   ├── task_query.go            # 任务查询辅助
│   │   ├── task_service.go          # 任务提交 / 查询服务
│   │   ├── textrank_service.go      # TextRank 算法实现
│   │   ├── textstat.go              # 文本统计
│   │   └── tfidf_service.go         # TF-IDF 算法实现
│   ├── store/                       # 内存存储层
│   │   ├── store.go                 # 通用存储接口
│   │   ├── memory_store.go          # 聚合内存实现（文章 / 结果 / 任务）
│   │   ├── article_store.go         # 文章存取接口与实现
│   │   ├── result_store.go          # 结果存取接口与实现
│   │   ├── task_store.go            # 任务存取接口与实现
│   │   ├── counter.go               # ID 生成器
│   │   ├── export.go                # 快照导出能力
│   │   ├── iterator.go              # 分页迭代器
│   │   └── query.go                 # 查询与过滤辅助
│   ├── taskqueue/                   # 内存任务队列与 worker 管理器
│   │   ├── queue.go
│   │   └── manager.go
│   └── textutil/                    # 文本工具：分词、N-gram、停用词、相似度、编码等
│       ├── charset.go
│       ├── file.go
│       ├── ngram.go
│       ├── ranking.go
│       ├── segment.go
│       ├── similarity.go
│       ├── stopwords.go
│       ├── stopwords_cn.go
│       ├── stopwords_en.go
│       ├── stopwords_reader.go
│       ├── string.go
│       ├── tokenizer.go
│       ├── tokenizer_ext.go
│       └── unicode.go
├── pkg/                             # 可复用公共包
│   ├── httputil/                    # HTTP 工具（请求头解码、辅助、响应状态记录）
│   ├── logger/                      # 结构化日志（字段包装、写入器）
│   └── response/                    # 统一 JSON 响应封装
├── web/
│   └── static/                      # 前端静态资源（index.html / app.js / style.css）
├── go.mod
├── benzhi.Dockerfile                # Docker 镜像构建文件
└── build_benzhi_docker.sh           # Docker 镜像构建脚本
```

---

## API 文档

所有接口统一返回 JSON 结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": { }
}
```

其中：
- `code=0` 表示成功，非 0 表示失败且值等于 HTTP 状态码。
- `message` 为描述信息，失败时为错误原因。
- `data` 为业务载荷，可选。

分页查询均支持查询参数 `offset` 与 `limit`（默认 `offset=0`，`limit=50`）。

### 1. 文章相关接口

#### 1.1 提交单篇文章（同步分析）

- **方法**：`POST`
- **路径**：`/api/v1/articles`
- **请求体**：

```json
{
  "title": "文章标题",
  "content": "文章正文内容（最大长度由 SERVER_MAX_ARTICLE_LENGTH 控制）"
}
```

- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "article_id": "字符串ID",
    "title": "文章标题",
    "summary": "生成的摘要",
    "keywords": [
      { "word": "关键词", "score": 0.95, "tf": 10, "idf": 3.2 }
    ],
    "duration_ms": 42
  }
}
```

- **失败响应**：
  - 400：请求体非法 / 内容为空 / 内容过长 / 参数非法
  - 429：同一内容失败次数过多
  - 500：处理过程发生内部错误

#### 1.2 查询文章详情

- **方法**：`GET`
- **路径**：`/api/v1/articles/{id}`
- **路径参数**：`id` - 文章 ID
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "ID",
    "title": "标题",
    "content": "正文",
    "status": "pending | ready | failed",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

- **失败响应**：404 文章不存在

#### 1.3 分页查询文章列表

- **方法**：`GET`
- **路径**：`/api/v1/articles`
- **查询参数**：`offset`，`limit`
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ { "id": "...", "title": "...", "content": "...", "status": "ready", "created_at": "...", "updated_at": "..." } ],
    "total": 123,
    "offset": 0,
    "limit": 50
  }
}
```

#### 1.4 获取单篇文章的分析结果

- **方法**：`GET`
- **路径**：`/api/v1/articles/{id}/result`
- **路径参数**：`id` - 文章 ID
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "article_id": "ID",
    "summary": "摘要",
    "keywords": [ { "word": "关键词", "score": 0.9, "tf": 8, "idf": 2.7 } ],
    "sentence_count": 15,
    "duration_ms": 31,
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

#### 1.5 分页查询所有分析结果

- **方法**：`GET`
- **路径**：`/api/v1/results`
- **查询参数**：`offset`，`limit`
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ { "article_id": "...", "summary": "...", "keywords": [], "sentence_count": 10, "duration_ms": 10, "created_at": "..." } ],
    "total": 100,
    "offset": 0,
    "limit": 50
  }
}
```

---

### 2. 批量任务相关接口

#### 2.1 提交批量分析任务

- **方法**：`POST`
- **路径**：`/api/v1/batch`
- **请求体**：

```json
{
  "articles": [
    { "title": "标题1", "content": "正文1" },
    { "title": "标题2", "content": "正文2" }
  ]
}
```

- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "任务ID",
    "type": "batch",
    "status": "pending",
    "article_ids": ["ID1", "ID2"],
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

#### 2.2 查询任务状态

- **方法**：`GET`
- **路径**：`/api/v1/tasks/{id}`
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "任务ID",
    "type": "single | batch",
    "status": "pending | running | success | failed",
    "article_ids": ["ID1", "ID2"],
    "error": "仅失败时返回",
    "created_at": "...",
    "updated_at": "..."
  }
}
```

#### 2.3 分页查询任务列表

- **方法**：`GET`
- **路径**：`/api/v1/tasks`
- **查询参数**：`offset`，`limit`
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [ { "id": "...", "type": "batch", "status": "success", "article_ids": [], "created_at": "...", "updated_at": "..." } ],
    "total": 50,
    "offset": 0,
    "limit": 50
  }
}
```

---

### 3. 健康检查接口

#### 3.1 存活检查

- **方法**：`GET`
- **路径**：`/health`
- **响应**：200

```json
{
  "code": 0,
  "message": "ok",
  "data": { "status": "up" }
}
```

#### 3.2 就绪检查

- **方法**：`GET`
- **路径**：`/ready`
- **响应**：服务就绪时 200，否则 503

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "status": "ready",
    "stats": { "articles": 10, "results": 10, "tasks": 2 }
  }
}
```

---

### 4. 观测与导出接口

#### 4.1 运行时指标快照

- **方法**：`GET`
- **路径**：`/api/v1/metrics`
- **成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "requests_total": 1000,
    "articles_total": 123,
    "tasks_total": 10,
    "tasks_pending": 1,
    "tasks_running": 2,
    "tasks_success": 7
  }
}
```

#### 4.2 存储快照导出

- **方法**：`GET`
- **路径**：`/api/v1/export`
- **成功响应（200）**：返回内存中全部文章、结果、任务的完整快照。

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "articles": [ { "id": "...", "title": "...", "content": "...", "status": "...", "created_at": "...", "updated_at": "..." } ],
    "results": [ { "article_id": "...", "summary": "...", "keywords": [], "sentence_count": 10, "duration_ms": 0, "created_at": "..." } ],
    "tasks": [ { "id": "...", "type": "...", "status": "...", "article_ids": [], "created_at": "...", "updated_at": "..." } ]
  }
}
```

---

### 5. 前端静态资源

- **方法**：`GET`
- **路径**：`/` 以及其下所有静态路径
- **说明**：映射 `./web/static` 目录内容（内置极简 Web UI）。

---

## 本地运行步骤

### 前置条件

- Go 1.22 及以上
- 任意可用的 8080 端口（可通过环境变量覆盖）

### 运行命令

```bash
# 1. 进入项目根目录
cd /path/to/project

# 2. 可选：设置环境变量（使用默认值可跳过）
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_LOG_LEVEL=info
export SERVER_LOG_FORMAT=json

# 3. 直接启动服务
go run ./cmd/server
```

启动成功后：

- HTTP 服务监听在 `0.0.0.0:8080`（默认）
- 访问 `http://localhost:8080/health` 验证存活
- 访问 `http://localhost:8080/ready` 验证就绪
- 访问 `http://localhost:8080/` 打开内置 Web UI

### 常用环境变量（前缀 `SERVER_`）

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `SERVER_HOST` | `0.0.0.0` | 监听主机 |
| `SERVER_PORT` | `8080` | 监听端口 |
| `SERVER_MAX_KEYWORD_COUNT` | `10` | 关键词提取数量上限 |
| `SERVER_MAX_SUMMARY_SENTENCES` | `5` | 摘要句子数量上限 |
| `SERVER_MAX_ARTICLE_LENGTH` | `100000` | 单篇文章最大字符数 |
| `SERVER_MIN_SENTENCE_LENGTH` | `3` | 参与打分的最短句子 token 数 |
| `SERVER_TEXTRANK_MAX_ITER` | `30` | TextRank 最大迭代次数 |
| `SERVER_TEXTRANK_DAMPING` | `0.85` | TextRank 阻尼系数 |
| `SERVER_WORKER_COUNT` | `4` | 异步 worker 数量 |
| `SERVER_QUEUE_CAPACITY` | `256` | 任务队列容量 |
| `SERVER_READ_TIMEOUT` | `10s` | HTTP 读超时 |
| `SERVER_WRITE_TIMEOUT` | `30s` | HTTP 写超时 |
| `SERVER_IDLE_TIMEOUT` | `60s` | HTTP 空闲连接超时 |
| `SERVER_SHUTDOWN_TIMEOUT` | `15s` | 优雅关闭超时 |
| `SERVER_LOG_LEVEL` | `info` | 日志等级：debug/info/warn/error |
| `SERVER_LOG_FORMAT` | `json` | 日志格式：json/text |

---

## Docker 构建和运行步骤

### Dockerfile

项目根目录提供 `benzhi.Dockerfile`，基于 `golang:1.22`，禁用 CGO，使用 `go run ./cmd/server` 启动。

```dockerfile
FROM golang:1.22
ENV CGO_ENABLED=0
WORKDIR /app
COPY . .
RUN go build ./...
CMD ["go", "run", "./cmd/server"]
```

### 方式一：直接使用 docker build

```bash
# 构建镜像
docker build -f benzhi.Dockerfile -t benzhi-go-app:latest .

# 运行容器（映射 8080 端口）
docker run --rm -d --name benzhi -p 8080:8080 benzhi-go-app:latest

# 验证
curl http://localhost:8080/health

# 查看日志
docker logs -f benzhi

# 停止容器
docker stop benzhi
```

### 方式二：使用构建脚本（支持跨平台）

项目提供 `build_benzhi_docker.sh`，可指定镜像名、标签、目标平台、上下文路径。

```bash
chmod +x build_benzhi_docker.sh

# 构建默认 linux/amd64 镜像
./build_benzhi_docker.sh benzhi-go-app latest linux/amd64 .

# 构建 linux/arm64 镜像（Apple Silicon / ARM 服务器）
./build_benzhi_docker.sh benzhi-go-app latest linux/arm64 .

# 运行
docker run --rm -d -p 8080:8080 \
  -e SERVER_PORT=8080 \
  -e SERVER_LOG_LEVEL=info \
  benzhi-go-app:latest
```

### 进入容器

```bash
docker run --rm -it benzhi-go-app:latest /bin/bash
```

---

## 测试命令示例

### 1. 运行全部单测

```bash
go test ./...
```

### 2. 运行并打开竞态检测（-race），可发现数据竞争

```bash
go test -race ./...
```

### 3. 反复执行 N 次，捕获偶发失败 / 竞态

```bash
# 连续跑 20 次全部用例，用于稳定性回归
go test -count=20 ./...

# 连续跑 20 次并启用 race
go test -race -count=20 ./...
```

### 4. 运行特定缺陷验证用例（本项目内置 concurrency 缺陷测试）

```bash
# 资源泄漏 + 死锁 + 数据竞争综合验证
go test -race -count=1 ./internal/service -run '^TestBugSumm27_DeferResourceLeakAndRace$'

# Handler 层并发 map 竞态验证
go test -race -count=1 ./internal/handler -run '^TestBugSumm27_HandlerConcurrentMapRace$'
```

### 5. 指定包多次重复运行，用于高置信度的稳定性验收

```bash
# 对 service / handler 两个并发热点包各跑 10 次 race
go test -race -count=10 ./internal/service ./internal/handler
```

### 6. 静态检查

```bash
go vet ./...
```

### 7. 手动冒烟测试（HTTP）

启动服务后，使用 curl 验证主要端点：

```bash
# 提交单篇文章
curl -s -X POST http://localhost:8080/api/v1/articles \
  -H 'Content-Type: application/json' \
  -d '{"title":"测试标题","content":"这是一段用于测试的文章内容。包含多个句子。关键词提取会计算每个词的重要性。"}'

# 查询文章列表
curl -s http://localhost:8080/api/v1/articles?limit=10

# 健康检查
curl -s http://localhost:8080/health

# 就绪检查
curl -s http://localhost:8080/ready
```

---

## 统一响应与错误码说明

- 所有接口采用 `{code, message, data}` 结构。
- `code=0` 对应 HTTP 200 表示成功。
- 失败时 `code` 等于 HTTP 状态码，常见值：
  - `400` 请求体或参数非法
  - `404` 资源不存在
  - `429` 请求失败次数过多 / 触发熔断
  - `500` 服务内部错误
  - `503` 服务尚未就绪
