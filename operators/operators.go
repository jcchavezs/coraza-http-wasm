// Source: https://github.com/corazawaf/coraza-proxy-wasm/tree/main/internal/operators

//go:build tinygo

package operators

import (
	wasilibs "github.com/corazawaf/coraza-wasilibs"
)

func Register() {
	wasilibs.RegisterRX()
	wasilibs.RegisterPM()
	wasilibs.RegisterSQLi()
	wasilibs.RegisterXSS()
}
