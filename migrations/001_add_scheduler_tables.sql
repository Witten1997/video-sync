-- ============================================
-- 定时任务功能 - 数据库迁移
-- ============================================
-- 版本: 001
-- 创建日期: 2025-01-15
-- 说明: 添加调度器相关表和扩展现有视频源表
-- ============================================

-- ============================================
-- 1. 新增表：sync_logs（同步日志）
-- ============================================
CREATE TABLE IF NOT EXISTS sync_logs (
    id                SERIAL PRIMARY KEY,
    task_id           TEXT NOT NULL UNIQUE,              -- 任务唯一标识（UUID）
    trigger_type      TEXT NOT NULL,                      -- 触发方式: auto/manual
    status            TEXT NOT NULL,                      -- 状态: running/completed/failed/cancelled
    start_at          TIMESTAMP WITH TIME ZONE NOT NULL,  -- 开始时间
    end_at            TIMESTAMP WITH TIME ZONE,           -- 结束时间
    duration_ms       INTEGER,                            -- 执行耗时（毫秒）

    sources_total     INTEGER DEFAULT 0,                  -- 总视频源数
    sources_scanned   INTEGER DEFAULT 0,                  -- 已扫描视频源数
    sources_failed    INTEGER DEFAULT 0,                  -- 扫描失败视频源数

    videos_found      INTEGER DEFAULT 0,                  -- 发现的视频总数
    videos_new        INTEGER DEFAULT 0,                  -- 新增的视频数
    videos_filtered   INTEGER DEFAULT 0,                  -- 被过滤掉的视频数
    videos_queued     INTEGER DEFAULT 0,                  -- 加入下载队列的视频数

    tasks_created     INTEGER DEFAULT 0,                  -- 创建的下载任务数
    tasks_completed   INTEGER DEFAULT 0,                  -- 完成的下载任务数
    tasks_failed      INTEGER DEFAULT 0,                  -- 失败的下载任务数

    error_message     TEXT,                               -- 错误信息
    metadata          JSONB,                              -- 额外元数据

    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE sync_logs IS '同步任务日志表';
COMMENT ON COLUMN sync_logs.task_id IS '任务唯一标识符（UUID格式）';
COMMENT ON COLUMN sync_logs.trigger_type IS '触发方式：auto（自动）或 manual（手动）';
COMMENT ON COLUMN sync_logs.status IS '任务状态：running/completed/failed/cancelled';
COMMENT ON COLUMN sync_logs.duration_ms IS '任务执行总耗时（毫秒）';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON sync_logs(status);
CREATE INDEX IF NOT EXISTS idx_sync_logs_start_at ON sync_logs(start_at DESC);
CREATE INDEX IF NOT EXISTS idx_sync_logs_trigger_type ON sync_logs(trigger_type);
CREATE INDEX IF NOT EXISTS idx_sync_logs_task_id ON sync_logs(task_id);

-- ============================================
-- 2. 新增表：video_source_scans（视频源扫描记录）
-- ============================================
CREATE TABLE IF NOT EXISTS video_source_scans (
    id                SERIAL PRIMARY KEY,
    sync_log_id       INTEGER NOT NULL,                   -- 关联的同步日志ID
    source_id         TEXT NOT NULL,                      -- 视频源ID（如 fav_123）
    source_type       TEXT NOT NULL,                      -- 视频源类型: favorite/submission/collection/watch_later
    source_name       TEXT,                               -- 视频源名称

    scanned_at        TIMESTAMP WITH TIME ZONE NOT NULL,  -- 扫描时间
    duration_ms       INTEGER,                            -- 扫描耗时（毫秒）
    success           BOOLEAN DEFAULT TRUE,               -- 是否成功
    error_message     TEXT,                               -- 错误信息

    videos_found      INTEGER DEFAULT 0,                  -- 发现的视频数
    videos_new        INTEGER DEFAULT 0,                  -- 新增的视频数
    videos_filtered   INTEGER DEFAULT 0,                  -- 被过滤的视频数
    videos_queued     INTEGER DEFAULT 0,                  -- 加入队列的视频数

    metadata          JSONB,                              -- 额外元数据（如API响应详情）

    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_vss_sync_log FOREIGN KEY (sync_log_id) REFERENCES sync_logs(id) ON DELETE CASCADE
);

COMMENT ON TABLE video_source_scans IS '视频源扫描详细记录';
COMMENT ON COLUMN video_source_scans.source_id IS '视频源标识符（格式：类型前缀_ID，如 fav_12345）';
COMMENT ON COLUMN video_source_scans.source_type IS '视频源类型';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_vss_sync_log_id ON video_source_scans(sync_log_id);
CREATE INDEX IF NOT EXISTS idx_vss_source_id ON video_source_scans(source_id);
CREATE INDEX IF NOT EXISTS idx_vss_scanned_at ON video_source_scans(scanned_at DESC);

-- ============================================
-- 3. 新增表：scheduler_state（调度器状态）
-- ============================================
CREATE TABLE IF NOT EXISTS scheduler_state (
    id                SERIAL PRIMARY KEY,
    is_running        BOOLEAN DEFAULT FALSE,              -- 是否正在运行
    last_run_at       TIMESTAMP WITH TIME ZONE,           -- 上次运行时间
    next_run_at       TIMESTAMP WITH TIME ZONE,           -- 下次运行时间
    current_sync_id   TEXT,                               -- 当前同步任务ID
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE scheduler_state IS '调度器状态表（单例）';

-- 初始化一条记录（如果不存在）
INSERT INTO scheduler_state (is_running)
SELECT FALSE
WHERE NOT EXISTS (SELECT 1 FROM scheduler_state LIMIT 1);

-- ============================================
-- 4. 扩展现有表：favorite（添加调度相关字段）
-- ============================================
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0;
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS health_status TEXT DEFAULT 'healthy';
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0;
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS last_scan_error TEXT;
ALTER TABLE favorite ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN favorite.priority IS '优先级（0-10，数字越大优先级越高）';
COMMENT ON COLUMN favorite.health_status IS '健康状态: healthy/degraded/unhealthy';
COMMENT ON COLUMN favorite.consecutive_failures IS '连续失败次数';
COMMENT ON COLUMN favorite.last_scan_at IS '最后扫描时间';
COMMENT ON COLUMN favorite.last_scan_error IS '最后扫描错误信息';
COMMENT ON COLUMN favorite.last_success_at IS '最后成功扫描时间';

-- ============================================
-- 5. 扩展现有表：watch_later（添加调度相关字段）
-- ============================================
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0;
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS health_status TEXT DEFAULT 'healthy';
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0;
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS last_scan_error TEXT;
ALTER TABLE watch_later ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN watch_later.priority IS '优先级（0-10，数字越大优先级越高）';
COMMENT ON COLUMN watch_later.health_status IS '健康状态: healthy/degraded/unhealthy';
COMMENT ON COLUMN watch_later.consecutive_failures IS '连续失败次数';
COMMENT ON COLUMN watch_later.last_scan_at IS '最后扫描时间';
COMMENT ON COLUMN watch_later.last_scan_error IS '最后扫描错误信息';
COMMENT ON COLUMN watch_later.last_success_at IS '最后成功扫描时间';

-- ============================================
-- 6. 扩展现有表：collection（添加调度相关字段）
-- ============================================
ALTER TABLE collection ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0;
ALTER TABLE collection ADD COLUMN IF NOT EXISTS health_status TEXT DEFAULT 'healthy';
ALTER TABLE collection ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0;
ALTER TABLE collection ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE collection ADD COLUMN IF NOT EXISTS last_scan_error TEXT;
ALTER TABLE collection ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN collection.priority IS '优先级（0-10，数字越大优先级越高）';
COMMENT ON COLUMN collection.health_status IS '健康状态: healthy/degraded/unhealthy';
COMMENT ON COLUMN collection.consecutive_failures IS '连续失败次数';
COMMENT ON COLUMN collection.last_scan_at IS '最后扫描时间';
COMMENT ON COLUMN collection.last_scan_error IS '最后扫描错误信息';
COMMENT ON COLUMN collection.last_success_at IS '最后成功扫描时间';

-- ============================================
-- 7. 扩展现有表：submission（添加调度相关字段）
-- ============================================
ALTER TABLE submission ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0;
ALTER TABLE submission ADD COLUMN IF NOT EXISTS health_status TEXT DEFAULT 'healthy';
ALTER TABLE submission ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0;
ALTER TABLE submission ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE submission ADD COLUMN IF NOT EXISTS last_scan_error TEXT;
ALTER TABLE submission ADD COLUMN IF NOT EXISTS last_success_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN submission.priority IS '优先级（0-10，数字越大优先级越高）';
COMMENT ON COLUMN submission.health_status IS '健康状态: healthy/degraded/unhealthy';
COMMENT ON COLUMN submission.consecutive_failures IS '连续失败次数';
COMMENT ON COLUMN submission.last_scan_at IS '最后扫描时间';
COMMENT ON COLUMN submission.last_scan_error IS '最后扫描错误信息';
COMMENT ON COLUMN submission.last_success_at IS '最后成功扫描时间';

-- ============================================
-- 8. 创建索引优化查询
-- ============================================
CREATE INDEX IF NOT EXISTS idx_favorite_enabled_priority ON favorite(enabled, priority DESC) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_watch_later_enabled ON watch_later(enabled) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_collection_enabled_priority ON collection(enabled, priority DESC) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_submission_enabled_priority ON submission(enabled, priority DESC) WHERE enabled = TRUE;

-- ============================================
-- 9. 验证迁移结果
-- ============================================
DO $$
BEGIN
    RAISE NOTICE '============================================';
    RAISE NOTICE '数据库迁移完成！';
    RAISE NOTICE '============================================';
    RAISE NOTICE '新增表：';
    RAISE NOTICE '  - sync_logs';
    RAISE NOTICE '  - video_source_scans';
    RAISE NOTICE '  - scheduler_state';
    RAISE NOTICE '';
    RAISE NOTICE '扩展字段：';
    RAISE NOTICE '  - favorite, watch_later, collection, submission';
    RAISE NOTICE '    添加了: priority, health_status, consecutive_failures';
    RAISE NOTICE '           last_scan_at, last_scan_error, last_success_at';
    RAISE NOTICE '============================================';
END $$;
