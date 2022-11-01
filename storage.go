package virtualbox

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

// StorageControllers list of a virtual machine storage controllers.
type StorageControllers []StorageController

// StorageController represents a virtualized storage controller.
type StorageController struct {
	Name        string
	SysBus      SystemBus
	Ports       uint // SATA port count 1--30
	Chipset     StorageControllerChipset
	HostIOCache bool
	Bootable    bool

	Devices []StorageMedium
}

// SystemBus represents the system bus of a storage controller.
type SystemBus string

const (
	// SysBusIDE when the storage controller provides an IDE bus.
	SysBusIDE = SystemBus("ide")
	// SysBusSATA when the storage controller provides a SATA bus.
	SysBusSATA = SystemBus("sata")
	// SysBusSCSI when the storage controller provides an SCSI bus.
	SysBusSCSI = SystemBus("scsi")
	// SysBusFloppy when the storage controller provides access to Floppy drives.
	SysBusFloppy = SystemBus("floppy")
	// SysBusSAS when the storage controller provides an SAS bus.
	SysBusSAS = SystemBus("sas")
	// SysBusUSB when the storage controller provides an USB bus.
	SysBusUSB = SystemBus("usb")
	// SysBusPCI when the storage controller provides an PCIe bus.
	SysBusPCI = SystemBus("pcie")
	// SysBusVirtioSCSI when the storage controller provides storage access through virtio.
	SysBusVirtioSCSI = SystemBus("virtio")
)

// StorageControllerChipset represents the hardware of a storage controller.
type StorageControllerChipset string

const (
	// CtrlLSILogic when the storage controller emulates LSILogic hardware.
	CtrlLSILogic = StorageControllerChipset("LSILogic")
	// CtrlLSILogicSAS when the storage controller emulates LSILogicSAS hardware.
	CtrlLSILogicSAS = StorageControllerChipset("LSILogicSAS")
	// CtrlBusLogic when the storage controller emulates BusLogic hardware.
	CtrlBusLogic = StorageControllerChipset("BusLogic")
	// CtrlIntelAHCI when the storage controller emulates IntelAHCI hardware.
	CtrlIntelAHCI = StorageControllerChipset("IntelAHCI")
	// CtrlPIIX3 when the storage controller emulates PIIX3 hardware.
	CtrlPIIX3 = StorageControllerChipset("PIIX3")
	// CtrlPIIX4 when the storage controller emulates PIIX4 hardware.
	CtrlPIIX4 = StorageControllerChipset("PIIX4")
	// CtrlICH6 when the storage controller emulates ICH6 hardware.
	CtrlICH6 = StorageControllerChipset("ICH6")
	// CtrlI82078 when the storage controller emulates I82078 hardware.
	CtrlI82078 = StorageControllerChipset("I82078")
	// CtlrUSB when the storage controller emulates USB hardware.
	CtlrUSB = StorageControllerChipset("USB")
	// CtlrNVMe when the storage controller emulates NVMe hardware.
	CtlrNVMe = StorageControllerChipset("NVMe")
	// CtrlVirtioSCSI when the storage controller is based on VirtIO.
	CtrlVirtioSCSI = StorageControllerChipset("VirtIO")
)

// StorageMedium represents the storage medium attached to a storage controller.
type StorageMedium struct {
	Port      uint
	Device    uint
	DriveType DriveType
	Medium    string // none|emptydrive|<filename|host:<drive>|iscsi
	UUID      string
}

// DriveType represents the hardware type of a drive.
type DriveType string

const (
	// DriveDVD when the drive is a DVD reader/writer.
	DriveDVD = DriveType("dvddrive")
	// DriveHDD when the drive is a hard disk or SSD.
	DriveHDD = DriveType("hdd")
	// DriveFDD when the drive is a floppy.
	DriveFDD = DriveType("fdd")
)

// UUIDOrMedium return this storagemedium UUID if available otherwise its medium
func (sm StorageMedium) UUIDOrMedium() string {
	if sm.UUID == "" {
		return sm.Medium
	}
	return sm.UUID
}

// IsNone return true if the medium is not an actual storage medium, false otherwise.
func (sm StorageMedium) IsNone() bool {
	return sm.UUID == "" && sm.Medium == "none"
}

// CloneHD virtual harddrive
func CloneHD(input, output string) error {
	return Manage().run("clonehd", input, output)
}

