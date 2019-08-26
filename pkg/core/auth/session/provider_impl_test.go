package session

import (
	"testing"
	gotime "time"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	corerand "github.com/skygeario/skygear-server/pkg/core/rand"
	"github.com/skygeario/skygear-server/pkg/core/time"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProvider(t *testing.T) {
	Convey("Provider", t, func() {
		store := NewMockStore()

		timeProvider := &time.MockProvider{}
		initialTime := gotime.Date(2006, 1, 2, 15, 4, 5, 0, gotime.UTC)
		timeProvider.TimeNow = initialTime
		timeProvider.TimeNowUTC = initialTime

		var provider Provider = &providerImpl{
			store: store,
			time:  timeProvider,
			rand:  corerand.InsecureRand,
		}

		Convey("creating session", func() {
			Convey("should be successful", func() {
				session, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session, ShouldResemble, &auth.Session{
					ID:                   session.ID,
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session.AccessToken,
					AccessTokenCreatedAt: initialTime,
				})
				So(session.AccessToken, ShouldHaveLength, tokenLength+len(session.ID)+1)
			})

			Convey("should allow creating multiple sessions for same principal", func() {
				session1, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session1, ShouldResemble, &auth.Session{
					ID:                   session1.ID,
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session1.AccessToken,
					AccessTokenCreatedAt: initialTime,
				})

				session2, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session2, ShouldResemble, &auth.Session{
					ID:                   session2.ID,
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session2.AccessToken,
					AccessTokenCreatedAt: initialTime,
				})

				So(session1.ID, ShouldNotEqual, session2.ID)
			})
		})

		Convey("getting session", func() {
			fixtureSession := auth.Session{
				ID:                   "session-id",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "session-id.access-token",
				AccessTokenCreatedAt: initialTime,
			}
			store.Sessions["session-id"] = fixtureSession

			Convey("should be successful", func() {
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeNil)
				So(session, ShouldResemble, &fixtureSession)
			})

			Convey("should reject non-existant session", func() {
				session, err := provider.GetByToken("session-id-unknown.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})

			Convey("should reject incorrect token", func() {
				session, err := provider.GetByToken("session-id.incorrect-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)

				session, err = provider.GetByToken("invalid-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})
		})

		Convey("accessing session", func() {
			session := auth.Session{
				ID:                   "session-id",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "access-token",
				AccessTokenCreatedAt: initialTime,
			}
			timeProvider.AdvanceSeconds(100)
			timeNow := timeProvider.TimeNowUTC
			store.Sessions["session-id"] = session

			Convey("should be update accessed at time", func() {
				err := provider.Access(&session)
				So(err, ShouldBeNil)
				So(session.AccessedAt, ShouldEqual, timeNow)
			})
		})

		Convey("invalidating session", func() {
			store.Sessions["session-id"] = auth.Session{
				ID:                   "session-id",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "access-token",
				AccessTokenCreatedAt: initialTime,
			}

			Convey("should be successful", func() {
				err := provider.Invalidate("session-id")
				So(err, ShouldBeNil)
				So(store.Sessions, ShouldBeEmpty)
			})

			Convey("should be successful for non-existant sessions", func() {
				err := provider.Invalidate("session-id-unknown")
				So(err, ShouldBeNil)
				So(store.Sessions, ShouldNotBeEmpty)
			})
		})
	})
}
