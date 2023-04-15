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

// func mdString(md metadata.MD, key string) string {
// 	vals := md.Get(key)
// 	var str string
// 	if len(vals) > 0 {
// 		str = vals[0]
// 	}
// 	return str
// }

// md, ok := metadata.FromIncomingContext(ctx)
// var nodeUuid string
// if ok {
// 	nodeUuid = mdString(md, "nodeUuid")
// }
