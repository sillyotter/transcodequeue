package main

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <IOKit/pwr_mgt/IOPMLib.h>

// kIOPMAssertionTypeNoDisplaySleep prevents display sleep,
// kIOPMAssertionTypeNoIdleSleep prevents idle sleep

// reasonForActivity is a descriptive string used by the system whenever it needs
// to tell the user why the system is not sleeping. For example,
// "Mail Compacting Mailboxes" would be a useful string.

long PreventSleep()
{
	// NOTE: IOPMAssertionCreateWithName limits the string to 128 characters.
	CFStringRef reasonForActivity= CFSTR("Encoding");

	IOPMAssertionID assertionID;
	IOReturn success = IOPMAssertionCreateWithName(kIOPMAssertionTypeNoIdleSleep,
	                                    kIOPMAssertionLevelOn, reasonForActivity, &assertionID);
	if (success == kIOReturnSuccess)
	{
		return (int)assertionID;
	}
	return -1;
}

void AllowSleep(long assert)
{
	IOPMAssertionID assertionID = assert;
	IOPMAssertionRelease(assertionID);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"runtime"
	"syscall"
)

func createPidFile(filename string) (*os.File, error) {

	file, err := os.OpenFile(filename,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_SYNC,
		syscall.S_IRUSR|syscall.S_IWUSR)

	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.F_WRLCK)
	if err != nil {
		file.Close()
		return nil, err
	}

	_, err = file.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

func removePidFile(file *os.File) {

	err := syscall.Flock(int(file.Fd()), syscall.F_UNLCK)
	if err != nil {
		return
	}

	file.Close()
	os.Remove(file.Name())
}

type TranscodeQueue struct {
	Requests chan string
}

func NewTranscodeQueue() *TranscodeQueue {
	return &TranscodeQueue{
		Requests: make(chan string, 100),
	}
}

func (this *TranscodeQueue) DoTranscode(file string, reply *bool) error {
	log.Println("Asked to transcode:", file)
	this.Requests <- file
	*reply = true
	return nil
}

func runCommand(cmd *exec.Cmd, die <-chan os.Signal) (bool, error) {

	isDone := make(chan error)
	if err := cmd.Start(); err != nil {
		return false, err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			isDone <- err
		}
		isDone <- nil
	}()

	select {
	case err := <-isDone:
		return true, err
	case <-die:
		log.Println("Killing process")
		if err := cmd.Process.Kill(); err != nil {
			return false, err
		}
		return false, nil
	}

}

var wasCanceled error = errors.New("Was Canceled")

func performTranscode(infile, outfile string, isDone chan<- error, sigc <-chan os.Signal) {
	//infile = fmt.Sprintf("%q", infile)
	//outfile = fmt.Sprintf("%q", outfile)

	//transcodeCommand := exec.Command("/bin/sleep", "10")
	transcodeCommand := exec.Command(
		"/usr/local/bin/HandBrakeCLI",
		"-O", "-I",
		"-f", "mp4",
		"--encoder", "x264",
		"--x264-preset", "faster",
		"--x264-tune", "film",
		"--h264-profile", "auto",
		"--h264-level", "auto",
		"--quality", "20",
		"--large-file",
		"--aencoder", "ca_aac,copy:ac3",
		"-B", "160",
		"--mixdown", "dpl2",
		"--aname", "English",
		"--loose-anamorphic",
		"--decomb",
		"--modulus", "2",
		"-i", infile,
		"-o", outfile)

	//transferCommand := exec.Command("/bin/sleep", "5")
	transferCommand := exec.Command(
		"/usr/bin/scp", "-B", "-C", "-q", outfile,
		"guy@mediaserver.local:/srv/Media/Movies/0\\ -\\ Inbox/TV")

	log.Println("Performing transcode of", infile)
	log.Println("Writing file", outfile)

	done, err := runCommand(transcodeCommand, sigc)

	if err != nil {
		log.Println(err.Error())
		isDone <- err
		return
	}
	if !done {
		isDone <- wasCanceled
		return
	}

	log.Println("Done transcoding file")
	log.Println("Transfering file to server")

	done, err = runCommand(transferCommand, sigc)

	if err == nil && done {
		os.Remove(infile)
		os.Remove(outfile)
		// leave commented out until im sure i understand the scp problem
	} else {
		if err != nil {
			log.Println(err.Error())
			isDone <- err

		} else {
			isDone <- wasCanceled
		}
		return
	}

	log.Println("Done transfering file to server")
	isDone <- nil
}

func requestTranscode(infile, unixSocket string) (reply bool, err error) {

	client, err := rpc.Dial("unix", unixSocket)
	if err != nil {
		return
	}
	err = client.Call("TranscodeQueue.DoTranscode", infile, &reply)
	return
}

func createTranscodeServer(pidfile *os.File, unixSocket string) error {

	sigc := make(chan os.Signal, 10)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer signal.Stop(sigc)

	q := NewTranscodeQueue()
	server := rpc.NewServer()
	server.Register(q)

	l, err := net.Listen("unix", unixSocket)
	if err != nil {
		return err
	}

	defer l.Close()

	go server.Accept(l)

	isDone := make(chan error)
	rxp, err := regexp.Compile(`(.*)(\.mpg)$`)

	if err != nil {
		return err
	}

	q.Requests <- os.Args[1]

	for {
		select {
		case req := <-q.Requests:

			outfile := rxp.ReplaceAllString(req, "$1.m4v")
			go performTranscode(req, outfile, isDone, sigc)

			select {
			case e := <-isDone:
				if e != nil {
					return e
				}
			}
		default:
			return nil
		}
	}
}

func main() {

	sleepId := C.PreventSleep()
	defer C.AllowSleep(sleepId)

	if len(os.Args) != 2 {
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Println("Starting...")

	toTranscode := os.Args[1]
	pidFileName := path.Join(os.TempDir(), "transcodequeue.pid")
	unixSocket := path.Join(os.TempDir(), "transcodequeue.sock")

	pidFile, err := createPidFile(pidFileName)

	if err != nil {
		log.Println("Sending server request to transcode", toTranscode)

		reply, err := requestTranscode(toTranscode, unixSocket)
		if err != nil {
			log.Panicln(err.Error())
		}

		if reply {
			log.Println("Successfully requested transcode")
		}
	} else {
		log.Println("Creating Server")

		defer removePidFile(pidFile)

		err = createTranscodeServer(pidFile, unixSocket)
		if err != nil {
			log.Println(err.Error())
		}
	}

	log.Println("Finished")
}
