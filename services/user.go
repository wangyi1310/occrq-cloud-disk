package services

import (
	"net/url"

	"github.com/wangyi1310/mycloud-disk/models"
	"github.com/wangyi1310/mycloud-disk/pkg/auth"
	"github.com/wangyi1310/mycloud-disk/pkg/email"
	"github.com/wangyi1310/mycloud-disk/pkg/hashid"
	"github.com/wangyi1310/mycloud-disk/serializer"
)

type RegisterUser struct {
	Name     string `json:"name" binding:"required,min=2,max=30"`
	Password string `json:"password" binding:"required,min=8,max=40"`
	Email    string `json:"email" binding:"required,email"`
}

type ActiveUser struct {
	Uid any
}

type UserService struct {
}

type LoginUser struct {
	//TODO 细致调整验证规则
	UserName string `form:"userName" json:"userName" binding:"required"`
	Password string `form:"Password" json:"Password" binding:"required,min=4,max=64"`
}

func (u *UserService) Login(login *LoginUser) (*models.User, error) {
	expectedUser, err := models.GetUserByEmail(login.UserName)
	if err != nil {
		return nil, err
	}

	if authOK, _ := expectedUser.CheckPassword(login.Password); !authOK {
		return nil, err
	}
	return &expectedUser, nil

}
func (u *UserService) Register(r *RegisterUser) serializer.Response {
	options := models.GetSettingByNames("email_active")
	enableEmailActive := models.IsTrueVal(options["email_active"])
	user := models.NewUser()
	if enableEmailActive {
		user.Status = models.NotActivicated
	}
	user.Nick = r.Name
	user.Email = r.Email
	user.SetPassword(r.Password)
	userNotActivated := false
	if err := models.DB.Create(&user).Error; err != nil {
		// 注册失败
		dbUser, _ := models.GetUserByEmail(r.Email)
		if dbUser.Status == models.NotActivicated {
			userNotActivated = true
			user = dbUser
		} else {
			return serializer.Err(serializer.CodeEmailExisted, "用户已存在", err)
		}
	}
	if enableEmailActive {
		base := models.GetSiteURL()
		userID := hashid.HashID(user.ID, hashid.UserID)
		controller, _ := url.Parse("/api/v3/user/activate?id=" + userID)
		activateURL, err := auth.SignURI(auth.GetDefaultAuth(), base.ResolveReference(controller).String(), 86400)
		if err != nil {
			return serializer.Err(serializer.CodeEncryptError, "Failed to sign the activation link", err)
		}

		// 取得签名
		credential := activateURL.Query().Get("sign")
		controller, _ = url.Parse("/api/v3/user/activate")
		finalURL := base.ResolveReference(controller)
		queries := finalURL.Query()
		queries.Add("id", userID)
		queries.Add("sign", credential)
		finalURL.RawQuery = queries.Encode()
		// 发送邮件
		// 返送激活邮件
		title, body := email.NewActivationEmail(user.Email,
			finalURL.String(),
		)
		if err := email.Send(user.Email, title, body); err != nil {
			return serializer.Err(serializer.CodeFailedSendEmail, "Failed to send activation email", err)
		}
		if userNotActivated == true {
			//原本在上面要抛出的DBErr，放来这边抛出
			return serializer.Err(serializer.CodeEmailSent, "User is not activated, activation email has been resent", nil)
		} else {
			return serializer.Response{Code: 203, Msg: "Success to send activation email, please check your email!"}
		}

	}
	return serializer.Response{Code: 200}
}

func (u *UserService) Activate(a *ActiveUser) serializer.Response {
	// 查找待激活用户
	uid := a.Uid
	user, err := models.GetUserByID(uid.(uint))
	if err != nil {
		return serializer.Err(serializer.CodeUserNotFound, "User not fount", err)
	}

	// 检查状态
	if user.Status != models.NotActivicated {
		return serializer.Err(serializer.CodeUserCannotActivate, "This user cannot be activated", nil)
	}

	// 激活用户
	user.SetStatus(models.Active)
	return serializer.Response{Data: user.Email}
}
