package images

import "time"

type ProductImage struct {
	Id           string
	ProjectId    string
	StorageKey   string
	ContentType  string
	SizeBytes    int
	OriginalName string
	CreatedAt    time.Time
}

type GeneratedImage struct {
	Id           string
	ProjectId    string
	GenerationId string // задача генерации в Higgsfield
	StorageKey   string // копия на диске, источник истины
	ContentType  string
	SizeBytes    int64
	SourceURL    string // ссылка от Higgsfield, только для отладки
	Status       string // pending_review | approved | rejected
	Version      int
	CreatedAt    time.Time
}
