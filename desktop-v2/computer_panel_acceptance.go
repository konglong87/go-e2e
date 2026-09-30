//go:build computeracceptance && darwin && cgo

package main

/*
#cgo CFLAGS: -DGO_E2E_PANEL_ACCEPTANCE
*/
import "C"
