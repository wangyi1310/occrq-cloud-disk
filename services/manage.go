package services

import "github.com/wangyi1310/mycloud-disk/conf"

var Management = make(map[string]interface{})

func init() {
	Management["user"] = &UserService{}
	service, err := NewLocalFileService(conf.SystemConfig.UploadDir, 1024*1024*100)
	if err != nil {
		return
	}
	Management["file"] = service
}

func GetService[T any](name string) T {
	return Management[name].(T)
}
