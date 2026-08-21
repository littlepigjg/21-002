package textutil

// enStopwords 是内置的英文停用词表，覆盖冠词、介词、代词、连词及常见助动词。
var enStopwords = []string{
	"a", "an", "the", "and", "or", "but", "if", "then", "else", "for",
	"while", "do", "does", "did", "is", "are", "was", "were", "be", "been",
	"being", "to", "of", "in", "on", "at", "by", "with", "from", "as",
	"it", "its", "this", "that", "these", "those", "i", "you", "he", "she",
	"we", "they", "them", "my", "your", "his", "her", "our", "their", "me",
	"him", "us", "not", "no", "nor", "so", "than", "too", "very", "can",
	"will", "just", "should", "could", "would", "may", "might", "must", "about", "into",
	"through", "during", "before", "after", "above", "below", "between", "out", "up", "down",
	"again", "further", "once", "here", "there", "when", "where", "why", "how", "all",
	"any", "both", "each", "few", "more", "most", "other", "some", "such", "only",
	"own", "same", "s", "t", "has", "have", "having", "don", "now", "d",
	"ll", "m", "o", "re", "ve", "y", "ain", "aren", "couldn", "didn",
	"doesn", "hadn", "hasn", "haven", "isn", "ma", "mightn", "mustn", "needn", "shan",
	"shouldn", "wasn", "weren", "won", "wouldn", "also", "however", "therefore", "thus", "yet",
}
