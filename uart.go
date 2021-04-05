package virtualbox

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// UARTKey key used to indentify vm uart config [uart1|2|3|4].
type UARTKey string

const (
	// UART1 key for uart 1
	UART1 = UARTKey("uart1")

	// UART2 key for uart 2
	UART2 = UARTKey("uart2")

	// UART3 key for uart 3
	UART3 = UARTKey("uart3")

	// UART4 key for uart 4
	UART4 = UARTKey("uart4")
)

// KnownUARTKeys is tha list off all known uart key
func KnownUARTKeys() []UARTKey {
	return []UARTKey{UART1, UART2, UART3, UART4}
}

// IsSupportedKey returns true if the key is supported, false otherwise.
func (key UARTKey) IsSupportedKey() bool {
	switch key {
	case UART1, UART2, UART3, UART4:
		return true
	default:
		return false
	}
}

// ToRank returns rank corresponding to the uart key
func (key UARTKey) ToRank() uint8 {
	switch key {
	case UART1:
		return 1
	case UART2:
		return 2
	case UART3:
		return 3
	case UART4:
		return 4
	default:
		panic(fmt.Errorf("unknown uartkey: %s (Known are %v)", key, KnownUARTKeys()))
	}
}

// UARTOffFromKey return a new UART holding this kex
func (key UARTKey) UARTOffFromKey() UART {
	return UART{Key: key}
}

// UARTKeyFromRank return uart key given rank
func UARTKeyFromRank(rank uint8) (UARTKey, error) {
	switch rank {
	case 1:
		return UART1, nil
	case 2:
		return UART2, nil
	case 3:
		return UART3, nil
	case 4:
		return UART4, nil
	default:
		return "", fmt.Errorf("rank (%d) not allowed. Valid values are 1,2,3,4", rank)
	}
}

//BasicSerialComConfig config holds serial port io base a.k.a port and IRQ
type BasicSerialComConfig struct {
	Port uint
	IRQ  uint
}

// PortAsIoBaseHexString return port as IO BAse Hey String e.g. 0X3E8
func (cc BasicSerialComConfig) PortAsIoBaseHexString() string {
	return fmt.Sprintf("0x%04x", cc.Port)
}

// IRQAsString return IRQ as string
func (cc BasicSerialComConfig) IRQAsString() string {
	return fmt.Sprintf("%d", cc.IRQ)
}

// COM1 IRQ=4 Port=0x3F8
func COM1() BasicSerialComConfig { return BasicSerialComConfig{Port: 0x3F8, IRQ: 4} }

// COM2 IRQ=3 Port=0x2F8
func COM2() BasicSerialComConfig { return BasicSerialComConfig{Port: 0x2F8, IRQ: 3} }

// COM3 IRQ=4 Port=0x3E8
func COM3() BasicSerialComConfig { return BasicSerialComConfig{Port: 0x3E8, IRQ: 4} }

// COM4 IRQ=3 Port=0x2E8
func COM4() BasicSerialComConfig { return BasicSerialComConfig{Port: 0x2E8, IRQ: 3} }

// initFromVMInfoValue parse string representation 0x03f8,4 format <IO Base>,<IRQ> and init with the found values
func (cc *BasicSerialComConfig) initFromVMInfoValue(vmInfoValue string) error {
	splits := strings.Split(vmInfoValue, ",")
	//TODO check what about if found unexpected format
	if len(splits) != 2 {
		return fmt.Errorf(
			`bad format for the vm info value of uart<1-N>
				expected: <IO Base hex>,<IRQ>
				but got : %s`,
			vmInfoValue)
	}
	//eskiping the 0x before parsing
	ioBase, errIOBase := strconv.ParseUint(splits[0][2:], 16, 64)
	irq, errIRQ := strconv.ParseUint(splits[1], 10, 32)
	if errIOBase != nil || errIRQ != nil {
		return fmt.Errorf(
			`could not parse IRQ, <IO Base>: %s | splits: %s:
						error for IO Base: %v
						error for IRQ= %v`, vmInfoValue, splits, errIOBase, errIRQ)
	}
	cc.Port = uint(ioBase)
	cc.IRQ = uint(irq)
	return nil
}

