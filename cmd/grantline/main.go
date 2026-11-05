package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/balkashynov/grantline/internal/compare"
	tf "github.com/balkashynov/grantline/internal/terraform"
)

func emit(v any) error { e := json.NewEncoder(os.Stdout); e.SetIndent("", "  "); return e.Encode(v) }
func read(path string) ([]byte, error) {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, e := os.Open(path)
		if e != nil {
			return nil, fmt.Errorf("could not open input file")
		}
		defer f.Close()
		r = f
	}
	b, e := io.ReadAll(io.LimitReader(r, compare.MaxInput+1))
	if e != nil {
		return nil, fmt.Errorf("could not read input")
	}
	if len(b) > compare.MaxInput {
		return nil, fmt.Errorf("input exceeds 16 MiB")
	}
	return b, nil
}
func run() int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: grantline compare -input comparison.json\n       grantline import -state state.json -from ADDRESS -to ADDRESS -require-ip-list true|false|unknown\nimport emits a sanitized comparison. compare exits 0=equal in scope, 2=differences, 3=incomplete, 1=invalid.")
		return 1
	}
	mode := os.Args[1]
	f := flag.NewFlagSet(mode, flag.ContinueOnError)
	input := f.String("input", "-", "comparison JSON path or - for stdin")
	state := f.String("state", "-", "terraform show -json STATE path or -")
	from := f.String("from", "", "organization API-key resource address")
	to := f.String("to", "", "organization service-account resource address")
	policy := f.String("require-ip-list", "unknown", "organization setting: true, false, unknown")
	if err := f.Parse(os.Args[2:]); err != nil {
		return 1
	}
	if f.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		return 1
	}
	var in compare.Input
	var err error
	switch mode {
	case "import":
		var required *bool
		if *policy != "unknown" {
			v, e := strconv.ParseBool(*policy)
			if e != nil {
				fmt.Fprintln(os.Stderr, "require-ip-list must be true, false, or unknown")
				return 1
			}
			required = &v
		}
		var data []byte
		data, err = read(*state)
		if err == nil {
			in, err = tf.Import(data, *from, *to, required)
		}
		if err == nil {
			err = emit(in)
		}
	case "compare":
		var data []byte
		data, err = read(*input)
		if err == nil {
			in, err = compare.Decode(data)
		}
		if err == nil {
			var report compare.Report
			report, err = compare.Analyze(in)
			if err == nil {
				if err = emit(report); err == nil {
					switch report.Status {
					case "differences_found":
						return 2
					case "incomplete":
						return 3
					}
				}
			}
		}
	default:
		err = fmt.Errorf("unknown command; use import or compare")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
func main() { os.Exit(run()) }
