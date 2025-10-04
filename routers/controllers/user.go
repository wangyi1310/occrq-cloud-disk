package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/wangyi1310/mycloud-disk/conf"
	"github.com/wangyi1310/mycloud-disk/models"
	"github.com/wangyi1310/mycloud-disk/pkg/session"
	"github.com/wangyi1310/mycloud-disk/serializer"
	"github.com/wangyi1310/mycloud-disk/services"
)

func UserLogin(c *gin.Context) {
	var login services.LoginUser
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(200, serializer.Err(serializer.CodeParamErr, "Failed to parse params", err))
		return
	}

	service := services.GetService[*services.UserService]("user")
	user, err := service.Login(&login)
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeCredentialInvalid, "Wrong email or password", err))
		return
	}

	session.SetSession(c, map[string]interface{}{
		"user_id": user.ID,
	})
	c.JSON(200, serializer.Response{Data: user})
}

func UserLogout(c *gin.Context) {
	session.DeleteSession(c)
	c.JSON(200, serializer.Response{})
}

// UserRegister 用户注册
func UserRegister(c *gin.Context) {
	// 注册逻辑
	var register services.RegisterUser
	err := c.BindJSON(&register)
	var res serializer.Response
	if err != nil {
		c.JSON(200, serializer.Err(serializer.CodeParamErr, "Failed to parse params", err))
		return
	}
	// 调用服务层注册方法
	service := services.GetService[*services.UserService]("user")
	res = service.Register(&register)
	c.JSON(200, res)
}

func UserActive(c *gin.Context) {
	uid, _ := c.Get("object_id")
	// 调用服务层激活方法
	service := services.GetService[*services.UserService]("user")
	res := service.Activate(&services.ActiveUser{
		Uid: uid,
	})
	c.JSON(200, res)
}

func UserInfo(c *gin.Context) {
	user, exist := c.Get("user")
	if !exist {
		c.JSON(200, serializer.Err(serializer.CodeCheckLogin, "User not login", nil))
		return
	}

	u := user.(*models.User)
	if u.Avatar == "" {
		u.Avatar = conf.DefaultAvatar
	}
	c.JSON(200, serializer.Response{Data: user.(*models.User)})
}
