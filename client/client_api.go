package client

import (
	"github.com/skycoin/cxo/gui"
	"time"
)

// clientApi is used to terminate client from command line tool;
// it's danger because every one can send http-reqest; there aren't
// any protection
type clientApi struct {
	*Client
}

func (api clientApi) Register(router *gui.Router) {
	router.DELETE("/daemon/close", api.Close)
}

func (api clientApi) Close(ctx *gui.Context) (err error) {
	if api.config.RemoteTermination == true {
		defer func() {
			// make sure that this response will be delivered to cli
			time.Sleep(300 * time.Millisecond)
			// terminate
			close(api.quit)
		}()
		err = ctx.JSON(200, map[string]interface{}{
			"detail": "terminating...",
			"status": 200,
		})
	} else {
		err = ctx.JSON(200, map[string]interface{}{
			"detail": "not allowed",
			"status": 403,
		})
	}
	return
}
