package ssohandler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/server/skyerr"

	"github.com/skygeario/skygear-server/pkg/auth"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
)

func AttachLoginAuthURLHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/sso/{provider}/login_auth_url", &LoginAuthURLHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type LoginAuthURLHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f LoginAuthURLHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &LoginAuthURLHandler{}
	inject.DefaultInject(h, f.Dependency, request)
	vars := mux.Vars(request)
	h.ProviderName = vars["provider"]
	return handler.APIHandlerToHandler(h, h.TxContext)
}

func (f LoginAuthURLHandlerFactory) ProvideAuthzPolicy() authz.Policy {
	return authz.PolicyFunc(policy.DenyNoAccessKey)
}

// LoginAuthURLRequestPayload login handler request payload
type LoginAuthURLRequestPayload struct {
	Scope       []string               `json:"scope"`
	Options     map[string]interface{} `json:"options"`
	CallbackURL string                 `json:"callback_url"`
	RawUXMode   string                 `json:"ux_mode"`
	UXMode      sso.UXMode
}

// Validate request payload
func (p LoginAuthURLRequestPayload) Validate() error {
	if p.CallbackURL == "" {
		return skyerr.NewInvalidArgument("Callback url is required", []string{"callback_url"})
	}

	if p.UXMode == sso.Undefined {
		return skyerr.NewInvalidArgument("UX mode is required", []string{"ux_mode"})
	}

	return nil
}

// LoginAuthURLHandler returns the SSO auth url by provider.
//
// curl \
//   -X POST \
//   -H "Content-Type: application/json" \
//   -H "X-Skygear-Api-Key: API_KEY" \
//   -d @- \
//   http://localhost:3000/sso/<provider>/login_auth_url \
// <<EOF
// {
//     "scope": ["openid", "profile"],
//     "options": {
//       "prompt": "select_account"
//     },
//     callback_url: <url>,
//     ux_mode: <ux_mode>
// }
// EOF
//
// {
//     "result": "<auth_url>"
// }
type LoginAuthURLHandler struct {
	TxContext    db.TxContext           `dependency:"TxContext"`
	AuthContext  coreAuth.ContextGetter `dependency:"AuthContextGetter"`
	Provider     sso.Provider           `dependency:"SSOProvider"`
	ProviderName string
}

func (h LoginAuthURLHandler) WithTx() bool {
	return true
}

func (h LoginAuthURLHandler) DecodeRequest(request *http.Request) (handler.RequestPayload, error) {
	payload := LoginAuthURLRequestPayload{
		// avoid nil pointer
		Scope:   make([]string, 0),
		Options: make(sso.Options),
	}
	err := json.NewDecoder(request.Body).Decode(&payload)
	payload.UXMode = sso.UXModeFromString(payload.RawUXMode)

	return payload, err
}

func (h LoginAuthURLHandler) Handle(req interface{}) (resp interface{}, err error) {
	if h.Provider == nil {
		err = skyerr.NewInvalidArgument("Provider is not supported", []string{h.ProviderName})
		return
	}
	payload := req.(LoginAuthURLRequestPayload)
	params := sso.GetURLParams{
		Scope:       payload.Scope,
		Options:     payload.Options,
		CallbackURL: payload.CallbackURL,
		UXMode:      payload.UXMode,
		Action:      "login",
	}
	if h.AuthContext.AuthInfo() != nil {
		params.UserID = h.AuthContext.AuthInfo().ID
	}
	url, err := h.Provider.GetAuthURL(params)
	if err != nil {
		return
	}
	resp = url
	return
}