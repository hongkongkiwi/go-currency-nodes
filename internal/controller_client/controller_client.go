/**
 * Controller Client Contains the gRPC server for calling functions on Nodes
 **/

package internal

import (
	"runtime"
	"strings"
)

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}