// User-defined IRQ=<> Port=<>

// odifyvm
//   [--uart<1-N> off|<I/O base> <IRQ>]

//   [--uartmode<1-N> disconnected|
//                    server <pipe>|
//                    client <pipe>|
//                    tcpserver <port>|
//                    tcpclient <hostname:port>|
//                    file <file>|
//                    <devicename>]
//   [--uarttype<1-N> 16450|16550A|16750]

// UARTType [--uarttype<1-N> 16450|16550A|16750]
type UARTType string

const (
	// UARTT16450 The most basic emulated UART which doesn't support FIFO operation.
	UARTT16450 = UARTType("16450")

	// UARTT16550A The successor of the 16450 UART introducing a 16 byte FIFO to reduce operational overhead.
	UARTT16550A = UARTType("16550A")

	//UARTT16750 This UART developed by Texas Instruments introduced a 64 byte FIFO and hardware flow control.
	UARTT16750 = UARTType("16750")

	// UARTTDefault default type 16550A
	UARTTDefault = UARTT16550A
)

// UARTTypeAllSupported return all UART types that are supported by VirtualBox
func UARTTypeAllSupported() []UARTType {
	return []UARTType{UARTT16450, UARTT16550A, UARTT16750}
}

// UARTTypeFromStringIfSupported return a supported UARTType represented by the given string or and error
func UARTTypeFromStringIfSupported(uartTypeStr string) (uartType UARTType, err error) {
	uartType = UARTType(uartTypeStr)
	if !uartType.IsSupportedUARTType() {
		err = fmt.Errorf("uart type(%s) not supported; supported are:%s", uartTypeStr, UARTTypeAllSupported())
	}
	return
}

// IsSupportedUARTType whether this UART Type is supported by VirtualBox
func (uartType UARTType) IsSupportedUARTType() bool {
	switch uartType {
	case UARTT16450, UARTT16550A, UARTT16750:
		return true
	default:
		return false
	}
}

// UARTMode uart mode specified using --uartmode<1-N>
type UARTMode string

const (
	// UARTModeServer server <pipe> -- Host Pipe: Configure Oracle VM VirtualBox to connect the virtual serial port to a software pipe on the host. (??RW folder must be specified)|
	UARTModeServer = UARTMode("server")

	// UARTModeClient client <pipe> -- Host Device: Connects the virtual serial port to a physical serial port on your host|
	UARTModeClient = UARTMode("client")
	// UARTModeTCPServer  tcpserver <port> -- Useful for forwarding serial traffic over TCP/IP, acting as a server|
	UARTModeTCPServer = UARTMode("tcpserver")

	// UARTModeTCPClient  tcpclient <hostname:port> -  it can act as a TCP client connecting to other servers|
	UARTModeTCPClient = UARTMode("tcpclient")

	// UARTModeFile   file <file>|
	UARTModeFile = UARTMode("file")

	// UARTModeDisconnected uartmode disconnected
	UARTModeDisconnected = UARTMode("disconnected")
)

// UARTModelAllSupported returns all supported uart modes.
func UARTModelAllSupported() []UARTMode {
	return []UARTMode{
		UARTModeServer, UARTModeClient,
		UARTModeTCPClient, UARTModeServer,
		UARTModeFile, UARTModeDisconnected,
	}
}

// UARTModeFromStringIfSupported returns a supported UARTMode given the string representation or an error
func UARTModeFromStringIfSupported(uartModeStr string) (UARTMode, error) {
	switch uartModeStr {
	case "server":
		return UARTModeServer, nil
	case "client":
		return UARTModeServer, nil
	case "tcpserver":
		return UARTModeTCPServer, nil
	case "tcpclient":
		return UARTModeTCPClient, nil
	case "file":
		return UARTModeFile, nil
	case "disconnected":
		return UARTModeDisconnected, nil
	default:
		return "", fmt.Errorf("unsupported uart mode[%s]; supported are: %s",
			uartModeStr, UARTModelAllSupported())

	}
}

