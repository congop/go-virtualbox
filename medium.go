package virtualbox

import "github.com/pkg/errors"

func UnregisterDisk(idOrFn string) error {
	stdout, stderr, err := Manage().runOutErr("closemedium", "disk", idOrFn)

	if err != nil {
		return errors.Wrapf(err, "fail to unregister disk: disk=%q, err=%q, our=%q", idOrFn, stderr, stdout)
	}
	return nil
}

func UnregisterDvd(idOrFn string) error {
	stdout, stderr, err := Manage().runOutErr("closemedium", "dvd", idOrFn)

	if err != nil {
		return errors.Wrapf(err, "fail to unregister disk: disk=%q, err=%q, our=%q", idOrFn, stderr, stdout)
	}
	return nil
}
