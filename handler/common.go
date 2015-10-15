package handler

import (
	"time"

	"github.com/oursky/skygear/uuid"
)

var (
	timeNowUTC = func() time.Time { return time.Now().UTC() }
	uuidNew    = uuid.New
	timeNow    = timeNowUTC
)
