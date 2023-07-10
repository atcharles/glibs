package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/boot"
	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/util"
)

var startCmd = &cobra.Command{
	Use: "start",
	Run: func(*cobra.Command, []string) {
		if daemon {
			killProcess()
			cmdStart()
			return
		}
		if err := writePid(); err != nil {
			log.Fatalf("写入pid文件失败: %s", err.Error())
		}

		if StartFunc != nil {
			StartFunc()
		}
	},
}

var StartFunc = boot.Start

func getPidFile() string {
	pf1 := filepath.Join(util.RootDir(), "data", fmt.Sprintf("%s.pid", config.C.AppName()))
	_ = os.MkdirAll(filepath.Dir(pf1), 0755)
	return pf1
}

func killWithPidFile() {
	pidFile := getPidFile()
	if _, e := os.Stat(pidFile); e != nil {
		return
	}
	pid, e := os.ReadFile(pidFile)
	if e != nil {
		return
	}
	if e = os.Remove(pidFile); e != nil {
		return
	}
	KillProcessByPID(string(pid))
}

func killProcess() {
	if util.AddressCanListen(config.Viper().GetString("server.host")) {
		return
	}
	_ = KillProcess(fmt.Sprintf("%s.out -x", config.C.AppName()))
	_ = KillProcess(fmt.Sprintf("%s -x", config.C.AppName()))

	killWithPidFile()
	for {
		if util.AddressCanListen(config.Viper().GetString("server.host")) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for {
		if util.AddressCanListen(config.Viper().GetString("server.pprof_port")) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Stop daemon success")
}

func writePid() (err error) {
	var pidFile = getPidFile()
	err = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	return
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
