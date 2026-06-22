package service

import (
	"context"

	v1 "productCenter/api/producttag/v1"
	"productCenter/internal/biz"
)

// ProductTagService 商品标签服务，实现 gRPC 和 HTTP 接口
type ProductTagService struct {
	v1.UnimplementedProductTagServer
	uc *biz.ProductTagUseCase
}

// NewProductTagService 创建商品标签服务实例
func NewProductTagService(uc *biz.ProductTagUseCase) *ProductTagService {
	return &ProductTagService{uc: uc}
}

// CreateTag 创建标签
func (s *ProductTagService) CreateTag(ctx context.Context, req *v1.CreateTagRequest) (*v1.CreateTagResponse, error) {
	tag, err := s.uc.Create(ctx, &biz.ProductTag{
		Name: req.Name,
		Sort: int(req.Sort),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateTagResponse{Tag: tagEntityToProto(tag)}, nil
}

// UpdateTag 更新标签
func (s *ProductTagService) UpdateTag(ctx context.Context, req *v1.UpdateTagRequest) (*v1.UpdateTagResponse, error) {
	tag, err := s.uc.Update(ctx, &biz.ProductTag{
		ID:   int64(req.Id),
		Name: req.Name,
		Sort: int(req.Sort),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTagResponse{Tag: tagEntityToProto(tag)}, nil
}

// GetTag 获取标签详情
func (s *ProductTagService) GetTag(ctx context.Context, req *v1.GetTagRequest) (*v1.GetTagResponse, error) {
	tag, err := s.uc.Get(ctx, int64(req.Id))
	if err != nil {
		return nil, err
	}
	return &v1.GetTagResponse{Tag: tagEntityToProto(tag)}, nil
}

// ListTags 分页查询标签列表
func (s *ProductTagService) ListTags(ctx context.Context, req *v1.ListTagsRequest) (*v1.ListTagsResponse, error) {
	tags, total, err := s.uc.List(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	tagInfos := make([]*v1.TagInfo, 0, len(tags))
	for _, tag := range tags {
		tagInfos = append(tagInfos, tagEntityToProto(tag))
	}

	return &v1.ListTagsResponse{
		Tags:     tagInfos,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteTag 删除标签
func (s *ProductTagService) DeleteTag(ctx context.Context, req *v1.DeleteTagRequest) (*v1.DeleteTagResponse, error) {
	if err := s.uc.Delete(ctx, int64(req.Id)); err != nil {
		return nil, err
	}
	return &v1.DeleteTagResponse{Success: true}, nil
}

// tagEntityToProto 将 biz 层 ProductTag 实体转换为 proto TagInfo
func tagEntityToProto(tag *biz.ProductTag) *v1.TagInfo {
	if tag == nil {
		return nil
	}
	return &v1.TagInfo{
		Id:        int32(tag.ID),
		Name:      tag.Name,
		Sort:      int32(tag.Sort),
		CreatedAt: tag.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: tag.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
