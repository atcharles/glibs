package cmd2

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/util"
)

var BeforeStartFunc func()
var StartFunc func()
var startCmd = &cobra.Command{
	Use: "start",
	Run: func(*cobra.Command, []string) {
		if BeforeStartFunc != nil {
			BeforeStartFunc()
		}
		if daemon {
			killProcess()
			cmdStart()
			return
		}
		if StartFunc != nil {
			StartFunc()
		}
	},
}

func cmdStart() {
	file1name := filepath.Join(util.RootDir(), "logs", "runtime.log")
	_ = os.MkdirAll(filepath.Dir(file1name), 0755)
	file1, _ := os.OpenFile(file1name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	ec := os.Args[0]
	str := fmt.Sprintf("%s -x", ec)
	for _, v := range os.Args[1:] {
		str += fmt.Sprintf(" %s", v)
	}
	str = strings.Replace(str, "-d", "", 1)
	cmd := StdExec(str)
	cmd.Stdout = file1
	cmd.Stderr = file1
	_ = cmd.Start()
	log.Println("Start daemon success")
}

func killProcess() {
	ec := os.Args[0]
	process := fmt.Sprintf("%s -x", ec)
	if !ProcessIsRunning(process) {
		return
	}
	_ = KillProcess(process)
	for {
		if !ProcessIsRunning(process) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Stop daemon success")
}
