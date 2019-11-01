package httpsigning

import (
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTPSigning(t *testing.T) {
	Convey("HTTP Signing", t, func() {
		Convey("Sign and Verify", func() {
			signTime := time.Date(2019, 10, 11, 3, 4, 5, 0, time.UTC)
			verifyTime := time.Date(2019, 10, 11, 3, 4, 5, 0, time.UTC)
			r, _ := http.NewRequest("GET", "https://example.com/", nil)
			key := []byte("secret")
			Sign(key, r, signTime, 5)

			err := Verify(key, r, verifyTime)
			So(err, ShouldBeNil)
		})

		Convey("Invalid signature", func() {
			signTime := time.Date(2019, 10, 11, 3, 4, 5, 0, time.UTC)
			verifyTime := time.Date(2019, 10, 11, 3, 4, 5, 0, time.UTC)
			r, _ := http.NewRequest("GET", "https://example.com/", nil)
			key := []byte("secret")
			Sign(key, r, signTime, 5)

			q := r.URL.Query()
			q.Set("x-skygear-signature", q.Get("x-skygear-signature")+"1")
			r.URL.RawQuery = q.Encode()

			err := Verify(key, r, verifyTime)
			So(err, ShouldBeError, ErrInvalidSignature)
		})

		Convey("Expired signature", func() {
			signTime := time.Date(2019, 10, 11, 3, 4, 5, 0, time.UTC)
			verifyTime := time.Date(2019, 10, 11, 3, 4, 11, 0, time.UTC)
			r, _ := http.NewRequest("GET", "https://example.com/", nil)
			key := []byte("secret")
			Sign(key, r, signTime, 5)

			err := Verify(key, r, verifyTime)
			So(err, ShouldBeError, ErrExpiredSignature)
		})
	})
}