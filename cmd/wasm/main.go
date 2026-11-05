//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/balkashynov/grantline/internal/compare"
)

func main() {
	analyze := js.FuncOf(func(this js.Value, args []js.Value) any {
		result := map[string]any{}
		if len(args) != 1 || args[0].Type() != js.TypeString {
			result["error"] = "Expected one JSON string"
		} else {
			in, err := compare.Decode([]byte(args[0].String()))
			if err != nil {
				result["error"] = err.Error()
			} else {
				report, err := compare.Analyze(in)
				if err != nil {
					result["error"] = err.Error()
				} else {
					result["report"] = report
				}
			}
		}
		b, _ := json.Marshal(result)
		return string(b)
	})
	js.Global().Set("grantlineAnalyze", analyze)
	select {}
}
