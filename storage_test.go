package virtualbox

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestNewStorageControllersFromProps(t *testing.T) {
	// log.SetOutput()
	type args struct {
		vmPropMap map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    *StorageControllers
		wantErr bool
	}{
		{
			name: "Should be able to recognize all storage types",
			args: args{
				vmPropMap: vmPropMapFromLocalRes(
					t, "testdata/vboxmanage-showvminfo-all-storagecontroller.pout"),
			},
			wantErr: false,
			want: &StorageControllers{
				{
					Name: "IDE", SysBus: "ide", Ports: 0x2, Chipset: "PIIX4", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{
						{Port: 0x1, Device: 0x0, DriveType: "", Medium: "emptydrive", UUID: ""},
					},
				},
				{
					Name: "SATA", SysBus: "sata", Ports: 0x1, Chipset: "IntelAHCI", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{
						{Port: 0x0, Device: 0x0, DriveType: "", Medium: "/media/bigstorage/worker2.vdi", UUID: "8c80c269-8569-4c90-b745-bac723810dab"},
					},
				},
				{
					Name: "Floppy", SysBus: "floppy", Ports: 0x1, Chipset: "I82078", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
				{
					Name: "LsiLogic", SysBus: "scsi", Ports: 0x10, Chipset: "LSILogic", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
				{
					Name: "LsiLogic SAS", SysBus: "sas", Ports: 0x1, Chipset: "LSILogicSAS", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
				{
					Name: "NVMe", SysBus: "unknown", Ports: 0x1, Chipset: "unknown", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
				{
					Name: "USB", SysBus: "usb", Ports: 0x8, Chipset: "USB", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
				{
					Name: "VirtIO", SysBus: "unknown", Ports: 0x1, Chipset: "unknown", HostIOCache: false, Bootable: true,
					Devices: []StorageMedium{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStorageControllersFromProps(tt.args.vmPropMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStorageControllersFromProps() \nerror = %v, \nwantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStorageControllersFromProps() \ngot = %#v, \nwant= %#v", got, tt.want)
			}
		})
	}
}

func vmPropMapFromLocalRes(t *testing.T, resRelPath string) (vmPropMap map[string]string) {
	resbytes, err := ioutil.ReadFile(resRelPath)
	if err != nil {
		t.Fatalf("could not load local res as vmPropMap: resRelPath=%s, err=%v", resRelPath, err)
		return nil
	}
	vmPropMap, err = vminfoAsPropMap(bytes.NewBuffer(resbytes))
	if err != nil {
		t.Fatalf("could not parse resource %s as vm info map", resRelPath)
	}
	return vmPropMap
}
