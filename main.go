package main

import "errors"
import "time"
import "fmt"
import "io/ioutil"
import "os"
import "os/exec"
import "os/user"
import "strconv"
import "syscall"

func main() {
	parseArgs()
}

// needRoot halts the program and displays an error if it is not being run as
// root. This should be called whenever an operation takes place that requires
// root privelages.
func needRoot() {
	uid := os.Getuid()
	if uid != 0 {
		fmt.Fprintln(os.Stderr, "ERR this utility must be run as root")
		os.Exit(1)
	}
}

// doStart performs a cell start operation.
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

// doStart performs a cell stop operation. It waits until the cell has stopped
// or a timeout of 16 seconds is reached before returning.
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

// doRestart performs a cell restart operation.
func doRestart() {
	doStop()
	doStart()
}

// doStatus performs a cell status get operation.
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

// getCellPid returns the pid of the cell being operated on.
func getCellPid() (pid int, err error) {
	// TODO: call isCellRunning before continuing to make sure we don't read
	// a random pid
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

// isCellRunning checks to see if the cell that is being operated on is
// currently running.
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

// getCellUid returns the user id and group id of the cell currently being
// operated on.
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

// spawnCell starts the cell that is currently being operated on, and detatches
// it.
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
