package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/boot"
	"github.com/atcharles/glibs/util"
)

var (
	BeforeStartFunc func()
	StartFunc       = boot.Start
	startCmd        = &cobra.Command{
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
			if err := writePid(); err != nil {
				log.Fatalf("写入pid文件失败: %s", err.Error())
			}

			if StartFunc != nil {
				StartFunc()
			}
		},
	}
)

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

func getPidFile() string {
	config := GetConfigFunc()
	pf1 := filepath.Join(util.RootDir(), "data", fmt.Sprintf("%s.pid", config.APPName))
	_ = os.MkdirAll(filepath.Dir(pf1), 0755)
	return pf1
}

func killProcess() {
	config := GetConfigFunc()
	if util.AddressCanListen(config.ServerHost) {
		return
	}
	_ = KillProcess(fmt.Sprintf("%s.out -x", config.APPName))
	_ = KillProcess(fmt.Sprintf("%s.run -x", config.APPName))
	_ = KillProcess(fmt.Sprintf("%s -x", config.APPName))

	stringArrHost := strings.Split(config.ServerHost, ":")
	if len(stringArrHost) == 2 {
		port, e := strconv.Atoi(stringArrHost[1])
		if e == nil {
			_ = KillProcessWithPort(uint32(port))
		}
	}

	killWithPidFile()
	for {
		if util.AddressCanListen(config.ServerHost) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for {
		if util.AddressCanListen(config.ServerPProfPort) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Stop daemon success")
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

func writePid() (err error) {
	var pidFile = getPidFile()
	err = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	return
}
