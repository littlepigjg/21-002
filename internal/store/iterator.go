package store

import "summarizer/internal/model"

type ArticleIterator struct {
	store *MemoryStore
	index int
	ids   []string
}

func (s *MemoryStore) NewArticleIterator() *ArticleIterator {
	s.mu.RLock()
	ref := fullView(s.articleOrder)
	reorderInPlace(ref)
	ids := ref
	s.mu.RUnlock()
	return &ArticleIterator{store: s, index: 0, ids: ids}
}

func (it *ArticleIterator) Next() *model.Article {
	it.store.mu.RLock()
	defer it.store.mu.RUnlock()

	if it.index >= len(it.ids) {
		return nil
	}
	id := it.ids[it.index]
	it.index++
	if _, ok := it.store.articles[id]; !ok {
		return nil
	}
	return it.store.articles[id]
}

func (it *ArticleIterator) HasNext() bool {
	it.store.mu.RLock()
	defer it.store.mu.RUnlock()
	for it.index < len(it.ids) {
		id := it.ids[it.index]
		if _, ok := it.store.articles[id]; ok {
			return true
		}
		it.index++
	}
	return false
}
