-- ============================================
-- 修复 GORM 列名映射问题
-- ============================================
-- 说明：将 GORM 自动生成的蛇形列名改为正确的列名
-- 影响表：video
-- 影响列：b_vid → bvid, pub_time → pubtime, fav_time → favtime, c_time → ctime
-- ============================================

-- 检查表结构
\d video;

-- 修复 video 表
DO $$
BEGIN
    -- 1. 如果存在 b_vid 列，重命名为 bvid
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'video' AND column_name = 'b_vid'
    ) THEN
        -- 如果同时存在 bvid 和 b_vid，先删除 b_vid
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'video' AND column_name = 'bvid'
        ) THEN
            ALTER TABLE video DROP COLUMN b_vid;
            RAISE NOTICE 'Dropped duplicate column b_vid';
        ELSE
            ALTER TABLE video RENAME COLUMN b_vid TO bvid;
            RAISE NOTICE 'Renamed b_vid to bvid';
        END IF;
    END IF;

    -- 2. 如果存在 pub_time 列，重命名为 pubtime
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'video' AND column_name = 'pub_time'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'video' AND column_name = 'pubtime'
        ) THEN
            ALTER TABLE video DROP COLUMN pub_time;
            RAISE NOTICE 'Dropped duplicate column pub_time';
        ELSE
            ALTER TABLE video RENAME COLUMN pub_time TO pubtime;
            RAISE NOTICE 'Renamed pub_time to pubtime';
        END IF;
    END IF;

    -- 3. 如果存在 fav_time 列，重命名为 favtime
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'video' AND column_name = 'fav_time'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'video' AND column_name = 'favtime'
        ) THEN
            ALTER TABLE video DROP COLUMN fav_time;
            RAISE NOTICE 'Dropped duplicate column fav_time';
        ELSE
            ALTER TABLE video RENAME COLUMN fav_time TO favtime;
            RAISE NOTICE 'Renamed fav_time to favtime';
        END IF;
    END IF;

    -- 4. 如果存在 c_time 列，重命名为 ctime
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'video' AND column_name = 'c_time'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = 'video' AND column_name = 'ctime'
        ) THEN
            ALTER TABLE video DROP COLUMN c_time;
            RAISE NOTICE 'Dropped duplicate column c_time';
        ELSE
            ALTER TABLE video RENAME COLUMN c_time TO ctime;
            RAISE NOTICE 'Renamed c_time to ctime';
        END IF;
    END IF;

    -- 5. 修复 tags 列类型（如果是 JSONB，改为 text[]）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'video' AND column_name = 'tags' AND data_type = 'jsonb'
    ) THEN
        -- 备份数据
        ALTER TABLE video RENAME COLUMN tags TO tags_old;

        -- 创建新列
        ALTER TABLE video ADD COLUMN tags text[];

        -- 迁移数据（将 JSONB 数组转为 text[]）
        UPDATE video SET tags = ARRAY(
            SELECT jsonb_array_elements_text(tags_old)
        ) WHERE tags_old IS NOT NULL;

        -- 删除旧列
        ALTER TABLE video DROP COLUMN tags_old;

        RAISE NOTICE 'Converted tags from JSONB to text[]';
    END IF;

END $$;

-- 验证修复结果
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'video'
AND column_name IN ('bvid', 'pubtime', 'favtime', 'ctime', 'tags', 'b_vid', 'pub_time', 'fav_time', 'c_time')
ORDER BY column_name;

-- 预期结果应该只有：
-- bvid     | character varying | NO
-- ctime    | timestamp with time zone | NO
-- favtime  | timestamp with time zone | NO
-- pubtime  | timestamp with time zone | NO
-- tags     | ARRAY | YES
