package session

import (
	"fmt"
	"sort"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

type MockProvider struct {
	Time    time.Provider
	counter int

	Sessions map[string]auth.Session
}

var _ Provider = &MockProvider{}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		Time:     &time.MockProvider{},
		Sessions: map[string]auth.Session{},
	}
}

func (p *MockProvider) Create(authnSess *auth.AuthnSession, beforeCreate func(*auth.Session) error) (*auth.Session, auth.SessionTokens, error) {
	now := p.Time.NowUTC()
	id := fmt.Sprintf("%s-%s-%d", authnSess.UserID, authnSess.PrincipalID, p.counter)
	sess := auth.Session{
		ID:                      id,
		ClientID:                authnSess.ClientID,
		UserID:                  authnSess.UserID,
		PrincipalID:             authnSess.PrincipalID,
		PrincipalType:           authnSess.PrincipalType,
		PrincipalUpdatedAt:      authnSess.PrincipalUpdatedAt,
		AuthenticatorID:         authnSess.AuthenticatorID,
		AuthenticatorType:       authnSess.AuthenticatorType,
		AuthenticatorOOBChannel: authnSess.AuthenticatorOOBChannel,
		AuthenticatorUpdatedAt:  authnSess.AuthenticatorUpdatedAt,
		CreatedAt:               now,
		AccessedAt:              now,
		AccessTokenHash:         "access-token-" + id,
		AccessTokenCreatedAt:    now,
	}
	tok := auth.SessionTokens{ID: id, AccessToken: "access-token-" + id}
	p.counter++

	if beforeCreate != nil {
		err := beforeCreate(&sess)
		if err != nil {
			return nil, tok, err
		}
	}

	p.Sessions[sess.ID] = sess

	return &sess, tok, nil
}

func (p *MockProvider) GetByToken(token string, kind auth.SessionTokenKind) (*auth.Session, error) {
	for _, s := range p.Sessions {
		var expectedToken string
		switch kind {
		case auth.SessionTokenKindAccessToken:
			expectedToken = s.AccessTokenHash
		case auth.SessionTokenKindRefreshToken:
			expectedToken = s.RefreshTokenHash
		default:
			continue
		}

		if expectedToken != token {
			continue
		}

		return &s, nil
	}
	return nil, ErrSessionNotFound
}

func (p *MockProvider) Get(id string) (*auth.Session, error) {
	session, ok := p.Sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return &session, nil
}

func (p *MockProvider) Access(s *auth.Session) error {
	s.AccessedAt = p.Time.NowUTC()
	p.Sessions[s.ID] = *s
	return nil
}

func (p *MockProvider) Invalidate(session *auth.Session) error {
	delete(p.Sessions, session.ID)
	return nil
}

func (p *MockProvider) InvalidateBatch(sessions []*auth.Session) error {
	for _, session := range sessions {
		delete(p.Sessions, session.ID)
	}
	return nil
}

func (p *MockProvider) InvalidateAll(userID string, sessionID string) error {
	for _, session := range p.Sessions {
		if session.UserID == userID && session.ID != sessionID {
			delete(p.Sessions, session.ID)
		}
	}
	return nil
}

func (p *MockProvider) List(userID string) (sessions []*auth.Session, err error) {
	for _, session := range p.Sessions {
		if session.UserID == userID {
			s := session
			sessions = append(sessions, &s)
		}
	}
	sort.Sort(sessionSlice(sessions))
	return
}

func (p *MockProvider) Refresh(session *auth.Session) (string, error) {
	session.AccessTokenHash = fmt.Sprintf("access-token-%s-%d", session.ID, p.counter)
	p.Sessions[session.ID] = *session
	p.counter++
	return session.AccessTokenHash, nil
}

func (p *MockProvider) UpdateMFA(sess *auth.Session, opts auth.AuthnSessionStepMFAOptions) error {
	now := p.Time.NowUTC()
	sess.AuthenticatorID = opts.AuthenticatorID
	sess.AuthenticatorType = opts.AuthenticatorType
	sess.AuthenticatorOOBChannel = opts.AuthenticatorOOBChannel
	sess.AuthenticatorUpdatedAt = &now
	p.Sessions[sess.ID] = *sess
	return nil
}

func (p *MockProvider) UpdatePrincipal(sess *auth.Session, principalID string) error {
	sess.PrincipalID = principalID
	p.Sessions[sess.ID] = *sess
	return nil
}
