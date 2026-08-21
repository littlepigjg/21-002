package search

import (
	"sort"

	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

// Posting 表示某个词在一篇文章中的出现次数。
type Posting struct {
	ArticleID string
	Count     int
}

// Hit 表示一条搜索结果。
type Hit struct {
	ArticleID string
	Score     float64
}

// Index 是基于内存的倒排索引，用于按关键词检索文章。
// 注意：Index 非并发安全，应在单 goroutine 或外部加锁下使用。
type Index struct {
	postings  map[string][]Posting
	articles  map[string]*model.Article
	stopwords *textutil.StopwordSet
}

// NewIndex 构造一个空索引。
func NewIndex(stopwords *textutil.StopwordSet) *Index {
	return &Index{
		postings:  make(map[string][]Posting),
		articles:  make(map[string]*model.Article),
		stopwords: stopwords,
	}
}

// Add 将一篇文章加入索引，按 token 建立倒排。
func (idx *Index) Add(article *model.Article) {
	idx.articles[article.ID] = article

	tokens := textutil.Tokenize(article.Content)
	tokens = idx.stopwords.Filter(tokens)

	freq := make(map[string]int)
	for _, t := range tokens {
		freq[t]++
	}
	for word, count := range freq {
		idx.postings[word] = append(idx.postings[word], Posting{ArticleID: article.ID, Count: count})
	}
}

// Remove 从索引中移除指定文章，并清理不再存在的倒排词条。
func (idx *Index) Remove(articleID string) {
	delete(idx.articles, articleID)

	for word, postings := range idx.postings {
		out := postings[:0]
		for _, p := range postings {
			if p.ArticleID != articleID {
				out = append(out, p)
			}
		}
		if len(out) == 0 {
			delete(idx.postings, word)
		} else {
			idx.postings[word] = out
		}
	}
}

// Search 根据查询返回匹配文章及其得分，按得分降序排列，最多 limit 条。
func (idx *Index) Search(query string, limit int) []Hit {
	tokens := tokenizeQuery(query, idx.stopwords)
	if len(tokens) == 0 || limit <= 0 {
		return nil
	}

	scores := make(map[string]float64)
	for _, t := range tokens {
		for _, p := range idx.postings[t] {
			scores[p.ArticleID] += float64(p.Count)
		}
	}

	hits := make([]Hit, 0, len(scores))
	for id, score := range scores {
		hits = append(hits, Hit{ArticleID: id, Score: score})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ArticleID < hits[j].ArticleID
	})

	if limit > len(hits) {
		limit = len(hits)
	}
	return hits[:limit]
}

// DocCount 返回索引中的文档数量。
func (idx *Index) DocCount() int {
	return len(idx.articles)
}
