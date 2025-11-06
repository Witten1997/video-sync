-- ============================================
-- bili-sync PostgreSQL Database Schema
-- ============================================
-- Version: 2.0
-- Created: 2025-10-31
-- Description: Complete database schema for bili-sync project
--              Based on bili-sync-refactor-requirements.md and bili-download models
-- ============================================

-- Drop existing tables (for clean install)
DROP TABLE IF EXISTS page CASCADE;
DROP TABLE IF EXISTS video CASCADE;
DROP TABLE IF EXISTS favorite CASCADE;
DROP TABLE IF EXISTS watch_later CASCADE;
DROP TABLE IF EXISTS collection CASCADE;
DROP TABLE IF EXISTS submission CASCADE;
DROP TABLE IF EXISTS config CASCADE;

-- ============================================
-- 1. Video Source Tables (视频源表)
-- ============================================

-- 1.1 Favorite Table (收藏夹)
CREATE TABLE favorite (
    id SERIAL PRIMARY KEY,
    f_id BIGINT NOT NULL UNIQUE,                -- B站收藏夹 ID
    name VARCHAR(255) NOT NULL,                 -- 收藏夹名称
    path VARCHAR(500),                          -- 保存路径模板
    enabled BOOLEAN DEFAULT TRUE NOT NULL,      -- 是否启用
    latest_row_at TIMESTAMP WITH TIME ZONE,     -- 最后扫描到的视频时间
    rule JSONB,                                 -- 过滤规则 (JSON格式)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

COMMENT ON TABLE favorite IS '收藏夹视频源';
COMMENT ON COLUMN favorite.f_id IS 'B站收藏夹ID';
COMMENT ON COLUMN favorite.path IS '视频保存路径模板';
COMMENT ON COLUMN favorite.enabled IS '是否启用该视频源';
COMMENT ON COLUMN favorite.latest_row_at IS '最后一次扫描到的视频时间，用于增量同步';
COMMENT ON COLUMN favorite.rule IS '过滤规则，JSONB格式存储';

-- 1.2 Watch Later Table (稍后再看)
CREATE TABLE watch_later (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) DEFAULT '稍后再看' NOT NULL,
    path VARCHAR(500),
    enabled BOOLEAN DEFAULT TRUE NOT NULL,
    latest_row_at TIMESTAMP WITH TIME ZONE,
    rule JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

COMMENT ON TABLE watch_later IS '稍后再看视频源 (单例模式)';
COMMENT ON COLUMN watch_later.name IS '视频源名称';

-- 1.3 Collection Table (合集/列表)
CREATE TABLE collection (
    id SERIAL PRIMARY KEY,
    c_id BIGINT NOT NULL UNIQUE,                -- B站合集 ID
    c_type VARCHAR(20),                         -- 合集类型 (series/season)
    name VARCHAR(255) NOT NULL,
    path VARCHAR(500),
    enabled BOOLEAN DEFAULT TRUE NOT NULL,
    latest_row_at TIMESTAMP WITH TIME ZONE,
    rule JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

COMMENT ON TABLE collection IS '合集/列表视频源';
COMMENT ON COLUMN collection.c_id IS 'B站合集ID';
COMMENT ON COLUMN collection.c_type IS '合集类型: series(视频列表) 或 season(番剧季度)';

-- 1.4 Submission Table (UP主投稿)
CREATE TABLE submission (
    id SERIAL PRIMARY KEY,
    upper_id BIGINT NOT NULL UNIQUE,            -- UP主 ID
    name VARCHAR(255) NOT NULL,                 -- UP主名称
    path VARCHAR(500),
    enabled BOOLEAN DEFAULT TRUE NOT NULL,
    latest_row_at TIMESTAMP WITH TIME ZONE,
    use_dynamic_api BOOLEAN DEFAULT FALSE NOT NULL, -- 是否使用动态 API
    rule JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

COMMENT ON TABLE submission IS 'UP主投稿视频源';
COMMENT ON COLUMN submission.upper_id IS 'UP主的用户ID (mid)';
COMMENT ON COLUMN submission.use_dynamic_api IS '是否使用动态API获取投稿列表';

-- ============================================
-- 2. Video and Page Tables (视频和分P表)
-- ============================================

-- 2.1 Video Table (视频主表)
CREATE TABLE video (
    id SERIAL PRIMARY KEY,
    bvid VARCHAR(20) NOT NULL UNIQUE,           -- B站视频BV号
    name VARCHAR(255) NOT NULL,                 -- 视频标题
    intro TEXT,                                 -- 视频简介
    cover VARCHAR(500),                         -- 封面图片URL
    tags JSONB,                                 -- 视频标签 (JSON数组)
    upper_id BIGINT NOT NULL,                   -- UP主ID
    upper_name VARCHAR(100),                    -- UP主名称
    upper_face VARCHAR(500),                    -- UP主头像URL
    category INTEGER,                           -- 视频分区ID
    pubtime TIMESTAMP WITH TIME ZONE NOT NULL,  -- 发布时间
    favtime TIMESTAMP WITH TIME ZONE NOT NULL,  -- 收藏时间
    ctime TIMESTAMP WITH TIME ZONE NOT NULL,    -- 创建时间
    single_page BOOLEAN,                        -- 是否为单P视频
    valid BOOLEAN DEFAULT TRUE NOT NULL,        -- 视频是否有效(未被删除)
    should_download BOOLEAN DEFAULT TRUE NOT NULL, -- 是否应该下载
    download_status INTEGER DEFAULT 0 NOT NULL, -- 下载状态位标志
    path VARCHAR(500),                          -- 本地存储路径

    -- 外键关系 (只能属于一个视频源)
    favorite_id INTEGER,
    watch_later_id INTEGER,
    collection_id INTEGER,
    submission_id INTEGER,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    -- 外键约束
    CONSTRAINT fk_video_favorite FOREIGN KEY (favorite_id) REFERENCES favorite(id) ON DELETE CASCADE,
    CONSTRAINT fk_video_watch_later FOREIGN KEY (watch_later_id) REFERENCES watch_later(id) ON DELETE CASCADE,
    CONSTRAINT fk_video_collection FOREIGN KEY (collection_id) REFERENCES collection(id) ON DELETE CASCADE,
    CONSTRAINT fk_video_submission FOREIGN KEY (submission_id) REFERENCES submission(id) ON DELETE CASCADE,

    -- 确保视频只属于一个视频源
    CONSTRAINT chk_single_source CHECK (
        (CASE WHEN favorite_id IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN watch_later_id IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN collection_id IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN submission_id IS NOT NULL THEN 1 ELSE 0 END) <= 1
    )
);

COMMENT ON TABLE video IS '视频主表，存储所有视频的元数据';
COMMENT ON COLUMN video.bvid IS 'B站视频唯一标识符 (BV号)';
COMMENT ON COLUMN video.tags IS '视频标签，以JSONB数组形式存储';
COMMENT ON COLUMN video.upper_id IS 'UP主的用户ID';
COMMENT ON COLUMN video.pubtime IS '视频在B站的发布时间';
COMMENT ON COLUMN video.favtime IS '视频被收藏的时间';
COMMENT ON COLUMN video.single_page IS 'true=单P视频, false=多P视频';
COMMENT ON COLUMN video.valid IS 'false表示视频已被删除或不可访问';
COMMENT ON COLUMN video.should_download IS '根据过滤规则判断是否需要下载';
COMMENT ON COLUMN video.download_status IS '位标志，每5位表示一个子任务的下载状态';

-- 2.2 Page Table (视频分P表)
CREATE TABLE page (
    id SERIAL PRIMARY KEY,
    video_id INTEGER NOT NULL,                  -- 所属视频ID
    cid BIGINT NOT NULL,                        -- B站分P的CID
    pid INTEGER NOT NULL,                       -- 分P编号 (从1开始)
    name VARCHAR(255),                          -- 分P标题
    duration INTEGER,                           -- 时长(秒)
    width INTEGER,                              -- 视频宽度
    height INTEGER,                             -- 视频高度
    image VARCHAR(500),                         -- 分P封面URL
    download_status INTEGER DEFAULT 0 NOT NULL, -- 下载状态位标志
    path VARCHAR(500),                          -- 本地存储路径
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    -- 外键约束
    CONSTRAINT fk_page_video FOREIGN KEY (video_id) REFERENCES video(id) ON DELETE CASCADE,

    -- 唯一约束 (同一视频下的CID不能重复)
    CONSTRAINT uq_page_video_cid UNIQUE (video_id, cid)
);

COMMENT ON TABLE page IS '视频分P表，存储多P视频的每一集信息';
COMMENT ON COLUMN page.cid IS 'B站分P的唯一标识符';
COMMENT ON COLUMN page.pid IS '分P序号 (从1开始)';
COMMENT ON COLUMN page.duration IS '分P时长，单位为秒';
COMMENT ON COLUMN page.download_status IS '位标志，每5位表示一个子任务的下载状态';

-- ============================================
-- 3. Configuration Table (配置表)
-- ============================================

CREATE TABLE config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,           -- 配置项键名
    value TEXT NOT NULL,                        -- 配置项值 (JSON格式)
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

COMMENT ON TABLE config IS '系统配置表，存储应用配置项';
COMMENT ON COLUMN config.key IS '配置项键名 (唯一)';
COMMENT ON COLUMN config.value IS '配置项的值，通常以JSON格式存储';

-- ============================================
-- 4. Indexes (索引)
-- ============================================

-- 4.1 Video Table Indexes
CREATE INDEX idx_video_bvid ON video(bvid);
CREATE INDEX idx_video_upper_id ON video(upper_id);
CREATE INDEX idx_video_pubtime ON video(pubtime DESC);
CREATE INDEX idx_video_favtime ON video(favtime DESC);
CREATE INDEX idx_video_created_at ON video(created_at DESC);
CREATE INDEX idx_video_favorite_id ON video(favorite_id) WHERE favorite_id IS NOT NULL;
CREATE INDEX idx_video_watch_later_id ON video(watch_later_id) WHERE watch_later_id IS NOT NULL;
CREATE INDEX idx_video_collection_id ON video(collection_id) WHERE collection_id IS NOT NULL;
CREATE INDEX idx_video_submission_id ON video(submission_id) WHERE submission_id IS NOT NULL;

-- 4.2 Page Table Indexes
CREATE INDEX idx_page_video_id ON page(video_id);
CREATE INDEX idx_page_cid ON page(cid);

-- 4.3 Video Source Tables Indexes
CREATE INDEX idx_favorite_f_id ON favorite(f_id);
CREATE INDEX idx_favorite_enabled ON favorite(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_watch_later_enabled ON watch_later(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_collection_c_id ON collection(c_id);
CREATE INDEX idx_collection_enabled ON collection(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_submission_upper_id ON submission(upper_id);
CREATE INDEX idx_submission_enabled ON submission(enabled) WHERE enabled = TRUE;

-- ============================================
-- 5. PostgreSQL-Specific Optimizations (PG特定优化)
-- ============================================

-- 5.1 JSONB GIN Indexes (JSONB GIN索引 - 用于快速JSON查询)
CREATE INDEX idx_favorite_rule ON favorite USING GIN (rule) WHERE rule IS NOT NULL;
CREATE INDEX idx_watch_later_rule ON watch_later USING GIN (rule) WHERE rule IS NOT NULL;
CREATE INDEX idx_collection_rule ON collection USING GIN (rule) WHERE rule IS NOT NULL;
CREATE INDEX idx_submission_rule ON submission USING GIN (rule) WHERE rule IS NOT NULL;
CREATE INDEX idx_video_tags ON video USING GIN (tags) WHERE tags IS NOT NULL;

-- 5.2 Partial Indexes (部分索引 - 针对常用查询条件)
-- 只索引需要下载的有效视频
CREATE INDEX idx_video_downloadable ON video(id, bvid, name)
WHERE should_download = TRUE AND valid = TRUE;

-- 只索引未完成下载的视频
CREATE INDEX idx_video_pending_download ON video(id, download_status)
WHERE download_status > 0 AND should_download = TRUE;

-- 只索引未完成下载的分P
CREATE INDEX idx_page_pending_download ON page(id, video_id, download_status)
WHERE download_status > 0;

-- 5.3 Composite Indexes (复合索引 - 用于常见查询组合)
CREATE INDEX idx_video_source_time ON video(favorite_id, favtime DESC NULLS LAST)
WHERE favorite_id IS NOT NULL;

CREATE INDEX idx_video_upper_pubtime ON video(upper_id, pubtime DESC);

-- ============================================
-- 6. Functions and Triggers (函数和触发器)
-- ============================================

-- 6.1 自动更新 updated_at 字段的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 6.2 为 config 表添加触发器
CREATE TRIGGER trigger_config_updated_at
    BEFORE UPDATE ON config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6.3 视频统计函数 (获取视频源的统计信息)
CREATE OR REPLACE FUNCTION get_favorite_stats(fav_id INTEGER)
RETURNS TABLE(
    total_videos BIGINT,
    downloaded_videos BIGINT,
    pending_videos BIGINT,
    invalid_videos BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*) as total_videos,
        COUNT(*) FILTER (WHERE download_status = 0 AND should_download = TRUE) as downloaded_videos,
        COUNT(*) FILTER (WHERE download_status > 0 AND should_download = TRUE) as pending_videos,
        COUNT(*) FILTER (WHERE valid = FALSE) as invalid_videos
    FROM video
    WHERE favorite_id = fav_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 7. Initial Data (初始数据)
-- ============================================

-- 插入默认配置项 (可根据需要添加)
INSERT INTO config (key, value) VALUES
    ('version', '"2.0.0"'),
    ('database_initialized', 'true'),
    ('schema_version', '"1.0.0"')
ON CONFLICT (key) DO NOTHING;

-- ============================================
-- 8. Database Maintenance (数据库维护)
-- ============================================

-- 8.1 Enable auto-vacuum for all tables
ALTER TABLE video SET (autovacuum_enabled = true, autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE page SET (autovacuum_enabled = true, autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE favorite SET (autovacuum_enabled = true);
ALTER TABLE watch_later SET (autovacuum_enabled = true);
ALTER TABLE collection SET (autovacuum_enabled = true);
ALTER TABLE submission SET (autovacuum_enabled = true);

-- ============================================
-- 9. Useful Queries (常用查询)
-- ============================================

-- 查询所有启用的视频源及其视频统计
-- SELECT f.id, f.name, f.enabled,
--        COUNT(v.id) as total_videos,
--        COUNT(v.id) FILTER (WHERE v.should_download = TRUE AND v.download_status = 0) as downloaded,
--        COUNT(v.id) FILTER (WHERE v.should_download = TRUE AND v.download_status > 0) as pending
-- FROM favorite f
-- LEFT JOIN video v ON v.favorite_id = f.id
-- WHERE f.enabled = TRUE
-- GROUP BY f.id, f.name, f.enabled;

-- 查询待下载���视频列表
-- SELECT v.id, v.bvid, v.name, v.upper_name, v.download_status
-- FROM video v
-- WHERE v.should_download = TRUE
--   AND v.valid = TRUE
--   AND v.download_status > 0
-- ORDER BY v.favtime DESC;

-- 查询某个视频的所有分P及下载状态
-- SELECT p.pid, p.name, p.duration, p.download_status
-- FROM page p
-- WHERE p.video_id = ?
-- ORDER BY p.pid;

-- ============================================
-- End of Schema
-- ============================================

-- Verify tables are created
SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;

COMMENT ON DATABASE bili_sync IS 'bili-sync 视频同步工具数据库 - 基于PostgreSQL';
