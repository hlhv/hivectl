package main

import (
	"errors"
	"time"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

func main() {
	ParseArgs()
}

func needRoot() {
	uid := os.Getuid()
	if uid != 0 {
		fmt.Fprintln(os.Stderr, "ERR this utility must be run as root")
		os.Exit(1)
	}
}

func doStart() {
	pid, running := isCellRunning()
	if running {
		fmt.Fprintln(
			os.Stderr,
			"(i)", options.cell, "is already running with pid", pid)
		os.Exit(1)
	}

	pid, err := spawnCell()
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"ERR could not start", options.cell+":", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(options.pidfile, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"ERR could not write pidfile of",
			options.cell+":", err)
		os.Exit(1)
	}

	fmt.Println(pid)
}

func doStop() {
	pid, running := isCellRunning()
	if !running {
		fmt.Println("!!! cell", options.cell, "is not running")
		return
	}

	process, _ := os.FindProcess(pid)
	err := process.Kill()
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"ERR could not kill", options.cell+":", err)
		os.Exit(1)
	}

	// wait for the process to exit, with a timeout
	timeoutPoint := time.Now()
	for time.Since(timeoutPoint) < 16*time.Second {
		_, running = isCellRunning()
		if !running {
			return
		}
		
		time.Sleep(100*time.Millisecond)
	}
}

func doRestart() {
	doStop()
	doStart()
}

func doStatus() {
	uid, gid, err := getCellUid()
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"ERR cell", options.cell, "does not exist")
		os.Exit(1)
	}

	pid, running := isCellRunning()

	fmt.Println("(i) cell", options.cell+":")
	fmt.Println("- running:", running)
	if running {
		fmt.Println("- pid:    ", pid)
	}
	fmt.Println("- uid:    ", uid)
	fmt.Println("- gid:    ", gid)
}

func getCellPid() (pid int, err error) {
	content, err := ioutil.ReadFile(options.pidfile)
	if err != nil {
		return 0, err
	}
	pid, err = strconv.Atoi(string(content))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func isCellRunning() (pid int, running bool) {
	pid, err := getCellPid()
	if err != nil {
		return
	}
	
	directoryInfo, err := os.Stat("/proc/")
	if os.IsNotExist(err) || !directoryInfo.IsDir() {
		// if /proc/ does not exist, fallback to sending a signal
		needRoot()
		process, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			return
		}
	} else {
		// if /proc/ exists, see if the process's directory exists there
		_, err = os.Stat("/proc/" + strconv.Itoa(pid))
		if err != nil {
			return
		}
	}

	running = true
	return
}

func getCellUid() (uid uint32, gid uint32, err error) {
	user, err := user.Lookup(options.cell)
	if err != nil {
		return 0, 0, err
	}

	puid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return 0, 0, err
	}
	pgid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return 0, 0, err
	}
	return uint32(puid), uint32(pgid), nil
}

func spawnCell() (pid int, err error) {
	needRoot()
	uid, gid, err := getCellUid()
	if err != nil {
		return 0, errors.New("user does not exist")
	}

	var cred = &syscall.Credential{
		Uid:         uid,
		Gid:         gid,
		Groups:      []uint32{},
		NoSetGroups: false,
	}

	// the Noctty flag is used to detach the process from parent tty
	var sysproc = &syscall.SysProcAttr{
		Credential: cred,
		Noctty:     true,
	}
	var attr = os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			nil,
			nil,
		},
		Sys: sysproc,
	}

	path, err := exec.LookPath(options.cell)
	if err != nil {
		return 0, errors.New("executable does not exist")
	}

	process, err := os.StartProcess(
		path, []string{
			path,
			"-L", "/var/log/hlhv/" + options.cellName,
		}, &attr)
	if err != nil {
		return 0, err
	}

	pid = process.Pid
	// process.Release actually detatches the process and init adopts it.
	return pid, process.Release()
}
