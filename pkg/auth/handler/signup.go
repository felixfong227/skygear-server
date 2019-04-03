package handler

import (
	"encoding/json"
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth/task"

	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/provider/anonymous"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/provider/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"

	"github.com/skygeario/skygear-server/pkg/auth"
	authAudit "github.com/skygeario/skygear-server/pkg/auth/dependency/audit"
	"github.com/skygeario/skygear-server/pkg/auth/response"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/audit"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authtoken"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/skydb"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

var ErrUserDuplicated = skyerr.NewError(skyerr.Duplicated, "user duplicated")

func AttachSignupHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/signup", &SignupHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type SignupHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f SignupHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &SignupHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	h.AuditTrail = h.AuditTrail.WithRequest(request)
	return handler.APIHandlerToHandler(h, h.TxContext)
}

func (f SignupHandlerFactory) ProvideAuthzPolicy() authz.Policy {
	return authz.PolicyFunc(policy.DenyNoAccessKey)
}

type SignupRequestPayload struct {
	LoginIDs   map[string]string      `json:"login_ids"`
	Password   string                 `json:"password"`
	RawProfile map[string]interface{} `json:"profile"`
}

func (p SignupRequestPayload) Validate() error {
	if p.isAnonymous() {
		//no validation logic for anonymous sign up
	} else {
		if len(p.LoginIDs) == 0 {
			return skyerr.NewInvalidArgument("empty login_id", []string{"login_id"})
		}

		if p.Password == "" {
			return skyerr.NewInvalidArgument("empty password", []string{"password"})
		}
	}

	return nil
}

func (p SignupRequestPayload) isAnonymous() bool {
	return len(p.LoginIDs) == 0 && p.Password == ""
}

// SignupHandler handles signup request
type SignupHandler struct {
	PasswordChecker        *authAudit.PasswordChecker `dependency:"PasswordChecker"`
	UserProfileStore       userprofile.Store          `dependency:"UserProfileStore"`
	TokenStore             authtoken.Store            `dependency:"TokenStore"`
	AuthInfoStore          authinfo.Store             `dependency:"AuthInfoStore"`
	PasswordAuthProvider   password.Provider          `dependency:"PasswordAuthProvider"`
	AnonymousAuthProvider  anonymous.Provider         `dependency:"AnonymousAuthProvider"`
	AuditTrail             audit.Trail                `dependency:"AuditTrail"`
	WelcomeEmailEnabled    bool                       `dependency:"WelcomeEmailEnabled"`
	AutoSendUserVerifyCode bool                       `dependency:"AutoSendUserVerifyCodeOnSignup"`
	UserVerifyKeys         []string                   `dependency:"UserVerifyKeys"`
	VerifyCodeStore        userverify.Store           `dependency:"VerifyCodeStore"`
	TxContext              db.TxContext               `dependency:"TxContext"`
	Logger                 *logrus.Entry              `dependency:"HandlerLogger"`
	TaskQueue              async.Queue                `dependency:"AsyncTaskQueue"`
}

func (h SignupHandler) WithTx() bool {
	return true
}

func (h SignupHandler) DecodeRequest(request *http.Request) (handler.RequestPayload, error) {
	payload := SignupRequestPayload{}
	err := json.NewDecoder(request.Body).Decode(&payload)
	return payload, err
}

func (h SignupHandler) Handle(req interface{}) (resp interface{}, err error) {
	payload := req.(SignupRequestPayload)

	err = h.verifyPayload(payload)
	if err != nil {
		return
	}

	now := timeNow()
	info := authinfo.NewAuthInfo()
	info.LastLoginAt = &now

	// Create AuthInfo
	if err = h.AuthInfoStore.CreateAuth(&info); err != nil {
		if err == skydb.ErrUserDuplicated {
			err = ErrUserDuplicated
			return
		}

		// TODO:
		// return proper error
		err = skyerr.NewError(skyerr.UnexpectedError, "Unable to save auth info")
		return
	}

	// Create Profile
	var userProfile userprofile.UserProfile
	if userProfile, err = h.UserProfileStore.CreateUserProfile(info.ID, &info, payload.RawProfile); err != nil {
		// TODO:
		// return proper error
		err = skyerr.NewError(skyerr.UnexpectedError, "Unable to save user profile")
		return
	}

	// Create Principal
	if err = h.createPrincipal(payload, info); err != nil {
		return
	}

	// Create auth token
	tkn, err := h.TokenStore.NewToken(info.ID)
	if err != nil {
		panic(err)
	}

	if err = h.TokenStore.Put(&tkn); err != nil {
		panic(err)
	}

	// Initialise verify state
	info.VerifyInfo = map[string]bool{}
	for _, key := range h.UserVerifyKeys {
		info.VerifyInfo[key] = false
	}

	authResp := response.NewAuthResponse(info, userProfile, tkn.AccessToken)
	authResp.LoginIDs = payload.LoginIDs

	// Populate the activity time to user
	info.LastSeenAt = &now
	if err = h.AuthInfoStore.UpdateAuth(&info); err != nil {
		err = skyerr.MakeError(err)
		return
	}

	h.AuditTrail.Log(audit.Entry{
		AuthID: info.ID,
		Event:  audit.EventSignup,
	})

	if h.WelcomeEmailEnabled {
		h.sendWelcomeEmail(userProfile.MergeLoginIDs(payload.LoginIDs))
	}

	if h.AutoSendUserVerifyCode {
		h.sendUserVerifyRequest(userProfile.MergeLoginIDs(payload.LoginIDs))
	}

	return authResp, nil
}

func (h SignupHandler) verifyPayload(payload SignupRequestPayload) (err error) {
	if payload.isAnonymous() {
		return
	}

	if valid := h.PasswordAuthProvider.IsLoginIDValid(payload.LoginIDs); !valid {
		err = skyerr.NewInvalidArgument("invalid login_ids", []string{"login_ids"})
		return
	}

	// validate password
	err = h.PasswordChecker.ValidatePassword(authAudit.ValidatePasswordPayload{
		PlainPassword: payload.Password,
	})

	return
}

func (h SignupHandler) createPrincipal(payload SignupRequestPayload, authInfo authinfo.AuthInfo) (err error) {
	if !payload.isAnonymous() {
		err = h.PasswordAuthProvider.CreatePrincipalsByLoginID(authInfo.ID, payload.Password, payload.LoginIDs)
		if err == skydb.ErrUserDuplicated {
			err = ErrUserDuplicated
		}
	} else {
		principal := anonymous.NewPrincipal()
		principal.UserID = authInfo.ID

		err = h.AnonymousAuthProvider.CreatePrincipal(principal)
	}

	return
}

func (h SignupHandler) sendWelcomeEmail(userProfile userprofile.UserProfile) {
	if email, ok := userProfile.Data["email"].(string); ok {
		h.TaskQueue.Enqueue(task.WelcomeEmailSendTaskName, task.WelcomeEmailSendTaskParam{
			Email:       email,
			UserProfile: userProfile,
		}, nil)
	}
}

func (h SignupHandler) sendUserVerifyRequest(userProfile userprofile.UserProfile) {
	for _, key := range h.UserVerifyKeys {
		if value, ok := userProfile.Data[key].(string); ok {
			h.TaskQueue.Enqueue(task.VerifyCodeSendTaskName, task.VerifyCodeSendTaskParam{
				Key:         key,
				Value:       value,
				UserProfile: userProfile,
			}, nil)
		}
	}
}
