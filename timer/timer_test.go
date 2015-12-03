/*
 * yubo@yubo.org
 * 2015-12-04
 */
package timer

import (
	"log"
	"testing"
	"time"
)

func timer_cb(data interface{}) {
	log.Println(data.(string))
}

func Test_timer(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	NewTimer(time.Second*2, timer_cb, "hello 2")
	NewTimer(time.Second, timer_cb, "hello 1")
	NewTimer(time.Second*3, timer_cb, "hello 3")
	time.Sleep(time.Second * 5)
}
