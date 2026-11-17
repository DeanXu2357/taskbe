//go:build go1.21

package logx_test

import (
	"log/slog"

	"github.com/DeanXu2357/taskbe/internal/logx"
)

var _ logx.Logger = (*slog.Logger)(nil)