func findStorageControllerByIndex(
	name string, iStr string, vmPropMap map[string]string,
) (*StorageController, error) {
	// storagecontrollername6="USB"
	// storagecontrollertype6="USB"
	// storagecontrollerinstance6="0"
	// storagecontrollermaxportcount6="8"
	// storagecontrollerportcount6="8"
	// storagecontrollerbootable6="on"
	scType := vmPropMap["storagecontrollertype"+iStr]
	//maxPortCount := vmPropMap["storagecontrollermaxportcount"+iStr]
	portCountStr := vmPropMap["storagecontrollerportcount"+iStr]
	portCount, err := strconv.Atoi(portCountStr)
	if err != nil {
		return nil, errors.Wrapf(
			err, "could not convert portCount(%s) from string to integer", portCountStr)
	}
	bootableStr := vmPropMap["storagecontrollerbootable"+iStr]
	bus, chipSet, err := vmInfogStrorageControllerTypeToBusAndChipset(scType)
	if err != nil {
		return nil, err
	}
	media, err := findStorageMedia(name, bus, portCount, vmPropMap)
	if err != nil {
		return nil, err
	}
	sc := StorageController{
		Name:    name,
		SysBus:  bus,
		Chipset: chipSet,
		Ports:   uint(portCount),
		//VM info does not return IO cache
		HostIOCache: false,
		Bootable:    bootableStr == "on",
		Devices:     *media,
	}
	return &sc, nil
}

func findStorageMedia(
	name string, bus SystemBus, portCount int, vmPropMap map[string]string,
) (*[]StorageMedium, error) {
	maxDevicePerPort := maxDevicePerPort(bus)
	media := make([]StorageMedium, 0, portCount*2)
	for p := 0; p < portCount; p++ {
		for d := 0; d < maxDevicePerPort; d++ {
			//"SATA-0-0"="/media/bigstorage/worker2.vdi"
			//"SATA-ImageUUID-0-0"="8c80c269-8569-4c90-b745-bac723810dab"
			indexSuffix := "-" + strconv.Itoa(p) + "-" + strconv.Itoa(d)
			medium, ok := vmPropMap[name+indexSuffix]
			if !ok {
				continue
			}
			uuid := vmPropMap[name+"-ImageUUID"+indexSuffix]
			sm := StorageMedium{
				Device:    uint(d),
				DriveType: "",
				Medium:    medium,
				Port:      uint(p),
				UUID:      uuid,
			}
			if !sm.IsNone() {
				media = append(media, sm)
			}
		}
	}

	media = append(make([]StorageMedium, 0, len(media)), media...)
	return &media, nil
}

func maxDevicePerPort(bus SystemBus) int {
	switch bus {
	case SysBusIDE, SysBusFloppy:
		return 2
	default:
		return 1
	}
}

func vmInfogStrorageControllerTypeToBusAndChipset(scType string) (SystemBus, StorageControllerChipset, error) {
	switch scType {
	case "IntelAhci":
		return SysBusSATA, CtrlIntelAHCI, nil
	case "LsiLogic":
		return SysBusSCSI, CtrlLSILogic, nil
	case "BusLogic":
		return SysBusSCSI, CtrlBusLogic, nil
	case "PIIX3":
		return SysBusIDE, CtrlPIIX3, nil
	case "PIIX4":
		return SysBusIDE, CtrlPIIX4, nil
	case "ICH6":
		return SysBusIDE, CtrlICH6, nil
	case "I82078":
		return SysBusFloppy, CtrlI82078, nil
	case "LsiLogicSas":
		return SysBusSAS, CtrlLSILogicSAS, nil
	case "USB":
		return SysBusUSB, CtlrUSB, nil
	case "NVMe":
		return SysBusPCI, CtlrNVMe, nil
	case "VirtioSCSI":
		return SysBusVirtioSCSI, CtrlVirtioSCSI, nil
	case "unknown":
		// VBoxManage showvminfo does not return actual config
		// (Bug? in virtual box code)
		// storagecontrollername5="NVMe"
		// storagecontrollertype5="unknown"
		// storagecontrollername7="VirtIO"
		// storagecontrollertype7="unknown"
		return SystemBus(scType), StorageControllerChipset(scType), nil
	default:
		return "", "", fmt.Errorf("controller type from vminfo not supported yet:%s", scType)
	}
}

// NewStorageControllersFromProps creates a new StorageControllers from a VM Info Map.
func NewStorageControllersFromProps(vmPropMap map[string]string) (*StorageControllers, error) {
	ctrls := make(StorageControllers, 0, 8)
	for i := 0; i < 8; i++ {
		iStr := strconv.Itoa(i)
		name, ok := vmPropMap["storagecontrollername"+iStr]
		if !ok {
			continue
		}
		sc, err := findStorageControllerByIndex(name, iStr, vmPropMap)
		if err != nil {
			return nil, err
		}
		ctrls = append(ctrls, *sc)
	}
	return &ctrls, nil
}

// DeviceMedia return all non empty medium of devices attachec to the storage controllers.
func (scs StorageControllers) DeviceMedia() []string {
	media := make([]string, 0, len(scs)*4)
	for _, sc := range scs {
		for _, d := range sc.Devices {
			medium := d.Medium
			if medium != "" {
				media = append(media, medium)
			}
		}
	}
	return media
}
