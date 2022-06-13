package virtualbox

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// MachineState stores the last retrieved VM state.
type MachineState string

const (
	// Poweroff is a MachineState value.
	Poweroff = MachineState("poweroff")
	// Running is a MachineState value.
	Running = MachineState("running")
	// Paused is a MachineState value.
	Paused = MachineState("paused")
	// Saved is a MachineState value.
	Saved = MachineState("saved")
	// Aborted is a MachineState value.
	Aborted = MachineState("aborted")
)

// Flag is an active VM configuration toggle
type Flag int

// Flag names in lowercases to be consistent with VBoxManage options.
const (
	ACPI       Flag = 1 << iota // --apic on|off: Enables and disables I/O APIC. With I/O APIC, operating systems can use more than 16 interrupt requests (IRQs) thus avoiding IRQ sharing for improved reliability. This setting is enabled by default.
	IOAPIC                      //--acpi on|off and --ioapic on|off: Determines whether the VM has ACPI and I/O APIC support.
	RTCUSEUTC                   // --rtcuseutc on|off: Sets the real-time clock (RTC) to operate in UTC time
	CPUHOTPLUG                  // -cpuhotplug on|off: Enables CPU hot-plugging. When enabled, virtual CPUs can be added to and removed from a virtual machine while it is running.
	PAE                         // --pae on|off: Enables and disables PAE
	LONGMODE                    // --longmode on|off: Enables and disables long mode.
	SYNTHCPU
	HPET             // --hpet on|off: Enables and disables a High Precision Event Timer (HPET) which can replace the legacy system timers. This is turned off by default. Note that Windows supports a HPET only from Vista onwards.
	HWVIRTEX         // --hwvirtex on|off: Enables and disables the use of hardware virtualization extensions, such as Intel VT-x or AMD-V, in the processor of your host system
	TRIPLEFAULTRESET // --triplefaultreset on|off: Enables resetting of the guest instead of triggering a Guru Meditation. Some guests raise a triple fault to reset the CPU so sometimes this is desired behavior. Works only for non-SMP guests.
	NESTEDPAGING     // --nestedpaging on|off: If hardware virtualization is enabled, this additional setting enables or disables the use of the nested paging feature in the processor of your host system
	LARGEPAGES       // --largepages on|off: If hardware virtualization and nested paging are enabled, for Intel VT-x only, an additional performance improvement of up to 5% can be obtained by enabling this setting. This causes the hypervisor to use large pages to reduce TLB use and overhead.
	VTXVPID          // -vtxvpid on|off: If hardware virtualization is enabled, for Intel VT-x only, this additional setting enables or disables the use of the tagged TLB (VPID) feature in the processor of your host system
	VTXUX            // --vtxux on|off: If hardware virtualization is enabled, for Intel VT-x only, this setting enables or disables the use of the unrestricted guest mode feature for executing your guest.
	ACCELERATE3D     // --accelerate3d on|off: If the Guest Additions are installed, this setting enables or disables hardware 3D acceleration.
	NESTED_HW_VIRT   //--nested-hw-virt on|off: If hardware virtualization is enabled, this setting enables or disables passthrough of hardware virtualization features to the guest.
	X2APIC           // --x2apic on|off: Enables and disables CPU x2APIC support. CPU x2APIC support helps operating systems run more efficiently on high core count configurations, and optimizes interrupt distribution in virtualized environments. This setting is enabled by default. Disable this setting when using host or guest operating systems that are incompatible with x2APIC support.
)