//   [--uartmode<1-N> disconnected|
//                    server <pipe> -- Host Pipe: Configure Oracle VM VirtualBox to connect the virtual serial port to a software pipe on the host. (??RW folder must be specified)|
//                    client <pipe> -- Host Device: Connects the virtual serial port to a physical serial port on your host|
//                    tcpserver <port> -- Useful for forwarding serial traffic over TCP/IP, acting as a server|
//                    tcpclient <hostname:port> -  it can act as a TCP client connecting to other servers|
//                    file <file>|
//                    <devicename>]

// PortMode_Disconnected
// Virtual device is not attached to any real host device.

// PortMode_HostPipe
// Virtual device is attached to a host pipe.

// PortMode_HostDevice
// Virtual device is attached to a host device.

// PortMode_RawFile
// Virtual device is attached to a raw file.

// PortMode_TCP
// Virtual device is attached to a TCP socket.

// UART represents a virtualized uart device, serial port.
type UART struct {
	ComConfig BasicSerialComConfig
	Type      UARTType
	Mode      UARTMode
	ModeData  string
	Key       UARTKey
}

// UARTs container of all UART of a VM
type UARTs []UART

// WithoutUARTHavingStateOff remove the UART having the <off> state from this UARTs and return it
func (uarts *UARTs) WithoutUARTHavingStateOff() *UARTs {
	n := 0
	for _, uart := range *uarts {
		if uart.IsOff() {
			continue
		}
		(*uarts)[n] = uart
		n++

	}
	*uarts = (*uarts)[:n]
	return uarts
}

// ModifyVMCommandParameters return the cmd parameters given the state of the UARTs.
// The return has the following format for each available UART
//   [--uart<1-N> off|<I/O base> <IRQ>]
//   [--uartmode<1-N> disconnected|
//   server <pipe>|
//   client <pipe>|
//   tcpserver <port>|
//   tcpclient <hostname:port>|
//   file <file>|
//   <devicename>]
// [--uarttype<1-N> 16450|16550A|16750]
func (uarts UARTs) ModifyVMCommandParameters() ([]string, error) {
	cmdParams := make([]string, 0, 32)
	for _, uartn := range uarts {
		cmdParamsUARTn, err := uartn.commandParameters()
		if err != nil {
			return nil, err
		}
		cmdParams = append(cmdParams, cmdParamsUARTn...)
	}
	return cmdParams, nil
}

// IsOff true if off false otherwise
func (uart UART) IsOff() bool {
	return BasicSerialComConfig{} == uart.ComConfig
}

//[--uart<1-N> off|<I/O base> <IRQ>]

func (uart UART) validate() error {
	return nil
}

func (uart UART) commandParameters() ([]string, error) {
	if err := uart.validate(); err != nil {
		return nil, err
	}

	commandFuncs := []func() (cmdName string, cmdValue string){
		uart.commandParameterUartN,
		uart.commandParameterUARTModeN,
		uart.commandParameterUARTTypeN,
	}
	commands := make([]string, 0, len(commandFuncs))
	for _, commandFunc := range commandFuncs {
		if cmdName, cmdValue := commandFunc(); cmdName != "" {
			cmdValueSplitsBySpace := strings.Split(cmdValue, " ")
			commands = append(commands, cmdName)
			commands = append(commands, cmdValueSplitsBySpace...)
		}
	}
	return commands, nil
}

// uartNCommandParameter return an uart<1-N> parameter of this uart.
// format [--uart<1-N> off|<I/O base> <IRQ>]
func (uart UART) commandParameterUartN() (cmdName string, cmdValue string) {
	if uart.IsOff() {
		return fmt.Sprintf("--uart%d", uart.Key.ToRank()), "off"
	}
	return fmt.Sprintf("--uart%d", uart.Key.ToRank()),
		fmt.Sprintf("0x%04x %d", uart.ComConfig.Port, uart.ComConfig.IRQ)
}

