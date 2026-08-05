// Package health preserves the repository's health boundary while delegating
// implementation to the shared runtime process adapter.
package health

import platformruntime "github.com/fei613293175/ylven/backend/internal/platform/runtime"

type Service = platformruntime.Service
var Handler = platformruntime.Handler
