package virtualbox

import "github.com/pkg/errors"

// Version return the version. E.g. 6.1.34r150636.
// format: <major>.<minor>.<patch>r<revision>
func Version() (string, error) {
	stdout, stderr, err := Manage().runOutErr("--version")
	if err != nil {
		return "", errors.Wrapf(err,
			"fail to get virtualbox version:\n\terr(%T)=%v, \n\tstdout=%s, "+
				"\n\tstderr=%s",
			err, err, stdout, stderr)
	}
	return stdout, nil
}
