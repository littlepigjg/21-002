package store

import "summarizer/internal/model"

// ArticleIterator 按插入顺序遍历文章。
// 注意：迭代器为单消费者设计，不应在多个 goroutine 中并发调用 Next。
type ArticleIterator struct {
	store *MemoryStore
	index int
}

// NewArticleIterator 构造一个文章迭代器。
func (s *MemoryStore) NewArticleIterator() *ArticleIterator {
	return &ArticleIterator{store: s, index: 0}
}

// Next 返回下一篇文章（深拷贝），遍历完毕返回 nil。
func (it *ArticleIterator) Next() *model.Article {
	it.store.mu.RLock()
	defer it.store.mu.RUnlock()

	if it.index >= len(it.store.articleOrder) {
		return nil
	}
	id := it.store.articleOrder[it.index]
	it.index++
	if a, ok := it.store.articles[id]; ok {
		return copyArticle(a)
	}
	return nil
}

// HasNext 判断是否还有下一篇文章。
func (it *ArticleIterator) HasNext() bool {
	it.store.mu.RLock()
	defer it.store.mu.RUnlock()
	return it.index < len(it.store.articleOrder)
}
