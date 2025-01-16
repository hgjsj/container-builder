package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func must(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func test() {
	fmt.Println(syscall.Getpid())
	cmd := exec.Command("sh")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Unshareflags: syscall.CLONE_NEWNET,
	}
	must(cmd.Run())
}

func run() {
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// it is like soft copy. only copy value, but not underlying value of each element
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
		// it is like hard copy. copy value of all element and sub element
		// need totally seperate mount namespace due to mount namespace is implict share with others even if clone a new one
		Unshareflags: syscall.CLONE_NEWNS,

		//this is for user namespace, map current user in ns to user out of ns
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	must(cmd.Run())
}

func child() {
	cgoup()
	chroot()
	syscall.Sethostname([]byte("container"))
	// mount proc to current namespace /proc directory, in this case it only show proceess running in this namespace,
	// otherwise all proceesses which are running in parant namespace show up
	syscall.Mount("proc", "/proc", "proc", 0, "")
	//must(syscall.Unmount("/mnt/iso1", 0))
	cmd := exec.Command(os.Args[2], os.Args[3:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())

}

func chroot() {
	//change root path to new directory which is export from busybox container
	syscall.Chroot("/opt/project/rootfs")
	//switch to root path
	os.Chdir("/")

}

func cgoup() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "sutton"), 0755)
	os.WriteFile(filepath.Join(pids, "sutton/pids.max"), []byte("10"), 0700)
	os.WriteFile(filepath.Join(pids, "sutton/notify_on_release"), []byte("1"), 0700)
	os.WriteFile(filepath.Join(pids, "sutton/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700)
}

func main() {
	if len(os.Args) <= 1 {
		panic("You have to enter command you want to run")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	case "test":
		test()
	default:
		panic("Error")
	}
}