// commandParameterUARTModeN return uart mode command parameter [--uartmode<1-N> ...] or empty string for off uart
// The UART ist required to be valid, as no validity check is done hier.
// full format as:
// [--uartmode<1-N> disconnected|
//                server <pipe>|
//                client <pipe>|
//                tcpserver <port>|
//                tcpclient <hostname:port>|
//                file <file>|
//                <devicename>]
func (uart UART) commandParameterUARTModeN() (cmdName string, cmdValue string) {
	if uart.IsOff() {
		return "", ""
	}
	switch uart.Mode {
	case UARTModeDisconnected:
		return fmt.Sprintf("--uartmode%d", uart.Key.ToRank()), string(UARTModeDisconnected)
	default:
		return fmt.Sprintf("--uartmode%d", uart.Key.ToRank()), string(uart.Mode) + " " + uart.ModeData

	}
}

// returns rhe type command parameter  [--uarttype<1-N> 16450|16550A|16750] or empty for <OFF UART>
// The UART is required to be valid as no validaticion check is done here
func (uart UART) commandParameterUARTTypeN() (cmdName string, cmdValue string) {
	if uart.IsOff() {
		return "", ""
	}
	return fmt.Sprintf("--uarttype%d", uart.Key.ToRank()), string(uart.Type)
}

// NewUARTsAllOff return URATs containing uart<1..4> which are off
func NewUARTsAllOff() *UARTs {
	// uarts, _ := NewUARTs(nil)
	// return uarts
	return &UARTs{
		UART1.UARTOffFromKey(), UART2.UARTOffFromKey(),
		UART3.UARTOffFromKey(), UART4.UARTOffFromKey(),
	}
}

// NewUARTsFromUARTMap return UARTs initialized with the given UART in the map of as UART off.
func NewUARTsFromUARTMap(uartMap map[UARTKey]UART) (*UARTs, error) {
	uarts := *NewUARTsAllOff()
	multierr := &multierror.Error{}
	for key, uart := range uartMap {
		if !key.IsSupportedKey() {
			multierr = multierror.Append(
				multierr,
				fmt.Errorf("cannot use UART with unsupported key: %s, supported are:%s", key, KnownUARTKeys()))
			continue
		}
		uarts[key.ToRank()-1] = uart
	}
	return &uarts, multierr.ErrorOrNil()

}

// NewUARTs creates UARTs from a VM Info Map.
func NewUARTs(vmPropMap map[string]string) (*UARTs, error) {
	uarts := make(UARTs, 0, 4)
	for i := 1; i <= 4; i++ {

		// uart1="0x03f8,4"
		//   [--uart<1-N> off|<I/O base> <IRQ>]
		key, err := UARTKeyFromRank(uint8(i))
		if nil != err {
			return nil, err
		}

		uart := UART{Key: key}

		if uartI, ok := vmPropMap[string(uart.Key)]; ok {
			if uartI != "off" {
				err = uart.ComConfig.initFromVMInfoValue(uartI)
				if nil != err {
					return nil, err
				}

				err = uart.initUARTModeFromVMInfoMap(vmPropMap)
				if nil != err {
					return nil, err
				}

				err = uart.initUARTTypeFromVMInfoMap(vmPropMap)
				if nil != err {
					return nil, err
				}
			}
		}
		uarts = append(uarts, uart)

	}
	return &uarts, nil
}

//   [--uartmode<1-N> disconnected|
//                    server <pipe> -- Host Pipe: Configure Oracle VM VirtualBox to connect the virtual serial port to a software pipe on the host. (??RW folder must be specified)|
//                    client <pipe> -- Host Device: Connects the virtual serial port to a physical serial port on your host|
//                    tcpserver <port> -- Useful for forwarding serial traffic over TCP/IP, acting as a server|
//                    tcpclient <hostname:port> -  it can act as a TCP client connecting to other servers|
//                    file <file>|
//                    <devicename>]

