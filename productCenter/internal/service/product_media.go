package service

import (
	"context"

	v1 "productCenter/api/productmedia/v1"
	"productCenter/internal/biz"
)

// ProductMediaService 商品副图服务，实现 gRPC 和 HTTP 接口
type ProductMediaService struct {
	v1.UnimplementedProductMediaServer
	uc *biz.ProductMediaUseCase
}

// NewProductMediaService 创建商品副图服务实例
func NewProductMediaService(uc *biz.ProductMediaUseCase) *ProductMediaService {
	return &ProductMediaService{uc: uc}
}

// CreateMedia 创建副图
func (s *ProductMediaService) CreateMedia(ctx context.Context, req *v1.CreateMediaRequest) (*v1.CreateMediaResponse, error) {
	media, err := s.uc.Create(ctx, &biz.ProductMedia{
		ProductID: req.ProductId,
		URL:       req.Url,
		Sort:      int(req.Sort),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateMediaResponse{Media: mediaEntityToProto(media)}, nil
}

// UpdateMedia 更新副图
func (s *ProductMediaService) UpdateMedia(ctx context.Context, req *v1.UpdateMediaRequest) (*v1.UpdateMediaResponse, error) {
	media, err := s.uc.Update(ctx, &biz.ProductMedia{
		ID:        req.Id,
		ProductID: req.ProductId,
		URL:       req.Url,
		Sort:      int(req.Sort),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateMediaResponse{Media: mediaEntityToProto(media)}, nil
}

// GetMedia 获取副图详情
func (s *ProductMediaService) GetMedia(ctx context.Context, req *v1.GetMediaRequest) (*v1.GetMediaResponse, error) {
	media, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetMediaResponse{Media: mediaEntityToProto(media)}, nil
}

// ListMedias 分页查询副图列表，支持按商品筛选
func (s *ProductMediaService) ListMedias(ctx context.Context, req *v1.ListMediasRequest) (*v1.ListMediasResponse, error) {
	medias, total, err := s.uc.List(ctx, req.Page, req.PageSize, req.ProductId)
	if err != nil {
		return nil, err
	}

	mediaInfos := make([]*v1.MediaInfo, 0, len(medias))
	for _, media := range medias {
		mediaInfos = append(mediaInfos, mediaEntityToProto(media))
	}

	return &v1.ListMediasResponse{
		Medias:   mediaInfos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteMedia 删除副图
func (s *ProductMediaService) DeleteMedia(ctx context.Context, req *v1.DeleteMediaRequest) (*v1.DeleteMediaResponse, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteMediaResponse{Success: true}, nil
}

// mediaEntityToProto 将 biz 层 ProductMedia 实体转换为 proto MediaInfo
func mediaEntityToProto(media *biz.ProductMedia) *v1.MediaInfo {
	if media == nil {
		return nil
	}
	return &v1.MediaInfo{
		Id:        media.ID,
		ProductId: media.ProductID,
		Url:       media.URL,
		Sort:      int64(media.Sort),
		CreatedAt: media.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: media.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
