package scp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blacknon/go-scplib"
	"github.com/blacknon/lssh/conf"
	sshcmd "github.com/blacknon/lssh/ssh"
)

type RunInfoScp struct {
	CopyFromType   string
	CopyFromPath   string
	CopyFromServer []string
	CopyToType     string
	CopyToPath     string
	CopyToServer   []string
	CopyData       *bytes.Buffer
	ServrNameMax   int
	PermissionFlag bool
	ConConfig      conf.Config
}

func (r *RunInfoScp) forScp(mode string) {
	finished := make(chan bool)
	x := 1

	targetServer := []string{}
	if mode == "push" {
		targetServer = r.CopyToServer
	} else {
		targetServer = r.CopyFromServer
	}
	for _, v := range targetServer {
		//y := x
		conServer := v
		go func() {
			sh := new(sshcmd.ConInfoCmd)
			cp := new(scplib.SCPClient)
			sh.Addr = r.ConConfig.Server[conServer].Addr
			sh.User = r.ConConfig.Server[conServer].User
			sh.Port = "22"
			if r.ConConfig.Server[conServer].Port != "" {
				sh.Port = r.ConConfig.Server[conServer].Port
			}
			sh.Pass = ""
			if r.ConConfig.Server[conServer].Pass != "" {
				sh.Pass = r.ConConfig.Server[conServer].Pass
			}
			sh.KeyPath = ""
			if r.ConConfig.Server[conServer].Key != "" {
				sh.KeyPath = r.ConConfig.Server[conServer].Key
			}

			session, err := sh.CreateSession()
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot connect %v:%v, %v \n", conServer, sh.Port, err)
				finished <- true
				return
			}
			cp.Permission = r.PermissionFlag
			cp.Session = session

			switch mode {
			case "push":
				// scp push
				if r.CopyToType == r.CopyFromType {
					err := cp.PutData(r.CopyData, r.CopyToPath)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to run: "+err.Error())
					}
				} else {
					err := cp.PutFile(r.CopyFromPath, r.CopyToPath)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to run: "+err.Error())
					}
					fmt.Println(conServer + " is exit.")
				}

			case "pull":
				toPath := r.CopyToPath

				// if multi server connect => path = /path/to/Dir/<ServerName>/Base
				if len(targetServer) > 1 {
					toDir := filepath.Dir(r.CopyToPath)
					toBase := filepath.Base(r.CopyToPath)
					serverDir := toDir + "/" + conServer

					err = os.Mkdir(serverDir, os.FileMode(uint32(0755)))
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to run: "+err.Error())
					}

					if toDir != toBase {
						toPath = serverDir + "/" + toBase
					} else {
						toPath = serverDir + "/"
					}
				}

				// scp pull
				if r.CopyToType == r.CopyFromType {
					//buf := new(bytes.Buffer)
					r.CopyData, err = cp.GetData(r.CopyFromPath)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to run: "+err.Error())
					}
				} else {
					err := cp.GetFile(r.CopyFromPath, toPath)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Failed to run: "+err.Error())
					}
					fmt.Println(conServer + " is exit.")
				}
			}

			finished <- true
		}()
		x++
	}

	for i := 1; i <= len(targetServer); i++ {
		<-finished
	}
}

func (r *RunInfoScp) ScpRun() {
	// get connect server name max length
	for _, conServerName := range append(r.CopyFromServer, r.CopyToServer...) {
		if r.ServrNameMax < len(conServerName) {
			r.ServrNameMax = len(conServerName)
		}
	}

	switch {
	case r.CopyFromType == "remote" && r.CopyToType == "remote":
		r.forScp("pull")
		r.forScp("push")
	case r.CopyFromType == "remote" && r.CopyToType == "local":
		r.forScp("pull")
	case r.CopyFromType == "local" && r.CopyToType == "remote":
		r.forScp("push")
	}
}