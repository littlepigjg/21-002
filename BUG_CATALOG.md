# BUG_CATALOG.md — 缺陷候选清单

本文档为「文章自动摘要与关键词提取系统」的缺陷候选清单，共 **30 个后端（Go）缺陷**。

## 使用说明

1. 缺陷按 `{项目缩写}-{类别}-{序号}` 命名，项目缩写为 `summ`。
2. 每个缺陷均跨至少 **2 个文件**，且**单缺陷注入时需修改 ≥ 60 行代码（不含注释与空行）**，禁止仅在单个文件内改几行。
3. 缺陷均为运行时缺陷：注入后 `go build ./...` 仍可通过、服务仍可启动，但按「触发方式」操作即可稳定复现错误现象。
4. 缺陷相互独立：注入某个缺陷不影响其它缺陷的触发条件。
5. 单文件缺陷占比满足 ≤ 30%（缺陷分布到 60+ 个文件，无单文件堆积）。

## 缺陷清单

| bug_id | bug_category | 缺陷描述 | 植入位置 | 预期表现 | 触发方式 | 缺陷难度 |
|--------|--------------|----------|----------|----------|----------|----------|
| summ-concur-001 | concurrency | 任务对象指针被提交与处理协程并发读写导致数据竞争 | internal/store/task_store.go.GetTask、internal/service/batch_service.go.processBatch、internal/service/task_service.go.SubmitBatch | 高并发轮询任务状态时报 data race，状态字段读写出错 | 提交批量任务后立即并发多次 GET /api/v1/tasks/{id} | 4 星 |
| summ-concur-002 | concurrency | worker 数为 0 时 WaitGroup 计数错配导致死锁 | internal/taskqueue/manager.go.Start、internal/taskqueue/manager.go.worker、cmd/server/main.go | 批量任务永远 pending，优雅关闭时 Wait 永久阻塞 | 将 SERVER_WORKER_COUNT 配为 0 后提交批量任务 | 3 星 |
| summ-concur-003 | concurrency | 队列关闭后仍向 channel 发送导致 panic | internal/taskqueue/queue.go.Enqueue、internal/taskqueue/queue.go.Close、internal/service/task_service.go.SubmitBatch | 关闭期间提交任务触发 send on closed channel panic | 服务关闭同时提交批量任务 | 5 星 |
| summ-concur-004 | concurrency | ID 生成改为非原子自增导致并发 ID 冲突 | internal/store/counter.go.Next、internal/service/article_service.go.Submit、internal/service/task_service.go.SubmitBatch | 并发提交出现重复 article_id，后写覆盖先写 | 并发 POST /api/v1/articles 提交多篇文章 | 4 星 |
| summ-concur-005 | concurrency | 结果 map 写入与快照遍历的锁保护范围不一致 | internal/store/result_store.go.SaveResult、internal/store/export.go.Snapshot、internal/store/memory_store.go.NewMemoryStore | 查询快照时并发写结果触发 concurrent map 读写 panic | 分析完成后同时访问 GET /api/v1/export | 4 星 |
| summ-concur-006 | concurrency | worker 忽略 ctx 且 channel 未关闭导致 goroutine 泄露 | internal/taskqueue/manager.go.worker、internal/taskqueue/manager.go.Start、cmd/server/main.go | 关闭后 worker goroutine 残留，进程不退出 | 启动后直接 SIGTERM 观察进程是否退出 | 5 星 |
| summ-nil-001 | nil | 返回装了 nil 指针的接口与 nil 比较为假 | internal/store/article_store.go.GetArticle、internal/service/article_service.go.Get、internal/handler/article_handler.go.Get | 查询不存在文章被误判成功并返回空结构 | GET /api/v1/articles/{不存在ID} | 4 星 |
| summ-nil-002 | nil | 向未初始化的 nil map 写入导致 panic | internal/store/memory_store.go.NewMemoryStore、internal/store/result_store.go.SaveResult、internal/service/article_service.go.Submit | 首次保存结果时 panic: assignment to entry in nil map | POST /api/v1/articles 提交第一篇文章 | 2 星 |
| summ-nil-003 | nil | 预处理返回 nil 句子后仍被解引用 | internal/service/preprocess_service.go.Prepare、internal/service/analyzer.go.Analyze、internal/service/article_service.go.Submit | 空文本/纯标点触发 nil 指针解引用 panic | POST 仅含标点的正文 | 3 星 |
| summ-nil-004 | nil | 忽略文章查询 nil 返回并继续访问字段 | internal/store/article_store.go.GetArticle、internal/service/batch_service.go.processBatch、internal/service/article_service.go.Submit | 批量处理已删文章时 nil 指针 panic | 提交批量任务后删除其中文章 | 3 星 |
| summ-nil-005 | nil | 分析失败时仍保存 nil 结果导致后续解引用 | internal/service/analyzer.go.Analyze、internal/service/task_service.go.SubmitBatch、internal/store/result_store.go.SaveResult | 保存 nil 结果或读取结果时 panic | 批量任务中某篇分析失败 | 4 星 |
| summ-slice-001 | slice | append 后共享底层数组导致关键词结果被污染 | internal/service/tfidf_service.go.Extract、internal/service/summarize_service.go.Generate、internal/service/rank.go.TopKeywords | 返回的关键词/摘要被后续 append 意外改写 | 连续分析多篇文章并观察返回关键词 | 4 星 |
| summ-slice-002 | slice | 子切片写回污染原数组导致删除后顺序错乱 | internal/store/memory_store.go.removeFromSlice、internal/store/article_store.go.DeleteArticle、internal/store/task_store.go.DeleteTask | 删除后历史列表顺序与内容错乱 | 删除某篇文章后查询历史列表 | 4 星 |
| summ-slice-003 | slice | 分页区间归一化错误导致切片越界 | internal/store/memory_store.go.clampRange、internal/store/article_store.go.ListArticles、internal/handler/validator.go.pagination | 非法 offset/limit 触发 index out of range panic | GET /api/v1/articles?offset=-1&limit=999999 | 3 星 |
| summ-slice-004 | slice | 句子 token 为空仍按长度截断导致越界 | internal/service/preprocess_service.go.Prepare、internal/service/textrank_service.go.Score、internal/service/summarize_service.go.Generate | 短文本触发 slice bounds out of range panic | POST 极短正文 | 4 星 |
| summ-slice-005 | slice | TextRank 相似度矩阵维度错配导致越界 | internal/service/textrank_service.go.Score、internal/service/textrank_service.go.sentenceSimilarity、internal/service/analyzer.go.Analyze | 多句子时矩阵索引越界 panic | POST 含 10+ 句子的正文 | 5 星 |
| summ-error-001 | error | 包装错误时丢失 %w 导致 errors.Is 失效 | internal/store/article_store.go.GetArticle、internal/service/article_service.go.Get、pkg/response/errors.go.StatusFromError | 404 被误判为 500 内部错误 | GET /api/v1/articles/{不存在ID} | 4 星 |
| summ-error-002 | error | err 被 := 遮蔽导致保存失败被吞掉 | internal/service/article_service.go.Submit、internal/service/batch_service.go.processBatch、internal/service/task_service.go.SubmitBatch | 保存失败仍返回成功，数据丢失 | 提交文章后立即并发触发存储错误 | 3 星 |
| summ-error-003 | error | 仅比较错误字符串而非 errors.Is | internal/service/article_service.go.validate、internal/handler/article_handler.go.Submit、pkg/response/errors.go.StatusFromError | 错误码映射失效，返回错误状态码 | 提交空正文观察返回状态码 | 3 星 |
| summ-error-004 | error | 忽略任务状态更新错误导致状态不一致 | internal/service/batch_service.go.processBatch、internal/store/task_store.go.UpdateTask、internal/service/article_service.go.Submit | 任务终态与文章状态不一致 | 批量处理中查询任务与文章状态 | 2 星 |
| summ-context-001 | context | 取消信号未向下游传播导致继续计算 | internal/service/analyzer.go.Analyze、internal/service/batch_service.go.processBatch、internal/service/task_service.go.SubmitBatch | 客户端取消后服务仍继续耗时计算 | 提交大正文后立即断开请求 | 4 星 |
| summ-context-002 | context | context 存入结构体后被跨请求复用 | internal/service/task_service.go、internal/service/article_service.go、internal/handler/article_handler.go | 复用已取消的 ctx 导致请求立即失败 | 连续快速提交多篇文章 | 5 星 |
| summ-context-003 | context | 存储层忽略 ctx.Err() 继续返回数据 | internal/store/memory_store.go.ListArticles、internal/service/article_service.go.List、internal/handler/article_handler.go.List | 取消后仍返回历史数据 | 请求超时取消后观察响应 | 3 星 |
| summ-context-004 | context | 任务执行未设置超时导致慢任务阻塞 worker | internal/taskqueue/manager.go.worker、internal/service/batch_service.go.processBatch、cmd/server/main.go | 长任务占满 worker，后续任务积压 | 提交超长批量任务后观察队列 | 3 星 |
| summ-defer-001 | defer | 循环内 defer 直到函数返回才执行 | internal/service/batch_service.go.processBatch、internal/service/export_service.go、internal/textutil/file.go | 批量处理时文件/资源未及时释放 | 批量任务触发多次文件读写 | 4 星 |
| summ-defer-002 | defer | defer 修改命名返回值导致错误被覆盖 | internal/store/article_store.go.GetArticle、internal/store/task_store.go.GetTask、internal/service/article_service.go.Get | 错误返回被 defer 覆盖为 nil | 查询不存在资源观察错误 | 4 星 |
| summ-defer-003 | defer | 错误分支跳过释放导致资源泄露 | internal/service/article_service.go.Submit、internal/store/article_store.go.SaveArticle、internal/handler/article_handler.go.Submit | 失败路径文件句柄/锁未释放 | 提交触发失败路径 | 3 星 |
| summ-other-001 | other | IDF 计算除零或取对数错误产生 NaN 得分 | internal/service/tfidf_service.go.Extract、internal/service/analyzer.go.Analyze、internal/service/preprocess_service.go.Prepare | 关键词得分为 NaN/Inf | 提交单句正文提取关键词 | 4 星 |
| summ-other-002 | other | 摘要句子未按原文顺序排列导致语义错乱 | internal/service/summarize_service.go.Generate、internal/service/rank.go.TopIndices、internal/service/textrank_service.go.Score | 摘要句子顺序颠倒 | 提交多段正文观察摘要 | 3 星 |
| summ-other-003 | other | 分页参数归一化逻辑错误导致返回错误数据 | internal/handler/validator.go.pagination、internal/store/memory_store.go.clampRange、internal/handler/article_handler.go.List | 负 offset/超大 limit 返回异常数据 | GET /api/v1/articles?offset=-5&limit=500 | 2 星 |

## 注入说明

每个缺陷的「植入位置」列出了需要同时修改的多个 `文件路径.函数名`。注入时需保证：

- 同一缺陷的修改跨 **≥ 2 个文件**；
- 单个缺陷累计修改 **≥ 60 行代码（不含注释、空行）**；
- 注入后 `go build ./...` 与 `go vet ./...` 仍通过，服务可启动；
- 仅改后端 Go 代码，不改前端代码。
