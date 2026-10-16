package images

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ImageResponse struct {
	Id          string `json:"id"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

type UploadImagesResponse struct {
	Images []ImageResponse `json:"images"`
}

type ListImagesRequest struct {
	Pagination Pagination `json:"pagination"`
}

type NewImage struct {
	StorageKey   string
	ContentType  string
	OriginalName string
	SizeBytes    int64
}

func toImageResponse(pi ProductImage) ImageResponse {
	return ImageResponse{
		Id:          pi.Id,
		ContentType: pi.ContentType,
		URL:         "/api/v1/images/" + pi.Id + "/file",
	}
}

func toImageResponses(images []ProductImage) []ImageResponse {
	productImages := make([]ImageResponse, 0, len(images))
	for _, i := range images {
		productImages = append(productImages, toImageResponse(i))
	}
	return productImages
}