// Convert bool to "on"/"off"
func bool2string(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// Get tests if flag is set. Return "on" or "off".
func (f Flag) Get(o Flag) string {
	return bool2string(f&o == o)
}

// Machine information.
type Machine struct {
	Name               string
	UUID               string
	State              MachineState
	CPUs               uint
	Memory             uint // main memory (in MB)
	VRAM               uint // video memory (in MB)
	CfgFile            string
	BaseFolder         string
	OSType             string
	Flag               Flag
	BootOrder          []string // max 4 slots, each in {none|floppy|dvd|disk|net}
	NICs               []NIC
	UARTs              UARTs
	StorageControllers StorageControllers
}

// New creates a new machine.
func New() *Machine {
	return &Machine{
		BootOrder: make([]string, 0, 4),
		NICs:      make([]NIC, 0, 4),
		UARTs:     *NewUARTsAllOff(),
	}
}

// Refresh reloads the machine information.
func (m *Machine) Refresh() error {
	id := m.Name
	if id == "" {
		id = m.UUID
	}
	mm, err := GetMachine(id)
	if err != nil {
		return err
	}
	*m = *mm
	return nil
}

// Start starts the machine.
func (m *Machine) Start() error {
	switch m.State {
	case Paused:
		return Manage().run("controlvm", m.Name, "resume")
	case Poweroff, Saved, Aborted:
		return Manage().run("startvm", m.Name, "--type", "headless")
	}
	return nil
}

// DisconnectSerialPort sets given serial port to disconnected.
func (m *Machine) DisconnectSerialPort(portNumber int) error {
	return Manage().run("modifyvm", m.Name, fmt.Sprintf("--uartmode%d", portNumber), "disconnected")
}

// Save suspends the machine and saves its state to disk.
func (m *Machine) Save() error {
	switch m.State {
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	case Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "savestate")
}

// Pause pauses the execution of the machine.
func (m *Machine) Pause() error {
	switch m.State {
	case Paused, Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "pause")
}

// Stop gracefully stops the machine.
func (m *Machine) Stop() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	}

	for m.State != Poweroff { // busy wait until the machine is stopped
		if err := Manage().run("controlvm", m.Name, "acpipowerbutton"); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		if err := m.Refresh(); err != nil {
			return err
		}
	}
	return nil
}

// Poweroff forcefully stops the machine. State is lost and might corrupt the disk image.
func (m *Machine) Poweroff() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	}
	return Manage().run("controlvm", m.Name, "poweroff")
}

// Restart gracefully restarts the machine.
func (m *Machine) Restart() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start()
}

// Reset forcefully restarts the machine. State is lost and might corrupt the disk image.
func (m *Machine) Reset() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	return Manage().run("controlvm", m.Name, "reset")
}

// Delete deletes the machine and associated disk images.
func (m *Machine) Delete() error {
	if err := m.Poweroff(); err != nil {
		return err
	}
	return Manage().run("unregistervm", m.Name, "--delete")
}

func (m *Machine) Unregister() error {
	if err := m.Poweroff(); err != nil {
		return err
	}
	return Manage().run("unregistervm", m.Name)
}

var mutex sync.Mutex

func vminfoAsPropMap(vmInfo io.Reader) (map[string]string, error) {
	/* Read all VM info into a map */
	propMap := make(map[string]string)
	s := bufio.NewScanner(vmInfo)
	for s.Scan() {
		res := reVMInfoLine.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		key := res[1]
		if key == "" {
			key = res[2]
		}
		val := res[3]
		if val == "" {
			val = res[4]
		}
		propMap[key] = val
	}
	if err := s.Err(); err != nil {
		return nil, errors.Wrap(err, "error parsing vminfo into map")
	}
	return propMap, nil
}

