package models

import (
	// "context"
	// "sort"
	// "strings"

	"errors"

	"github.com/fatih/color"
	"github.com/wangyi1310/mycloud-disk/conf"
	"github.com/wangyi1310/mycloud-disk/pkg/cache"
	"github.com/wangyi1310/mycloud-disk/pkg/log"
	"github.com/wangyi1310/mycloud-disk/pkg/util"
	"gorm.io/gorm"
)

// 是否需要迁移
func needMigration() bool {
	var setting Setting
	return DB.Where("name = ?", "db_version_"+conf.RequiredDBVersion).First(&setting).Error != nil
}

// 执行数据迁移
func migration() {
	// 确认是否需要执行迁移
	if !needMigration() {
		log.Log().Info("Database version fulfilled, skip schema migration.")
		return

	}

	log.Log().Info("Start initializing database schema...")

	// 清除所有缓存
	if instance, ok := cache.Store.(*cache.RedisStore); ok {
		instance.DeleteAll()
	}

	// 自动迁移模式
	if conf.DatabaseConfig.Type == "mysql" {
		DB = DB.Set("gorm:table_options", "ENGINE=InnoDB")
	}

	err := DB.AutoMigrate(&User{}, &Setting{})
	if err != nil {
		log.Log().Error("Database migration failed. error: " + err.Error())
		return
	}

	// 创建初始存储策略
	addDefaultPolicy()

	// 创建初始用户组
	addDefaultGroups()

	// 创建初始管理员账户
	addDefaultUser()

	// 向设置数据表添加初始设置
	addDefaultSettings()

	// 执行数据库升级脚本
	execUpgradeScripts()

	log.Log().Info("Finish initializing database schema.")

}

func addDefaultPolicy() {

	// _, err := GetPolicyByID(uint(1))
	// // 未找到初始存储策略时，则创建
	// if gorm.IsRecordNotFoundError(err) {
	// 	defaultPolicy := Policy{
	// 		Name:               "Default storage policy",
	// 		Type:               "local",
	// 		MaxSize:            0,
	// 		AutoRename:         true,
	// 		DirNameRule:        "uploads/{uid}/{path}",
	// 		FileNameRule:       "{uid}_{randomkey8}_{originname}",
	// 		IsOriginLinkEnable: false,
	// 		OptionsSerialized: PolicyOption{
	// 			ChunkSize: 25 << 20, // 25MB
	// 		},
	// 	}
	// 	if err := DB.Create(&defaultPolicy).Error; err != nil {
	// 		util.Log().Panic("Failed to create default storage policy: %s", err)
	// 	}
	// }
}

func addDefaultSettings() {
	for _, value := range defaultSettings {
		DB.Where(Setting{Name: value.Name}).Create(&value)
	}
}

func addDefaultGroups() {
}

func addDefaultUser() {
	tx := DB.Begin()
	result := tx.Set("gorm:auto_preload", true).Where("id", 1).First(&User{})
	err := result.Error
	
	password := util.RandStringRunes(8)
	if err != nil {
		// 未找到初始用户时，则创建
		if errors.Is(err, gorm.ErrRecordNotFound) {
			defaultUser := NewUser()
			defaultUser.Email = "admin@cloudreve.org"
			defaultUser.Nick = "admin"
			defaultUser.Status = Active
			defaultUser.GroupID = 1
			err = defaultUser.SetPassword(password)
			if err != nil {
				log.Log().Panic("Failed to create password: %s", err)
			}

			if err = tx.Set("gorm:auto_preload", true).Create(&defaultUser).Error; err != nil {
				log.Log().Panic("Failed to create initial root user: %+v", err)
			}

			c := color.New(color.FgWhite).Add(color.BgBlack).Add(color.Bold)
			log.Log().Info("Admin user name: " + c.Sprint("admin@cloudreve.org"))
			log.Log().Info("Admin password: " + c.Sprint(password))
			tx.Commit()
		}
	}
}

func execUpgradeScripts() {
	// s := invoker.ListPrefix("UpgradeTo")
	// versions := make([]*version.Version, len(s))
	// for i, raw := range s {
	// 	v, _ := version.NewVersion(strings.TrimPrefix(raw, "UpgradeTo"))
	// 	versions[i] = v
	// }
	// sort.Sort(version.Collection(versions))

	// for i := 0; i < len(versions); i++ {
	// 	invoker.RunDBScript("UpgradeTo"+versions[i].String(), context.Background())
	// }
}
