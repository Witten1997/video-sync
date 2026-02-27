package database

import (
	"fmt"
	"strings"
	"time"

	"bili-download/internal/config"
	"bili-download/internal/database/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db *gorm.DB
)

// Connect 连接数据库
func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接数据库
	var err error
	db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.GetConnMaxLifetime())

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping 数据库失败: %w", err)
	}

	return db, nil
}

// Get 获取数据库实例
func Get() *gorm.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 先执行手动迁移，处理特殊情况
	if err := manualMigrations(db); err != nil {
		return err
	}

	// 删除 video/page 表上所有外键约束（改为应用层管理关联）
	dropAllForeignKeys(db, "video")
	dropAllForeignKeys(db, "page")

	// 依次迁移每个模型，单独处理错误
	allModels := []interface{}{
		&models.Video{},
		&models.Page{},
		&models.Favorite{},
		&models.WatchLater{},
		&models.Collection{},
		&models.Submission{},
		&models.DownloadRecord{},
		&models.User{},
	}

	// 禁用外键约束迁移，避免级联关联表时触发约束错误
	// 迁移 session 使用 Silent 日志，避免 GORM 内部 DROP CONSTRAINT 失败时输出 ERROR 日志
	migrateDB := db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	migrateDB.Config.DisableForeignKeyConstraintWhenMigrating = true
	migrator := migrateDB.Migrator()

	for _, model := range allModels {
		if err := migrator.AutoMigrate(model); err != nil {
			if isConstraintNotExistsError(err) {
				// GORM 尝试 DROP 不存在的约束导致失败，用原生 SQL 补建缺失列
				if migrator.HasTable(model) {
					addMissingColumns(db, migrator, model)
				} else {
					if createErr := migrator.CreateTable(model); createErr != nil {
						return fmt.Errorf("AutoMigrate %T failed: %v, CreateTable also failed: %v", model, err, createErr)
					}
				}
				continue
			}
			return err
		}
	}

	return nil
}

// dropAllForeignKeys 动态查询并删除指定表上所有外键约束
func dropAllForeignKeys(db *gorm.DB, tableName string) {
	var constraints []string
	db.Raw(`
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE table_name = ? AND constraint_type = 'FOREIGN KEY'
	`, tableName).Scan(&constraints)

	for _, c := range constraints {
		db.Exec(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", tableName, c))
	}
}

// isConstraintNotExistsError 检查是否是约束不存在的错误
func isConstraintNotExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "does not exist") && strings.Contains(errStr, "constraint")
}

// addMissingColumns 对比模型字段，用原生 SQL 补建数据库中缺失的列
func addMissingColumns(db *gorm.DB, migrator gorm.Migrator, model interface{}) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return
	}
	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		if !migrator.HasColumn(model, field.DBName) {
			if err := migrator.AddColumn(model, field.DBName); err != nil {
				fmt.Printf("[Migrate] AddColumn %s.%s failed: %v\n", stmt.Schema.Table, field.DBName, err)
			}
		}
	}
}

// manualMigrations 手动处理特殊的数据库迁移
func manualMigrations(db *gorm.DB) error {
	// 检查 video 表是否存在
	if db.Migrator().HasTable(&models.Video{}) {
		// 删除 bvid 的唯一约束，允许同一视频在不同视频源中存在
		if err := db.Exec(`ALTER TABLE video DROP CONSTRAINT IF EXISTS video_bvid_key`).Error; err != nil {
			return fmt.Errorf("删除 video_bvid_key 约束失败: %w", err)
		}

		// 检查 tags 字段的类型
		var dataType string
		err := db.Raw(`
			SELECT data_type
			FROM information_schema.columns
			WHERE table_name = 'video'
			AND column_name = 'tags'
		`).Scan(&dataType).Error

		if err != nil {
			return fmt.Errorf("检查 tags 字段类型失败: %w", err)
		}

		// 如果是 jsonb 或 json 类型，需要先转换为 text[]
		if dataType == "jsonb" || dataType == "json" {
			// 先删除可能存在的 GIN 索引
			if err := db.Exec(`DROP INDEX IF EXISTS idx_video_tags`).Error; err != nil {
				return fmt.Errorf("删除 tags 索引失败: %w", err)
			}

			// 创建临时函数来转换 jsonb 到 text[]
			err = db.Exec(`
				CREATE OR REPLACE FUNCTION jsonb_to_text_array(jsonb)
				RETURNS text[] AS $$
					SELECT CASE
						WHEN $1 IS NULL THEN NULL
						WHEN jsonb_typeof($1) = 'array' THEN
							ARRAY(SELECT jsonb_array_elements_text($1))
						ELSE ARRAY[]::text[]
					END;
				$$ LANGUAGE SQL IMMUTABLE;
			`).Error

			if err != nil {
				return fmt.Errorf("创建转换函数失败: %w", err)
			}

			// 使用函数转换字段类型
			err = db.Exec(`
				ALTER TABLE video
				ALTER COLUMN tags TYPE text[]
				USING jsonb_to_text_array(tags)
			`).Error

			if err != nil {
				return fmt.Errorf("转换 tags 字段类型失败: %w", err)
			}

			// 删除临时函数
			_ = db.Exec(`DROP FUNCTION IF EXISTS jsonb_to_text_array(jsonb)`).Error
		}
	}

	return nil
}
