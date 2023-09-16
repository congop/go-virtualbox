package virtualbox

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/pkg/errors"
)

var (
	manage Command
)

var (
	reVMNameUUID      = regexp.MustCompile(`"(.+)" {([0-9a-f-]+)}`)
	reVMInfoLine      = regexp.MustCompile(`(?:"(.+)"|(.+))=(?:"(.*)"|(.*))`)
	reColonLine       = regexp.MustCompile(`(.+):\s+(.*)`)
	reMachineNotFound = regexp.MustCompile(`Could not find a registered machine named '(.+)'`)
	// matches VBoxManage: error: Could not find a registered machine with UUID {f0e5424d-77d7-45c4-b5bb-9aadc379cdb0}
	reMachineNotFoundByUuid = regexp.MustCompile(`Could not find a registered machine with UUID {.+}`)
)

// Manage returns the Command to run VBoxManage/VBoxControl.
func Manage() Command {
	if manage != nil {
		return manage
	}

	sudoer, err := isSudoer()
	if err != nil {
		Debug("Error getting sudoer status: '%v'", err)
	}

	if vbprog, err := LookupVBoxProgram("VBoxManage"); err == nil {
		manage = command{program: vbprog, sudoer: sudoer, guest: false}
	} else if vbprog, err := LookupVBoxProgram("VBoxControl"); err == nil {
		manage = command{program: vbprog, sudoer: sudoer, guest: true}
	} else {
		// Did not find a VirtualBox management command
		manage = command{program: "false", sudoer: false, guest: false}
	}
	Debug("manage: '%+v'", manage)
	return manage
}

// LookupVBoxProgram searches for an executable with the given name.
//
// On Windows: If environment variable VBOX_INSTALL_PATH exists will return ${VBOX_INSTALL_PATH}/vbprogName.exe,
// or look in the default install directory c:\\Program Files\Oracle\VirtualBox or use exec.LookPath to resolve the command
//
// # On Non Windows(linux, ...) exe.LookPath is used
//
// @param vbprooName the name of the virtual box executable (without extension in Windows and .exe is assumed or otherwise exec.LookPath used PATHTEXT and appropriate defaults)
func LookupVBoxProgram(vbprogName string) (string, error) {

	if runtime.GOOS == osWindows {
		if p := os.Getenv("VBOX_INSTALL_PATH"); p != "" {
			return filepath.Join(p, vbprogName+".exe"), nil
		} else {
			vbprog := filepath.Join("C:\\", "Program Files", "Oracle", "VirtualBox", vbprogName+".exe")
			switch _, err := os.Stat(vbprog); {
			case err == nil:
				// may not be a regular file or link pointing to a regular file, but lets not care for now
				// as it will be soon revealed if used
				return vbprog, nil
			case err != nil && !os.IsNotExist(err):
				return "", errors.Wrapf(err,
					"LookupVBoxProgram -- fail to os-stat Program Files executable location candidate:"+
						"\n\tvprogNameBase=%s \n\tvbprog=%s \n\terr=%v",
					vbprogName, vbprog, err)
			default:
				return lookupVBoxProgramByExecLookPath(vbprogName)
			}
		}
	}

	return lookupVBoxProgramByExecLookPath(vbprogName)
}

func lookupVBoxProgramByExecLookPath(vbprogName string) (string, error) {
	progPath, err := exec.LookPath(vbprogName)
	if err != nil {
		if !errors.Is(err, exec.ErrDot) {
			return "", errors.Wrapf(err,
				"LookupVBoxProgram -- fail to exec.Lookpath:"+
					"\n\tvprogNameBase=%s \n\terr=%v",
				vbprogName, err)
		}
		progPathAbs, err := filepath.Abs(progPath)
		if err != nil {
			return "", errors.Wrapf(err,
				"LookupVBoxProgram -- fail to get absolute path of program executable:"+
					"\n\tvprogNameBase=%s \n\tprogPath=%s \n\terr=%v",
				vbprogName, progPath, err)
		}
		return progPathAbs, nil
	}

	return progPath, nil
}

func isSudoer() (bool, error) {
	me, err := user.Current()
	if err != nil {
		return false, err
	}
	Debug("User: '%+v'", me)
	if groupIDs, err := me.GroupIds(); runtime.GOOS == "linux" {
		if err != nil {
			return false, err
		}
		Debug("groupIDs: '%+v'", groupIDs)
		for _, groupID := range groupIDs {
			group, err := user.LookupGroupId(groupID)
			if err != nil {
				return false, err
			}
			Debug("group: '%+v'", group)
			if group.Name == "sudo" {
				return true, nil
			}
		}
	}
	return false, nil
}
