package main

import (
        "os"
        "fmt"
        "errors"
        "strconv"
        "os/exec"
        "os/user"
        "syscall"
        "io/ioutil"
)

func main () {
        uid := os.Getuid()
        if uid != 0 {
                fmt.Println("ERR this utility must be run as root")
                os.Exit(1)
        }
        ParseArgs()
}

func doStart () {
        if isCellRunning() {
                fmt.Println("(i)", options.cell, "is already running")
                os.Exit(1)
        }

        pid, err := spawnCell()
        if err != nil {
                fmt.Println("ERR could not start", options.cell + ":", err)
                os.Exit(1)
        }

        err = ioutil.WriteFile(options.pidfile, []byte(strconv.Itoa(pid)), 0644)
        if err != nil {
                fmt.Println (
                        "ERR could not write pidfile of",
                        options.cell + ":", err)
                os.Exit(1)
        }

        fmt.Println(pid)
}

func doStop () {
        pid, err := getCellPid()
        if err != nil {
                fmt.Println (
                        "ERR could not read pidfile of",
                        options.cell + ":", err)
                os.Exit(1)
        }

        process, err := os.FindProcess(pid)
        err = process.Kill()
        if err != nil {
                fmt.Println("ERR could not kill", options.cell + ":", err)
                os.Exit(1)
        }
}

func doRestart () {
        doStop()
        doStart()
}

func getCellPid () (pid int, err error) {
        content, err := ioutil.ReadFile(options.pidfile)
        if err != nil { return 0, err }
        pid, err = strconv.Atoi(string(content))
        if err != nil { return 0, err }
        return pid, nil
}

func isCellRunning () (running bool) {
        pid, err := getCellPid()
        if err != nil { return false }
        _, err = os.FindProcess(pid)
        if err == nil { return false }
        return true
}

func getCellUid () (uid uint32, gid uint32, err error) {
        user, err := user.Lookup(options.cell)
        if err != nil { return 0, 0, err }

        puid, err := strconv.Atoi(user.Uid)
        if err != nil { return 0, 0, err }
        pgid, err := strconv.Atoi(user.Gid)
        if err != nil { return 0, 0, err }
        return uint32(puid), uint32(pgid), nil
}

func spawnCell () (pid int, err error) {
        uid, gid, err := getCellUid()
        if err != nil { return 0, errors.New("user does not exist") }
        
        var cred = &syscall.Credential {
                Uid:         uid,
                Gid:         gid,
                Groups:      []uint32{},
                NoSetGroups: false,
        }
        
        // the Noctty flag is used to detach the process from parent tty
        var sysproc = &syscall.SysProcAttr {
                Credential: cred,
                Noctty:     true,
        }
        var attr = os.ProcAttr {
                Dir: ".",
                Env: os.Environ(),
                Files: []*os.File {
                        os.Stdin,
                        nil,
                        nil,
                },
                Sys: sysproc,
        }

        path, err := exec.LookPath(options.cell)
        if err != nil { return 0, errors.New("executable does not exist") }
        
        process, err := os.StartProcess(path, []string { path }, &attr)
        if err != nil { return 0, err }
        
        pid = process.Pid
        // process.Release actually detatches the process and init adopts it.
        return pid, process.Release();
}
