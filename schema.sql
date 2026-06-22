-- ============================================================
-- 商品中心（Product Center）建表 DDL — 最终版
-- 数据库：MySQL 8.0+
-- 引擎：InnoDB
-- 价格单位：分（INT）
-- 删除策略：软删除（deleted_at）
-- 约束：逻辑外键
-- ============================================================

-- 1. 店铺表
create table shops
(
    id          bigint auto_increment comment '主键id'
        primary key,
    shop_name   varchar(255) not null comment '商店名称',
    description text         null comment '商店描述',
    created_at  datetime     null comment '创建时间',
    updated_at  datetime     null comment '修改时间',
    deleted_at  datetime     null comment '软删除时间'
) comment '商店表';

create index idx_shop_name
    on shops (shop_name);


-- 2. 商品标签表（原 product_type，现为标签）
create table product_tag
(
    id         int auto_increment comment '主键'
        primary key,
    name       varchar(500)  not null comment '标签名称',
    sort       int default 0 not null comment '排序',
    created_at datetime      not null comment '创建时间',
    updated_at datetime      not null comment '修改时间',
    deleted_at datetime      null comment '软删除时间'
) comment '商品标签表';

create index idx_product_tag_sort
    on product_tag (sort);


-- 3. 商品表
create table products
(
    id               bigint auto_increment comment '主键id'
        primary key,
    shop_id          bigint        not null comment '商店id（逻辑外键）',
    name             varchar(255)  not null comment '商品名称',
    description      text          null comment '商品描述',
    main_image_url   varchar(500)  not null comment '主商品图片',
    price            int           not null comment '售价（分）',
    compare_at_price int           null comment '划线价（分）',
    status           tinyint       not null default 0 comment '状态：0=草稿 1=上架 2=下架',
    sort             int default 0 not null comment '排序',
    created_at       datetime      not null comment '创建时间',
    updated_at       datetime      not null comment '修改时间',
    deleted_at       datetime      null comment '软删除时间'
);

create index idx_products_shop_status
    on products (shop_id, status);

create index idx_products_status
    on products (status);


-- 4. 商品副图表
create table product_media
(
    id         bigint auto_increment comment '主键id'
        primary key,
    product_id bigint        not null comment '商品id（逻辑外键）',
    url        varchar(500)  null comment '图片url',
    sort       int default 0 not null comment '图片排序',
    created_at datetime      not null comment '创建日期',
    updated_at datetime      not null comment '更新日期',
    deleted_at datetime      null comment '软删除时间'
) comment '商品副图表';

create index idx_media_product
    on product_media (product_id);


-- 5. 商品标签关联表（多对多）
create table product_tag_mapping
(
    product_id bigint  not null comment '商品id（逻辑外键）',
    tag_id     int     not null comment '标签id（逻辑外键）',
    created_at datetime null comment '创建时间',
    primary key (product_id, tag_id)
) comment '商品-标签关联表';

create index idx_tag_mapping_product
    on product_tag_mapping (product_id);

create index idx_tag_mapping_tag
    on product_tag_mapping (tag_id);


-- 6. 商品SKU表
create table sku
(
    id         bigint auto_increment comment '主键id'
        primary key,
    product_id bigint       not null comment '商品id（逻辑外键）',
    sku        varchar(255) not null comment 'sku编码',
    price      int          null comment '价格（分）',
    stock      int          null comment '库存',
    title      varchar(100) not null comment 'sku标题',
    img_url    varchar(500) null comment 'sku图',
    created_at datetime     null comment '创建时间',
    updated_at datetime     not null comment '更新时间',
    deleted_at datetime     null comment '软删除时间'
) comment '商品SKU表';

create index idx_sku_product
    on sku (product_id);
