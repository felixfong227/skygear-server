package policy

import (
	"net/http"
	"testing"

	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDenyDisabledUser(t *testing.T) {
	Convey("Test DenyDisabledUser", t, func() {
		Convey("should not return error if auth context has no auth info", func() {
			req, _ := http.NewRequest("POST", "/", nil)
			ctx := MemoryContextGetter{}

			err := DenyDisabledUser(req, ctx)
			So(err, ShouldBeNil)
		})

		Convey("should return error if user is disabled", func() {
			req, _ := http.NewRequest("POST", "/", nil)
			ctx := MemoryContextGetter{
				mAuthInfo: &authinfo.AuthInfo{
					ID:       "ID",
					Disabled: true,
				},
			}

			err := DenyDisabledUser(req, ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("should pass if user is not disabled", func() {
			req, _ := http.NewRequest("POST", "/", nil)
			ctx := MemoryContextGetter{
				mAuthInfo: &authinfo.AuthInfo{
					ID:       "ID",
					Disabled: false,
				},
			}

			err := DenyDisabledUser(req, ctx)
			So(err, ShouldBeNil)
		})

	})
}