func (uart *UART) initUARTModeFromVMInfoMap(vmPropMap map[string]string) error {
	//uartmode1="file,/tmp/ubuntu-focal-1"
	//uartmode2="tcpclient,127.0.0.1:5555"
	// uartmode3="tcpserver,6666"
	// !!! Host Device / checked (forced) connect to existing pip/socket | path/address=/dev/ttyS0
	//     uartmode4="/dev/ttyS0"
	// uartmode4="disconnected"
	rank := uart.Key.ToRank()
	modeName := fmt.Sprintf("uartmode%d", rank)
	modeStrValue, ok := vmPropMap[modeName]
	if !ok {
		return nil
	}

	if string(UARTModeDisconnected) == modeStrValue {
		uart.Mode = UARTModeDisconnected
		return nil
	}
	modeStartValueSplits := strings.Split(modeStrValue, ",")
	if len(modeStartValueSplits) == 2 {
		mode := UARTMode(modeStartValueSplits[0])
		switch mode {
		case UARTModeClient, UARTModeDisconnected, UARTModeFile, UARTModeServer, UARTModeTCPClient, UARTModeTCPServer:
			uart.Mode = mode
			uart.ModeData = modeStartValueSplits[1]
			return nil
		default:
			return fmt.Errorf("unsupported mode: %s, original vm info value:%s", modeStartValueSplits[0], modeStrValue)
		}

	}
	return fmt.Errorf(
		"unsupported format (expetec is: unsupported |<mode>,<modevalue>) for uart mode: modename=%s modevalue=%s, uartkey=%s",
		modeName, modeStrValue, uart.Key)
}

// initUARTTypeFromVMInfoMap parse info map value of type ([--uarttype<1-N> 16450|16550A|16750]) and init uart type.
func (uart *UART) initUARTTypeFromVMInfoMap(vmPropMap map[string]string) error {
	rank := uart.Key.ToRank()
	typeName := fmt.Sprintf("uarttype%d", rank)
	typeValue, ok := vmPropMap[typeName]
	if !ok {
		return nil
	}
	// TODO check or not? ist it better to read and than check using a validate? but you looses context?
	uartType := UARTType(typeValue)
	if !uartType.IsSupportedUARTType() {
		return fmt.Errorf("UARTType not supported: %s, supported are: %s", uartType, UARTTypeAllSupported())
	}
	uart.Type = uartType
	return nil
}

// NewUART return a new ART given strings representing key, type, port, irq, mode and mode-data
func NewUART(keyStr, typeStr, portStr, irqStr, modeStr, modeDataStr string) (*UART, error) {
	key := UARTKey(keyStr)
	if !UARTKey(keyStr).IsSupportedKey() {
		return nil, fmt.Errorf(
			"key(original string value:%s, UARTKey:%s) is not supported; supported are %s",
			keyStr, key, KnownUARTKeys())
	}

	uart := UART{Key: key}
	multierr := &multierror.Error{}
	var err error
	uart.Mode, err = UARTModeFromStringIfSupported(modeStr)
	multierr = multierror.Append(multierr, err)
	uart.ModeData = modeDataStr

	if portStr != "" && irqStr != "" {

		var port uint
		_, errPort := fmt.Sscanf(portStr, "0x%x", &port)
		multierr = multierror.Append(multierr, errPort)
		//var irq uint64
		irq, errIRQ := strconv.ParseUint(irqStr, 10, 64) //() fmt.Sscanf(irqStr, "%d", &irq)
		multierr = multierror.Append(multierr, errIRQ)
		uart.ComConfig.IRQ = uint(irq)
		uart.ComConfig.Port = port

	} else if (portStr != "" && irqStr == "") || (portStr == "" && irqStr != "") {
		multierr = multierror.Append(multierr,
			fmt.Errorf("port and irq must all be empty or all non empty: port=%s, irq=%s", portStr, irqStr))
	}

	uart.Type, err = UARTTypeFromStringIfSupported(typeStr)
	multierr = multierror.Append(multierr, err)

	return &uart, multierr.ErrorOrNil()
}