// GetMachine finds a machine by its name or UUID.
func GetMachine(id string) (*Machine, error) {
	/* There is a strage behavior where running multiple instances of
	'VBoxManage showvminfo' on same VM simultaneously can return an error of
	'object is not ready (E_ACCESSDENIED)', so we sequential the operation with a mutex.
	Note if you are running multiple process of go-virtualbox or 'showvminfo'
	in the command line side by side, this not gonna work. */
	mutex.Lock()
	stdout, stderr, err := Manage().runOutErr("showvminfo", id, "--machinereadable")
	mutex.Unlock()
	if err != nil {
		if reMachineNotFound.FindString(stderr) != "" {
			return nil, ErrMachineNotExist
		}
		return nil, errors.Wrapf(err, "Error with showvminfo for id=%s, \nstderr:%s",
			id, stderr)
	}

	/* Read all VM info into a map */
	propMap, err := vminfoAsPropMap(strings.NewReader(stdout))
	if err != nil {
		return nil, err
	}

	/* Extract basic info */
	m := New()
	m.Name = propMap["name"]
	m.UUID = propMap["UUID"]
	m.State = MachineState(propMap["VMState"])
	n, err := strconv.ParseUint(propMap["memory"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.Memory = uint(n)
	n, err = strconv.ParseUint(propMap["cpus"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.CPUs = uint(n)
	n, err = strconv.ParseUint(propMap["vram"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.VRAM = uint(n)
	m.CfgFile = propMap["CfgFile"]
	m.BaseFolder = filepath.Dir(m.CfgFile)

	/* Extract NIC info */
	for i := 1; i <= 4; i++ {
		var nic NIC
		nicType, ok := propMap[fmt.Sprintf("nic%d", i)]
		if !ok || nicType == "none" {
			break
		}
		nic.Network = NICNetwork(nicType)
		nic.Hardware = NICHardware(propMap[fmt.Sprintf("nictype%d", i)])
		if nic.Hardware == "" {
			return nil, fmt.Errorf("could not find corresponding 'nictype%d'", i)
		}
		nic.MacAddr = propMap[fmt.Sprintf("macaddress%d", i)]
		if nic.MacAddr == "" {
			return nil, fmt.Errorf("could not find corresponding 'macaddress%d'", i)
		}
		if nic.Network == NICNetHostonly {
			nic.HostInterface = propMap[fmt.Sprintf("hostonlyadapter%d", i)]
		} else if nic.Network == NICNetBridged {
			nic.HostInterface = propMap[fmt.Sprintf("bridgeadapter%d", i)]
		} else if nic.Network == NICNetNAT {
			// TODO set with( --natnet1 "default") result in (natnet1="nat") what should we map some where
			nic.NetworkName = propMap[fmt.Sprintf("natnet%d", i)]
		} else if nic.Network == NICNetNATNetwork {
			nic.NetworkName = propMap[fmt.Sprintf("nat-network%d", i)]
		}
		m.NICs = append(m.NICs, nic)
	}

	pUARTs, errNewUART := NewUARTs(propMap)
	if errNewUART != nil {
		return nil, errors.Wrap(errNewUART, "Error reading UARTs data vfrom vm info")
	}
	m.UARTs = *pUARTs

	scs, err := NewStorageControllersFromProps(propMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read storage controllers")
	}
	m.StorageControllers = *scs

	// if err := s.Err(); err != nil {
	// 	return nil, err
	// }
	return m, nil
}

// ListMachines lists all registered machines.
func ListMachines() ([]*Machine, error) {
	out, err := Manage().runOut("list", "vms")
	if err != nil {
		return nil, err
	}
	ms := []*Machine{}
	s := bufio.NewScanner(strings.NewReader(out))
	for s.Scan() {
		res := reVMNameUUID.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		m, err := GetMachine(res[1])
		if err != nil {
			// Sometimes a VM is listed but not available, so we need to handle this.
			if err == ErrMachineNotExist {
				continue
			} else {
				return nil, err
			}
		}
		ms = append(ms, m)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

// CreateMachine creates a new machine. If basefolder is empty, use default.
func CreateMachine(uuid, name, basefolder string) (*Machine, error) {
	if name == "" || uuid == "" {
		return nil, fmt.Errorf("machine name(=%s) or uuid(=%s) is empty", name, uuid)
	}

	// Check if a machine with the given name already exists.
	ms, err := ListMachines()
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		if m.Name == name {
			return nil, ErrMachineExist
		}
	}

	// Create and register the machine.
	args := []string{"createvm", "--uuid", uuid, "--name", name, "--register"}
	if basefolder != "" {
		args = append(args, "--basefolder", basefolder)
	}
	if err = Manage().run(args...); err != nil {
		return nil, err
	}

	m, err := GetMachine(name)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Modify changes the settings of the machine.
func (m *Machine) Modify(override ...CmdArg) error {
	cmdArgs := CmdArgs{}
	args := []string{"modifyvm", m.Name}
	cmdArgs.Append("--firmware", "bios")
	cmdArgs.Append("--bioslogofadein", "off")
	cmdArgs.Append("--bioslogofadeout", "off")
	cmdArgs.Append("--bioslogodisplaytime", "0")
	cmdArgs.Append("--biosbootmenu", "disabled")

	cmdArgs.Append("--ostype", m.OSType)
	cmdArgs.Append("--cpus", fmt.Sprintf("%d", m.CPUs))
	cmdArgs.Append("--memory", fmt.Sprintf("%d", m.Memory))
	cmdArgs.Append("--vram", fmt.Sprintf("%d", m.VRAM))

	cmdArgs.Append("--acpi", m.Flag.Get(ACPI))
	cmdArgs.Append("--ioapic", m.Flag.Get(IOAPIC))
	cmdArgs.Append("--rtcuseutc", m.Flag.Get(RTCUSEUTC))
	cmdArgs.Append("--cpuhotplug", m.Flag.Get(CPUHOTPLUG))
	cmdArgs.Append("--pae", m.Flag.Get(PAE))
	cmdArgs.Append("--longmode", m.Flag.Get(LONGMODE))
	//TODO check cause error VBoxManage: error: Unknown option: --synthcpu
	//"--synthcpu", m.Flag.Get(SYNTHCPU),
	cmdArgs.Append("--hpet", m.Flag.Get(HPET))
	cmdArgs.Append("--hwvirtex", m.Flag.Get(HWVIRTEX))
	cmdArgs.Append("--triplefaultreset", m.Flag.Get(TRIPLEFAULTRESET))
	cmdArgs.Append("--nestedpaging", m.Flag.Get(NESTEDPAGING))
	cmdArgs.Append("--largepages", m.Flag.Get(LARGEPAGES))
	cmdArgs.Append("--vtxvpid", m.Flag.Get(VTXVPID))
	cmdArgs.Append("--vtxux", m.Flag.Get(VTXUX))
	cmdArgs.Append("--accelerate3d", m.Flag.Get(ACCELERATE3D))
	cmdArgs.Append("--nested-hw-virt", m.Flag.Get(NESTED_HW_VIRT))

	for i, dev := range m.BootOrder {
		if i > 3 {
			break // Only four slots `--boot{1,2,3,4}`. Ignore the rest.
		}
		cmdArgs.Append(fmt.Sprintf("--boot%d", i+1), dev)
	}

	for i, nic := range m.NICs {
		n := i + 1
		if err := appendNicParams(n, nic, &cmdArgs); err != nil {
			return err
		}
	}

	uartsCmdArgs, err := m.UARTs.ModifyVMCmdArgs()
	if err != nil {
		return errors.Wrap(err, "Error getting UARTs Modify VM Command Parameters")
	}
	cmdArgs.AppendCmdArgs(uartsCmdArgs...)
	cmdArgs.AppendOverride(override...)

	args = append(args, cmdArgs.Args()...)

	if stdout, stderr, err := Manage().runOutErr(args...); err != nil {
		return errors.Wrapf(err,
			"Error executing <VBoxManage modifyvm ...> \nARGS:%s\n STDOUTs=%s\nSTDERR=%s\n",
			args, stdout, stderr)
	}

	return m.Refresh()
}

// AddNATPF adds a NAT port forarding rule to the n-th NIC with the given name.
func (m *Machine) AddNATPF(n int, name string, rule PFRule) error {
	return Manage().run("controlvm", m.Name, fmt.Sprintf("natpf%d", n),
		fmt.Sprintf("%s,%s", name, rule.Format()))
}

// DelNATPF deletes the NAT port forwarding rule with the given name from the n-th NIC.
func (m *Machine) DelNATPF(n int, name string) error {
	return Manage().run("controlvm", m.Name, fmt.Sprintf("natpf%d", n), "delete", name)
}

func appendNicParams(n int, nic NIC, cmdArgs *CmdArgs) error {
	cmdArgs.Append(fmt.Sprintf("--nic%d", n), string(nic.Network))
	cmdArgs.Append(fmt.Sprintf("--nictype%d", n), string(nic.Hardware))
	cmdArgs.Append(fmt.Sprintf("--cableconnected%d", n), "on")
	if nic.MacAddr != "" {
		cmdArgs.Append(fmt.Sprintf("--macaddress%d", n), nic.MacAddr)
	}
	if nic.Network == NICNetHostonly {
		cmdArgs.Append(fmt.Sprintf("--hostonlyadapter%d", n), nic.HostInterface)
	} else if nic.Network == NICNetBridged {
		cmdArgs.Append(fmt.Sprintf("--bridgeadapter%d", n), nic.HostInterface)
	} else if nic.Network == NICNetNAT {
		if nic.NetworkName != "" {
			//[--natnet<1-N> <network>|default]
			cmdArgs.Append(fmt.Sprintf("--natnet%d", n), nic.NetworkName)
		} else {
			cmdArgs.Append(fmt.Sprintf("--natnet%d", n), "default")
		}
	} else if nic.Network == NICNetNATNetwork {
		if nic.NetworkName != "" {
			//[--nat-network<1-N> <network name>]
			cmdArgs.Append(fmt.Sprintf("--nat-network%d", n), nic.NetworkName)
		}
	} else if nic.Network == NICNetInternal {
		if nic.NetworkName != "" {
			//[--intnet<1-N> <network name>]
			cmdArgs.Append(fmt.Sprintf("--intnet%d", n), nic.NetworkName)
		}
	}
	return nil
}

// SetNIC set the n-th NIC.
func (m *Machine) SetNIC(rank int, nic NIC) error {
	cmdArgs := CmdArgs{}
	if err := appendNicParams(rank, nic, &cmdArgs); err != nil {
		return err
	}

	args := []string{"modifyvm", m.Name}
	args = append(args, cmdArgs.Args()...)
	Trace("SetNic -- VBoxManage : args=%v", args)
	return Manage().run(args...)
}

// AddStorageCtl adds a storage controller with the given name.
func (m *Machine) AddStorageCtl(name string, ctl StorageController) error {
	args := []string{"storagectl", m.Name, "--name", name}
	if ctl.SysBus != "" {
		args = append(args, "--add", string(ctl.SysBus))
	}
	if ctl.Ports > 0 {
		args = append(args, "--portcount", fmt.Sprintf("%d", ctl.Ports))
	}
	if ctl.Chipset != "" {
		args = append(args, "--controller", string(ctl.Chipset))
	}
	args = append(args, "--hostiocache", bool2string(ctl.HostIOCache))
	args = append(args, "--bootable", bool2string(ctl.Bootable))
	return Manage().run(args...)
}

// DelStorageCtl deletes the storage controller with the given name.
func (m *Machine) DelStorageCtl(name string) error {
	return Manage().run("storagectl", m.Name, "--name", name, "--remove")
}

// AttachStorage attaches a storage medium to the named storage controller.
func (m *Machine) AttachStorage(ctlName string, medium StorageMedium) error {
	return Manage().run("storageattach", m.Name, "--storagectl", ctlName,
		"--port", fmt.Sprintf("%d", medium.Port),
		"--device", fmt.Sprintf("%d", medium.Device),
		"--type", string(medium.DriveType),
		"--medium", medium.UUIDOrMedium(),
	)
}

// DetachStorage detaches a storage medium from the named storage controller.
func (m *Machine) DetachStorage(ctlName string, medium StorageMedium) error {
	return Manage().run("storageattach", m.Name, "--storagectl", ctlName,
		"--port", fmt.Sprintf("%d", medium.Port),
		"--device", fmt.Sprintf("%d", medium.Device),
		"--type", string(medium.DriveType),
		"--medium", "none",
	)
}

// SetExtraData attaches custom string to the VM.
func (m *Machine) SetExtraData(key, val string) error {
	return Manage().run("setextradata", m.Name, key, val)
}

// GetExtraData retrieves custom string from the VM.
func (m *Machine) GetExtraData(key string) (*string, error) {
	value, err := Manage().runOut("getextradata", m.Name, key)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	/* 'getextradata get' returns 0 even when the key is not found,
	so we need to check stdout for this case */
	if strings.HasPrefix(value, "No value set") {
		return nil, nil
	}
	trimmed := strings.TrimPrefix(value, "Value: ")
	return &trimmed, nil
}

// DeleteExtraData removes custom string from the VM.
func (m *Machine) DeleteExtraData(key string) error {
	return Manage().run("setextradata", m.Name, key)
}

// CloneMachine clones the given machine name into a new one.
func CloneMachine(baseImageName string, newImageName string, register bool) error {
	if register {
		return Manage().run("clonevm", baseImageName, "--name", newImageName, "--register")
	}
	return Manage().run("clonevm", baseImageName, "--name", newImageName)
}
