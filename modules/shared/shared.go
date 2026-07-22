package shared

import (
	"go.uber.org/zap"

	"platform/ent"
)

var (
	Logger    *zap.SugaredLogger
	EntClient *ent.Client
)
