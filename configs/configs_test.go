package configs

import (
	"fmt"
	"github.com/legenove/cocore"
	"testing"
)

func init() {
	cocore.InitApp(true, "", "$GOPATH/src/go_svc/ocls_tasks/conf", "")
}

func TestGetTaskConf(t *testing.T) {
	fmt.Println(GetTaskConf())
}
